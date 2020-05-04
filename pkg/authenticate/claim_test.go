package authenticate_test

import (
	"fmt"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestClaimActions(t *testing.T) {
	tests := []struct {
		Claims         *authenticate.Claims
		ExpectedAction model.Action
	}{
		{
			Claims: &authenticate.Claims{
				Claims: &jwt.Claims{
					Subject: "tests/foo",
				},
			},
		},
		{
			Claims: &authenticate.Claims{
				Claims: &jwt.Claims{
					Subject: fmt.Sprintf("steps/%s", (&model.Step{
						Run:  model.Run{ID: "foo"},
						Name: "bar",
					}).Hash().HexEncoding()),
				},
				RelayRunID: "foo",
				RelayName:  "bar",
			},
			ExpectedAction: &model.Step{
				Run:  model.Run{ID: "foo"},
				Name: "bar",
			},
		},
		{
			Claims: &authenticate.Claims{
				Claims: &jwt.Claims{
					Subject: fmt.Sprintf("triggers/%s", (&model.Trigger{Name: "foo"}).Hash().HexEncoding()),
				},
				RelayName: "foo",
			},
			ExpectedAction: &model.Trigger{
				Name: "foo",
			},
		},
		{
			Claims: &authenticate.Claims{
				Claims: &jwt.Claims{
					Subject: fmt.Sprintf("steps/%s", (&model.Step{
						Run:  model.Run{ID: "foo"},
						Name: "bar",
					}).Hash().HexEncoding()),
				},
				RelayRunID: "baz",
				RelayName:  "bar",
			},
			// Mismatched hash.
		},
		{
			Claims: &authenticate.Claims{
				Claims: &jwt.Claims{
					Subject: fmt.Sprintf("triggers/%s", (&model.Trigger{Name: "foo"}).Hash().HexEncoding()),
				},
				RelayName: "bar",
			},
			// Mismatched hash.
		},
	}
	for _, test := range tests {
		t.Run(test.Claims.Subject, func(t *testing.T) {
			require.Equal(t, test.ExpectedAction, test.Claims.Action())
		})
	}
}
