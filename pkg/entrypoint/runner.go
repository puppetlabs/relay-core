// This content has been partially derived from Tekton
// https://github.com/tektoncd/pipeline

package entrypoint

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	DefaultErrorExitCode = -1
	DefaultResultsPath   = "/tekton/results"
)

type RealRunner struct {
	signals chan os.Signal

	Config *Config
}

var _ Runner = (*RealRunner)(nil)

// FIXME Determine how to handle, log, and report on errors
// Many errors that might occur should not necessarily abort the basic command processing
// Logging these errors should potentially not occur either, as it adds internal information
// Logging command outputs should default more cleanly to the standard streams
func (rr *RealRunner) Run(args ...string) error {
	if len(args) == 0 {
		return nil
	}

	ctx := context.Background()

	// Receive system signals on "rr.signals"
	if rr.signals == nil {
		rr.signals = make(chan os.Signal, 1)
	}
	defer close(rr.signals)
	signal.Notify(rr.signals)
	defer signal.Reset()

	if err := updatePath(); err != nil {
		log.Println(err)
	}

	var err error
	var logOut *plspb.LogCreateResponse
	var logErr *plspb.LogCreateResponse

	name, args := args[0], args[1:]

	mu := rr.Config.MetadataAPIURL
	if mu != nil {
		if rr.Config.SecureLogging {
			logOut, err = rr.postLog(ctx, mu, &plspb.LogCreateRequest{
				Name: "stdout",
			})
			if err != nil {
				log.Println(err)
			}

			logErr, err = rr.postLog(ctx, mu, &plspb.LogCreateRequest{
				Name: "stderr",
			})
			if err != nil {
				log.Println(err)
			}
		}

		if err := rr.setStepInitTimer(ctx, mu); err != nil {
			log.Println(err)
		}
	}

	var cmd *exec.Cmd

	whenContext, cancel := context.WithCancel(ctx)

	// Goroutine for signals forwarding
	go func() {
		for s := range rr.signals {
			// Forward signal to main process and all children
			switch s {
			case syscall.SIGINT, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGTERM:
				cancel()

				if cmd != nil && cmd.Process != nil {
					_ = syscall.Kill(-cmd.Process.Pid, s.(syscall.Signal))
				}
			}
		}
	}()

	whenCondition := &model.ActionStatusWhenCondition{
		Timestamp:           time.Now().UTC(),
		WhenConditionStatus: model.WhenConditionStatusUnknown,
	}

	if mu != nil {
		_ = rr.putStatus(ctx, mu,
			&model.ActionStatus{
				WhenCondition: &model.ActionStatusWhenCondition{
					Timestamp:           time.Now().UTC(),
					WhenConditionStatus: model.WhenConditionStatusEvaluating,
				},
			})

		whenConditionStatus, err := rr.evaluateConditions(whenContext, mu)

		whenCondition = &model.ActionStatusWhenCondition{
			Timestamp:           time.Now().UTC(),
			WhenConditionStatus: whenConditionStatus,
		}

		_ = rr.putStatus(ctx, mu,
			&model.ActionStatus{
				WhenCondition: whenCondition,
			})
		if err != nil {
			return err
		} else if whenCondition == nil ||
			whenCondition.WhenConditionStatus != model.WhenConditionStatusSatisfied {
			return nil
		}
	}

	if mu != nil {
		if err := rr.getEnvironmentVariables(ctx, mu); err != nil {
			log.Println(err)
		}

		if name != path.Join(model.InputScriptMountPath, model.InputScriptName) {
			if err := rr.validateSchemas(ctx, mu); err != nil {
				log.Println(err)
			}
		}
	}

	cmd = exec.Command(name, args...)

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

	if whenContext.Err() != nil {
		rr.handleProcessState(ctx, mu, nil, whenCondition)

		return nil
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	go rr.scan(ctx, mu, scannerOut, os.Stdout, logOut, doneOut)
	go rr.scan(ctx, mu, scannerErr, os.Stderr, logErr, doneErr)

	<-doneOut
	<-doneErr

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			rr.handleProcessState(ctx, mu, exitErr.ProcessState, whenCondition)

			return nil
		}

		rr.handleProcessState(ctx, mu, nil, whenCondition)

		return err
	}

	rr.handleProcessState(ctx, mu, cmd.ProcessState, whenCondition)

	return nil
}

