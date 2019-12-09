package state

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

var (
	ErrStateClientKeyEmpty      = errors.New("key is required but was empty")
	ErrStateClientValueEmpty    = errors.New("value is required but was empty")
	ErrStateClientTaskNameEmpty = errors.New("taskName is required but was empty")
	ErrStateClientNotFound      = errors.New("state was not found")
)

type StateClient interface {
	SetState(ctx context.Context, key, value string) error
	GetState(ctx context.Context, key string) (string, error)
}

type DefaultStateClient struct {
	apiURL *url.URL
}

func (c DefaultStateClient) SetState(ctx context.Context, key, value string) error {
	if key == "" {
		return ErrStateClientKeyEmpty
	}

	if value == "" {
		return ErrStateClientValueEmpty
	}

	loc := *c.apiURL
	loc.Path = path.Join(loc.Path, key)

	buf := bytes.NewBufferString(value)

	req, err := http.NewRequest("PUT", loc.String(), buf)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return nil
}

func (c DefaultStateClient) GetState(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", ErrStateClientKeyEmpty
	}

	loc := *c.apiURL
	loc.Path = path.Join(loc.Path, key)

	req, err := http.NewRequest("GET", loc.String(), nil)
	if err != nil {
		return "", err
	}

	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return "", ErrStateClientNotFound
		}

		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var state StateEnvelope

	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return "", err
	}

	v, err := state.Value.Decode()
	if err != nil {
		return "", err
	}

	return string(v), nil
}

func NewDefaultStateClient(location *url.URL) StateClient {
	return &DefaultStateClient{apiURL: location}
}
