package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/relay-core/pkg/manager/api"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventManager(t *testing.T) {
	ctx := context.Background()

	trigger := &model.Trigger{
		Name: "foo",
	}

	data := map[string]interface{}{
		"foo": "bar",
		"baz": []interface{}{float64(1), float64(2), float64(3)},
	}

	key := uuid.New().String()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/events", r.URL.Path)
		assert.Equal(t, "Bearer token", r.Header.Get("authorization"))

		var env struct {
			Source struct {
				Type    string `json:"type"`
				Trigger struct {
					Name string `json:"name"`
				} `json:"trigger"`
			} `json:"source"`
			Data map[string]interface{} `json:"data"`
			Key  string                 `json:"key"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&env))

		assert.Equal(t, "trigger", env.Source.Type)
		assert.Equal(t, trigger.Name, env.Source.Trigger.Name)
		assert.Equal(t, data, env.Data)
		assert.Equal(t, key, env.Key)

		w.WriteHeader(http.StatusAccepted)
	}))
	defer s.Close()

	em := api.NewEventManager(trigger, fmt.Sprintf("%s/api/events", s.URL), "token")

	ev, err := em.Emit(ctx, data, key)
	require.NoError(t, err)
	require.Equal(t, data, ev.Data)
}
