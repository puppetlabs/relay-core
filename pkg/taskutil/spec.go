package taskutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

const SpecURLEnvName = "SPEC_URL"

// SpecLoader returns an io.Reader containing the bytes
// of a task spec. This is used as input to a spec unmarshaler.
// An error is returned if the operation fails.
type SpecLoader interface {
	LoadSpec() (io.Reader, errors.Error)
}

type RemoteSpecLoader struct {
	u      *url.URL
	client *http.Client
}

func (r RemoteSpecLoader) LoadSpec() (io.Reader, errors.Error) {
	var client = http.DefaultClient

	if r.client != nil {
		client = http.DefaultClient
	}

	resp, err := client.Get(r.u.String())
	if err != nil {
		return nil, errors.NewTaskUtilSpecLoaderError("network request failed").WithCause(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewTaskUtilSpecLoaderError(fmt.Sprintf("unexpected status code %d", resp.StatusCode))
	}

	defer resp.Body.Close()

	buf := &bytes.Buffer{}

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, errors.NewTaskUtilSpecLoaderError("reading response from remote service failed").WithCause(err)
	}

	return buf, nil
}

func NewRemoteSpecLoader(u *url.URL, client *http.Client) RemoteSpecLoader {
	return RemoteSpecLoader{
		u:      u,
		client: client,
	}
}

type LocalSpecLoader struct {
	path string
}

func (l LocalSpecLoader) LoadSpec() (io.Reader, errors.Error) {
	b, err := ioutil.ReadFile(l.path)
	if err != nil {
		return nil, errors.NewTaskUtilSpecLoaderError("could not read file").WithCause(err)
	}

	return bytes.NewBuffer(b), nil
}

func NewLocalSpecLoader(path string) LocalSpecLoader {
	return LocalSpecLoader{path: path}
}

type SpecDecoder interface {
	DecodeSpec(io.Reader, interface{}) errors.Error
}

type DefaultJSONSpecDecoder struct{}

func (u DefaultJSONSpecDecoder) DecodeSpec(r io.Reader, v interface{}) errors.Error {
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return errors.NewTaskUtilDefaultJSONSpecDecodingError().WithCause(err)
	}

	return nil
}

type DefaultPlanOptions struct {
	Client  *http.Client
	SpecURL string
}

func PopulateSpecFromDefaultPlan(v interface{}, opts DefaultPlanOptions) errors.Error {
	location := opts.SpecURL

	if location == "" {
		location := os.Getenv(SpecURLEnvName)
		if location == "" {
			return errors.NewTaskUtilDefaultSpecPlanFailed("SPEC_URL was empty")
		}
	}

	u, err := url.Parse(location)
	if err != nil {
		return errors.NewTaskUtilDefaultSpecPlanFailed("parsing SPEC_URL failed").WithCause(err)
	}

	var loader SpecLoader

	switch u.Scheme {
	case "file":
		if u.Host != "" {
			return errors.NewTaskUtilDefaultSpecPlanFailed("unable to read from remote host in file URL")
		}

		loader = NewLocalSpecLoader(u.Path)
	case "http", "https":
		loader = NewRemoteSpecLoader(u, opts.Client)
	default:
		return errors.NewTaskUtilDefaultSpecPlanFailed(fmt.Sprintf("unknown scheme %s in spec URL", u.Scheme))
	}
	decoder := DefaultJSONSpecDecoder{}

	r, err := loader.LoadSpec()
	if err != nil {
		return errors.NewTaskUtilDefaultSpecPlanFailed("loading spec failed").WithCause(err)
	}

	if err := decoder.DecodeSpec(r, v); err != nil {
		return errors.NewTaskUtilDefaultSpecPlanFailed("decoding failed").WithCause(err)
	}

	return nil
}