func (rr *RealRunner) scan(ctx context.Context, mu *url.URL, scanner *bufio.Scanner, out *os.File, lcr *plspb.LogCreateResponse, done chan<- bool) {
	for scanner.Scan() {
		line := scanner.Text()

		if mu != nil && lcr != nil {
			message := &plspb.LogMessageAppendRequest{
				LogId:     lcr.GetLogId(),
				Payload:   []byte(line),
				Timestamp: timestamppb.New(time.Now().UTC()),
			}

			if _, err := rr.postLogMessage(ctx, mu, message); err != nil {
				log.Println(err)
			}
		}

		_, _ = out.WriteString(line)
		_, _ = out.WriteString("\n")
	}
	done <- true
}

func updatePath() error {
	currentPath := os.Getenv("PATH")
	return os.Setenv("PATH", currentPath+":/var/lib/puppet/relay")
}

func (rr *RealRunner) handleProcessState(ctx context.Context, mu *url.URL, ps *os.ProcessState, wc *model.ActionStatusWhenCondition) {
	if mu != nil {
		exitCode := DefaultErrorExitCode
		if ps != nil {
			exitCode = ps.ExitCode()
		}
		_ = rr.putStatus(ctx, mu,
			&model.ActionStatus{
				ProcessState: &model.ActionStatusProcessState{
					ExitCode:  exitCode,
					Timestamp: time.Now().UTC(),
				},
				WhenCondition: wc,
			},
		)
	}
}

func (rr *RealRunner) evaluateConditions(ctx context.Context, mu *url.URL) (model.WhenConditionStatus, error) {
	ee := &url.URL{Path: "/conditions"}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, mu.ResolveReference(ee).String(), http.NoBody)
	if err != nil {
		return model.WhenConditionStatusFailure, err
	}

	// TODO This needs to be configurable
	contextWithTimeout, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	success := false
	err = retry.Wait(contextWithTimeout, func(ctx context.Context) (bool, error) {
		resp, err := rr.getResponse(ctx, req, []retry.WaitOption{})
		if err != nil {
			return retry.Done(err)
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var env api.GetConditionsResponseEnvelope
			if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
				if err == io.EOF {
					success = true
					return retry.Done(nil)
				}

				return retry.Done(err)
			}

			if !env.Resolved {
				return retry.Repeat(errors.New("conditions not resolved"))
			}

			success = env.Success
			return retry.Done(nil)
		default:
			return retry.Done(nil)
		}
	})
	if err != nil {
		return model.WhenConditionStatusFailure, err
	}

	if !success {
		return model.WhenConditionStatusNotSatisfied, nil
	}

	return model.WhenConditionStatusSatisfied, nil
}

func (rr *RealRunner) getEnvironmentVariables(ctx context.Context, mu *url.URL) error {
	ee := &url.URL{Path: "/environment"}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, mu.ResolveReference(ee).String(), http.NoBody)
	if err != nil {
		return err
	}

	resp, err := rr.getResponse(ctx, req, []retry.WaitOption{})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var r api.GetSpecResponseEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return err
	}

	if r.Value.Data != nil {
		switch t := r.Value.Data.(type) {
		case map[string]interface{}:
			for name, value := range t {
				os.Setenv(name, fmt.Sprintf("%v", value))
			}
		}
	}

	return nil
}

