// This content has been partially derived from Tekton
// https://github.com/tektoncd/pipeline

package entrypoint

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
	"google.golang.org/protobuf/proto"
)

const (
	// FIXME This should be in a common location
	MetadataAPIURLEnvName = "METADATA_API_URL"
)

type RealRunner struct {
	signals chan os.Signal

	TimeoutLong  time.Duration
	TimeoutShort time.Duration
}

var _ Runner = (*RealRunner)(nil)

// FIXME Determine how to handle, log, and report on errors
// Many errors that might occur should not necessarily abort the basic command processing
// Logging these errors should potentially not occur either, as it potentially polutes the expected command logs
// with potential internal information
// Logging command outputs should default more cleanly to the standard streams
// Additionally, integration tests will not (currently) have access to the Metadata API and would cause failures
func (rr *RealRunner) Run(args ...string) error {
	if len(args) == 0 {
		return nil
	}

	if err := updatePath(); err != nil {
		log.Println(err)
	}

	mu, err := getMetadataAPIURL()
	if err != nil {
		log.Println(err)
	}

	if mu != nil {
		if err := rr.getEnvironmentVariables(mu); err != nil {
			log.Println(err)
		}

		if err := rr.validateSchemas(mu); err != nil {
			log.Println(err)
		}
	}

	name, args := args[0], args[1:]

	// Receive system signals on "rr.signals"
	if rr.signals == nil {
		rr.signals = make(chan os.Signal, 1)
	}
	defer close(rr.signals)
	signal.Notify(rr.signals)
	defer signal.Reset()

	var logOut *plspb.LogCreateResponse
	var logErr *plspb.LogCreateResponse
	if mu != nil {
		logOut, err = rr.postLog(mu, &plspb.LogCreateRequest{
			Name: "stdout",
		})
		if err != nil {
			log.Println(err)
		}

		logErr, err = rr.postLog(mu, &plspb.LogCreateRequest{
			Name: "stderr",
		})
		if err != nil {
			log.Println(err)
		}
	}

	cmd := exec.Command(name, args...)

	// dedicated PID group used to forward signals to
	// main process and all children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	doneOut := make(chan bool)
	doneErr := make(chan bool)

	scannerOut := bufio.NewScanner(stdoutPipe)
	scannerErr := bufio.NewScanner(stderrPipe)

	// Start defined command
	if err := cmd.Start(); err != nil {
		return err
	}

	// Goroutine for signals forwarding
	go func() {
		for s := range rr.signals {
			// Forward signal to main process and all children
			if s != syscall.SIGCHLD {
				_ = syscall.Kill(-cmd.Process.Pid, s.(syscall.Signal))
			}
		}
	}()

	go rr.scan(mu, scannerOut, os.Stdout, logOut, doneOut)
	go rr.scan(mu, scannerErr, os.Stderr, logErr, doneErr)

	<-doneOut
	<-doneErr

	// Wait for command to exit
	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func (rr *RealRunner) scan(mu *url.URL, scanner *bufio.Scanner, out *os.File, lcr *plspb.LogCreateResponse, done chan<- bool) {
	for scanner.Scan() {
		line := scanner.Text()

		if lcr != nil {
			message := &plspb.LogMessageAppendRequest{
				LogId:   lcr.GetLogId(),
				Payload: []byte(line),
			}

			if ts, err := ptypes.TimestampProto(time.Now().UTC()); err == nil {
				message.Timestamp = ts
			}

			_, err := rr.postLogMessage(mu, message)
			if err != nil {
				log.Println(err)
			}
		}

		out.WriteString(line)
		out.WriteString("\n")
	}
	done <- true
}

func updatePath() error {
	path := os.Getenv("PATH")
	return os.Setenv("PATH", path+":/var/lib/puppet/relay")
}

func getMetadataAPIURL() (*url.URL, error) {
	metadataAPIURL := os.Getenv(MetadataAPIURLEnvName)
	if metadataAPIURL == "" {
		return nil, nil
	}

	return url.Parse(metadataAPIURL)
}

func (rr *RealRunner) getEnvironmentVariables(mu *url.URL) error {
	ee := &url.URL{Path: "/environment"}

	req, err := http.NewRequest(http.MethodGet, mu.ResolveReference(ee).String(), nil)
	if err != nil {
		return err
	}

	resp, err := getResponse(req, rr.TimeoutLong, []retry.WaitOption{})
	if err != nil {
		return err
	}

	if resp != nil && resp.Body != nil {
		var r model.JSONResultEnvelope
		json.NewDecoder(resp.Body).Decode(&r)

		if r.Value.Data != nil {
			switch t := r.Value.Data.(type) {
			case map[string]interface{}:
				for name, value := range t {
					os.Setenv(name, fmt.Sprintf("%v", value))
				}
			}
		}
	}

	return nil
}

func (rr *RealRunner) postLog(mu *url.URL, request *plspb.LogCreateRequest) (*plspb.LogCreateResponse, error) {
	le := &url.URL{Path: "/logs"}

	buf, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, mu.ResolveReference(le).String(), bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := getResponse(req, rr.TimeoutLong, []retry.WaitOption{})
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := &plspb.LogCreateResponse{}
	err = proto.Unmarshal(body, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (rr *RealRunner) postLogMessage(mu *url.URL, request *plspb.LogMessageAppendRequest) (*plspb.LogMessageAppendResponse, error) {
	lme := &url.URL{Path: fmt.Sprintf("/logs/%s/messages", request.GetLogId())}

	buf, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, mu.ResolveReference(lme).String(), bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := getResponse(req, rr.TimeoutShort, []retry.WaitOption{})
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := &plspb.LogMessageAppendResponse{}
	err = proto.Unmarshal(body, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// validateSchemas calls validation endpoints on the metadata-api to validate
// step schemas.
//
// TODO: as part of our effort to catch early schema and documentation errors
// in core steps, we are just capturing errors to a logging service and fixing
// the step repos. This means a simple http call to the validate endpoint is
// fired off and the response is ignored.
//
// Once we have determined that things are stable, we will
// begin propagating these errors to the frontend and stopping the steps from
// running if they don't validate.
func (rr *RealRunner) validateSchemas(mu *url.URL) error {
	ve := &url.URL{Path: "/validate"}

	req, err := http.NewRequest(http.MethodPost, mu.ResolveReference(ve).String(), nil)
	if err != nil {
		return err
	}

	// We are ignoring the response for now because this endpoint just sends
	// all validation errors to the error capturing system.
	_, err = getResponse(req, rr.TimeoutLong, []retry.WaitOption{})
	if err != nil {
		return err
	}

	return nil
}

func getResponse(request *http.Request, timeout time.Duration, waitOptions []retry.WaitOption) (*http.Response, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var response *http.Response
	err := retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		var rerr error
		response, rerr = http.DefaultClient.Do(request)
		if rerr != nil {
			return false, rerr
		}

		if response != nil {
			// TODO Consider expanding to all 5xx (and possibly some 4xx) status codes
			switch response.StatusCode {
			case http.StatusInternalServerError, http.StatusBadGateway,
				http.StatusServiceUnavailable, http.StatusGatewayTimeout:
				return false, nil
			}
		}

		return true, nil
	}, waitOptions...)
	if err != nil {
		return nil, err
	}

	return response, nil
}
