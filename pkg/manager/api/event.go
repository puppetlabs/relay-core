package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type UnexpectedResponseError struct {
	StatusCode int
	Cause      errawr.Error
}

func (e *UnexpectedResponseError) Error() string {
	return fmt.Sprintf("received HTTP %d from API: %+v", e.StatusCode, e.Cause)
}

// TODO: Support event sources other than triggers?

type EventManager struct {
	me    model.Action
	url   string
	token string
}

var _ model.EventManager = &EventManager{}

func (m *EventManager) Emit(ctx context.Context, data map[string]interface{}) (*model.Event, error) {
	switch at := m.me.(type) {
	case *model.Trigger:
		encoded := make(map[string]transfer.JSONInterface, len(data))
		for k, v := range data {
			encoded[k] = transfer.JSONInterface{Data: v}
		}

		env := &postEventRequestEnvelope{
			Source: &triggerEventSourceEnvelope{
				Type: "trigger",
				Trigger: &triggerIdentifierEnvelope{
					Name: at.Name,
				},
			},
			Data: encoded,
		}

		b, err := json.Marshal(env)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodPost, m.url, bytes.NewBuffer(b))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.token))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			err := &UnexpectedResponseError{
				StatusCode: resp.StatusCode,
			}

			// Try to decode an error.
			var env utilapi.ErrorEnvelope
			if derr := json.NewDecoder(resp.Body).Decode(&env); derr == nil && env.Error != nil {
				err.Cause = env.Error.AsError()
			}

			return nil, err
		}

		return &model.Event{
			Data: data,
		}, nil
	default:
		return nil, model.ErrRejected
	}
}

func NewEventManager(action model.Action, url, token string) *EventManager {
	return &EventManager{
		me:    action,
		url:   url,
		token: token,
	}
}

type triggerIdentifierEnvelope struct {
	Name string `json:"name"`
}

type triggerEventSourceEnvelope struct {
	Type    string                     `json:"type"`
	Trigger *triggerIdentifierEnvelope `json:"trigger"`
}

type postEventRequestEnvelope struct {
	Source *triggerEventSourceEnvelope       `json:"source"`
	Data   map[string]transfer.JSONInterface `json:"data"`
}
