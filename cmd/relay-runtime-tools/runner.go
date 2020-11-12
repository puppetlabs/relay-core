// This content has been partially derived from Tekton
// https://github.com/tektoncd/pipeline

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/puppetlabs/relay-core/pkg/entrypoint"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
)

type realRunner struct {
	signals chan os.Signal
}

var _ entrypoint.Runner = (*realRunner)(nil)

func (rr *realRunner) Run(args ...string) error {
	if len(args) == 0 {
		return nil
	}

	path := os.Getenv("PATH")
	os.Setenv("PATH", path+":/var/lib/puppet/relay")

	metadataAPIURL := os.Getenv("METADATA_API_URL")

	// FIXME Cannot abort due to errors here ...
	// Integration tests will not have access to the Metadata API and would cause failures
	mu, err := url.Parse(metadataAPIURL)
	if err != nil {
		log.Println(err)
	}

	if mu != nil {
		if err := getEnvironmentVariables(mu); err != nil {
			log.Println(err)
		}

		if err := validateSchemas(mu); err != nil {
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

	logOut, err := postLog(mu, &plspb.LogCreateRequest{
		Name: "stdout",
	})
	if err != nil {
		log.Println(err)
	}

	logErr, err := postLog(mu, &plspb.LogCreateRequest{
		Name: "stderr",
	})
	if err != nil {
		log.Println(err)
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

	go scan(mu, scannerOut, os.Stdout, logOut.GetLogId(), doneOut)
	go scan(mu, scannerErr, os.Stderr, logErr.GetLogId(), doneErr)

	<-doneOut
	<-doneErr

	// Wait for command to exit
	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func scan(mu *url.URL, scanner *bufio.Scanner, out *os.File, id string, done chan<- bool) {
	for scanner.Scan() {
		line := scanner.Text()

		postLogMessage(mu, &plspb.LogMessageAppendRequest{
			LogId:   id,
			Payload: []byte(line),
		})

		out.WriteString(line)
		out.WriteString("\n")
	}
	done <- true
}

func getEnvironmentVariables(mu *url.URL) error {
	ee := &url.URL{Path: "/environment"}

	req, err := http.NewRequest(http.MethodGet, mu.ResolveReference(ee).String(), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp != nil && resp.Body != nil {
		var r evaluate.JSONResultEnvelope
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

func postLog(mu *url.URL, request *plspb.LogCreateRequest) (*plspb.LogCreateResponse, error) {
	le := &url.URL{Path: "/logs"}

	buf, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, mu.ResolveReference(le).String(), bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	response := &plspb.LogCreateResponse{}
	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func postLogMessage(mu *url.URL, request *plspb.LogMessageAppendRequest) (*plspb.LogMessageAppendResponse, error) {
	lme := &url.URL{Path: "/logs/messages"}

	buf, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, mu.ResolveReference(lme).String(), bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	response := &plspb.LogMessageAppendResponse{}
	err = json.NewDecoder(resp.Body).Decode(response)
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
func validateSchemas(mu *url.URL) error {
	ve := &url.URL{Path: "/validate"}

	req, err := http.NewRequest(http.MethodPost, mu.ResolveReference(ve).String(), nil)
	if err != nil {
		return err
	}

	// We are ignoring the response for now because this endpoint just sends
	// all validation errors to the error capturing system.
	if _, err = http.DefaultClient.Do(req); err != nil {
		return err
	}

	return nil
}
