package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/serialize"
	sdktestutil "github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/puppetlabs/relay-core/pkg/manager/memory"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/puppetlabs/relay-core/pkg/util/typeutil"
	"github.com/puppetlabs/relay-core/pkg/workflow/validation"
	"github.com/stretchr/testify/require"
)

func TestValidationCapture(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	var cases = []struct {
		description string
		sc          *opt.SampleConfig
		err         errors.Error
	}{
		{
			description: "missing spec schema",
			sc: &opt.SampleConfig{
				Connections: map[memory.ConnectionKey]map[string]interface{}{
					{Type: "aws", Name: "test-aws-connection"}: {
						"accessKeyID":     "testaccesskey",
						"secretAccessKey": "testsecretaccesskey",
					},
				},
				Secrets: map[string]string{
					"test-secret-key": "test-secret-value",
				},
				Runs: map[string]*opt.SampleConfigRun{
					"test": {
						Steps: map[string]*opt.SampleConfigStep{
							"current-task": {
								Image: "relaysh/image:latest",
								Spec: opt.SampleConfigSpec{
									"superSecret": serialize.YAMLTree{
										Tree: sdktestutil.JSONSecret("test-secret-key"),
									},
									"superConnection": serialize.YAMLTree{
										Tree: sdktestutil.JSONConnection("aws", "test-aws-connection"),
									},
									"mergedConnection": serialize.YAMLTree{
										Tree: sdktestutil.JSONInvocation("merge", []interface{}{
											sdktestutil.JSONConnection("aws", "test-aws-connection"),
											map[string]interface{}{"merge": "me"},
										}),
									},
									"superNormal": serialize.YAMLTree{
										Tree: "test-normal-value",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "invalid spec schema",
			sc: &opt.SampleConfig{
				Connections: map[memory.ConnectionKey]map[string]interface{}{
					{Type: "kubernetes", Name: "test-kubernetes-connection"}: {
						"server":               "https://127.0.0.1:6443",
						"token":                "test",
						"certificateAuthority": "test",
					},
				},
				Runs: map[string]*opt.SampleConfigRun{
					"test": {
						Steps: map[string]*opt.SampleConfigStep{
							"current-task": {
								Image: "relaysh/kubernetes-step-kubectl:latest",
								Spec: opt.SampleConfigSpec{
									"cluster": {
										Tree: map[string]interface{}{
											"name":       "test-cluster",
											"connection": sdktestutil.JSONConnection("kubernetes", "test-kubernetes-connection"),
										},
									},
								},
							},
						},
					},
				},
			},
			err: errors.NewValidationSchemaValidationError().WithCause(&validation.SchemaValidationError{
				Cause: &typeutil.ValidationError{
					FieldErrors: []*typeutil.FieldValidationError{
						{Context: "(root)", Field: "(root)", Description: "command is required", Type: "required"},
					},
				},
			}),
		},
		{
			description: "valid spec schema",
			sc: &opt.SampleConfig{
				Connections: map[memory.ConnectionKey]map[string]interface{}{
					{Type: "kubernetes", Name: "test-kubernetes-connection"}: {
						"server":               "https://127.0.0.1:6443",
						"token":                "test",
						"certificateAuthority": "test",
					},
				},
				Runs: map[string]*opt.SampleConfigRun{
					"test": {
						Steps: map[string]*opt.SampleConfigStep{
							"current-task": {
								Image: "relaysh/kubernetes-step-kubectl:latest",
								Spec: opt.SampleConfigSpec{
									"cluster": {
										Tree: map[string]interface{}{
											"name":       "test-cluster",
											"connection": sdktestutil.JSONConnection("kubernetes", "test-kubernetes-connection"),
										},
									},
									"command": serialize.YAMLTree{
										Tree: "apply",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			tokenMap := tokenGenerator.GenerateAll(ctx, c.sc)

			currentTaskToken, found := tokenMap.ForStep("test", "current-task")
			require.True(t, found)

			testutil.WithStepMetadataSchemaRegistry(t, "testdata/step-metadata.json", func(reg validation.SchemaRegistry) {
				authenticator := sample.NewAuthenticator(c.sc, tokenGenerator.Key())
				h := api.NewHandler(authenticator, api.WithSchemaRegistry(reg))

				req, err := http.NewRequest(http.MethodPost, "/validate", nil)
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+currentTaskToken)

				resp := httptest.NewRecorder()
				h.ServeHTTP(resp, req)
				require.Equal(t, http.StatusOK, resp.Result().StatusCode, resp.Body.String())

				credential, err := authenticator.Authenticate(req)
				require.NoError(t, err)

				m := credential.Managers
				sm := m.StepMessages()

				sms, err := sm.List(ctx)
				require.NoError(t, err)

				if c.err != nil {
					require.Len(t, sms, 1)
					require.Equal(t, c.err.Error(), sms[0].Details)
				} else {
					require.Len(t, sms, 0)
				}
			})
		})
	}
}