func (rr *RealRunner) postLog(ctx context.Context, mu *url.URL, request *plspb.LogCreateRequest) (*plspb.LogCreateResponse, error) {
	le := &url.URL{Path: "/logs"}

	buf, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, mu.ResolveReference(le).String(), bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := rr.getResponse(ctx, req, []retry.WaitOption{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.Header.Get("content-type") {
	case "application/octet-stream":
		response := &plspb.LogCreateResponse{}
		err = proto.Unmarshal(body, response)
		if err != nil {
			return nil, err
		}

		return response, nil
	}

	return nil, nil
}

func (rr *RealRunner) postLogMessage(ctx context.Context, mu *url.URL, request *plspb.LogMessageAppendRequest) (*plspb.LogMessageAppendResponse, error) {
	lme := &url.URL{Path: fmt.Sprintf("/logs/%s/messages", request.GetLogId())}

	buf, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, mu.ResolveReference(lme).String(), bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := rr.getResponse(ctx, req, []retry.WaitOption{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.Header.Get("content-type") {
	case "application/octet-stream":
		response := &plspb.LogMessageAppendResponse{}
		err = proto.Unmarshal(body, response)
		if err != nil {
			return nil, err
		}

		return response, nil
	}

	return nil, nil
}

func (rr *RealRunner) putStatus(ctx context.Context, mu *url.URL, status *model.ActionStatus) error {
	le := &url.URL{Path: "/status"}

	env := mapActionStatusRequest(status)

	buf, err := json.Marshal(env)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodPut, mu.ResolveReference(le).String(), bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	resp, err := rr.getResponse(ctx, req, []retry.WaitOption{})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func mapActionStatusRequest(as *model.ActionStatus) *api.PutActionStatusRequestEnvelope {
	env := &api.PutActionStatusRequestEnvelope{}

	if as.ProcessState != nil {
		env.ProcessState = &api.ActionStatusProcessState{
			ExitCode:  as.ProcessState.ExitCode,
			Timestamp: as.ProcessState.Timestamp,
		}
	}

	if as.WhenCondition != nil {
		env.WhenCondition = &api.ActionStatusWhenCondition{
			Timestamp:           as.WhenCondition.Timestamp,
			WhenConditionStatus: as.WhenCondition.WhenConditionStatus,
		}
	}

	return env
}

// validateSchemas calls validation endpoints on the metadata-api to validate
// step schemas.
func (rr *RealRunner) validateSchemas(ctx context.Context, mu *url.URL) error {
	ve := &url.URL{Path: "/validate"}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, mu.ResolveReference(ve).String(), http.NoBody)
	if err != nil {
		return err
	}

	resp, err := rr.getResponse(ctx, req, []retry.WaitOption{})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// FIXME This should send the explicit start time to the endpoint
func (rr *RealRunner) setStepInitTimer(ctx context.Context, mu *url.URL) error {
	te := &url.URL{Path: path.Join("/timers", url.PathEscape(model.TimerStepInit))}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut, mu.ResolveReference(te).String(), http.NoBody)
	if err != nil {
		return err
	} else {
		resp, err := rr.getResponse(ctx, req, []retry.WaitOption{})
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	return nil
}

func (rr *RealRunner) getResponse(ctx context.Context, request *http.Request, waitOptions []retry.WaitOption) (*http.Response, error) {
	contextWithTimeout, cancel := context.WithTimeout(ctx, rr.Config.DefaultTimeout)
	defer cancel()

	var response *http.Response
	err := retry.Wait(contextWithTimeout, func(ctx context.Context) (bool, error) {
		var rerr error
		response, rerr = http.DefaultClient.Do(request)
		if rerr != nil {
			return retry.Repeat(rerr)
		}

		if response != nil {
			// TODO Consider expanding to all 5xx (and possibly some 4xx) status codes
			switch response.StatusCode {
			case http.StatusInternalServerError, http.StatusBadGateway,
				http.StatusServiceUnavailable, http.StatusGatewayTimeout:
				return retry.Repeat(fmt.Errorf("unexpected status code %d", response.StatusCode))
			}
		}

		return retry.Done(nil)
	}, waitOptions...)
	if err != nil {
		return nil, err
	}

	return response, nil
}
