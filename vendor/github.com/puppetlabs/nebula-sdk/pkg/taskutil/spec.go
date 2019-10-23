package taskutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

const SpecURLEnvName = "SPEC_URL"

// SpecLoader returns an io.Reader containing the bytes
// of a task spec. This is used as input to a spec unmarshaler.
// An error is returned if the operation fails.
type SpecLoader interface {
	LoadSpec() (io.Reader, error)
}

type RemoteSpecLoader struct {
	u      *url.URL
	client *http.Client
}

func (r RemoteSpecLoader) LoadSpec() (io.Reader, error) {
	var client = http.DefaultClient

	if r.client != nil {
		client = http.DefaultClient
	}

	resp, err := client.Get(r.u.String())
	if err != nil {
		return nil, fmt.Errorf("network request failed: %+v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	buf := &bytes.Buffer{}

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("reading response from remote service failed: %+v", err)
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

func (l LocalSpecLoader) LoadSpec() (io.Reader, error) {
	b, err := ioutil.ReadFile(l.path)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %+v", l.path, err)
	}

	return bytes.NewBuffer(b), nil
}

func NewLocalSpecLoader(path string) LocalSpecLoader {
	return LocalSpecLoader{path: path}
}

type SpecDecoder interface {
	DecodeSpec(io.Reader, interface{}) error
}

type DefaultJSONSpecDecoder struct{}

func (u DefaultJSONSpecDecoder) DecodeSpec(r io.Reader, v interface{}) error {
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return fmt.Errorf("failed to decode JSON: %+v", err)
	}

	return nil
}

type DefaultPlanOptions struct {
	Client  *http.Client
	SpecURL string
}

func PopulateSpecFromDefaultPlan(v interface{}, opts DefaultPlanOptions) error {
	location := opts.SpecURL

	if location == "" {
		location := os.Getenv(SpecURLEnvName)
		if location == "" {
			return errors.New("SPEC_URL was empty")
		}
	}

	u, err := url.Parse(location)
	if err != nil {
		return fmt.Errorf("parsing SPEC_URL failed: %+v", err)
	}

	var loader SpecLoader

	switch u.Scheme {
	case "file":
		if u.Host != "" {
			return errors.New("unable to read from remote host in file URL")
		}

		loader = NewLocalSpecLoader(u.Path)
	case "http", "https":
		loader = NewRemoteSpecLoader(u, opts.Client)
	default:
		return fmt.Errorf("unknown scheme %s in spec URL", u.Scheme)
	}
	decoder := DefaultJSONSpecDecoder{}

	r, err := loader.LoadSpec()
	if err != nil {
		return err
	}

	if err := decoder.DecodeSpec(r, v); err != nil {
		return err
	}

	return nil
}
