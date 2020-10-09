package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/alertstest"
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/expr/serialize"
	sdktestutil "github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/puppetlabs/relay-core/pkg/manager/memory"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/puppetlabs/relay-core/pkg/util/typeutil"
	"github.com/puppetlabs/relay-core/pkg/workflow/spec"
	"github.com/stretchr/testify/require"
)

func TestGetSpec(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	sc := &opt.SampleConfig{
		Connections: map[memory.ConnectionKey]map[string]interface{}{
			memory.ConnectionKey{Type: "aws", Name: "test-aws-connection"}: {
				"accessKeyID":     "testaccesskey",
				"secretAccessKey": "testsecretaccesskey",
			},
		},
		Secrets: map[string]string{
			"test-secret-key": "test-secret-value",
		},
		Runs: map[string]*opt.SampleConfigRun{
			"test": &opt.SampleConfigRun{
				Parameters: map[string]interface{}{
					"nice-parameter": "nice-value",
				},
				Steps: map[string]*opt.SampleConfigStep{
					"previous-task": &opt.SampleConfigStep{
						Outputs: map[string]interface{}{
							"test-output-key": "test-output-value",
							"test-structured-output-key": map[string]interface{}{
								"a":       "value",
								"another": "thing",
							},
						},
					},
					"current-task": &opt.SampleConfigStep{
						Spec: opt.SampleConfigSpec{
							"superParameter": serialize.YAMLTree{
								Tree: sdktestutil.JSONParameter("nice-parameter"),
							},
							"superSecret": serialize.YAMLTree{
								Tree: sdktestutil.JSONSecret("test-secret-key"),
							},
							"superOutput": serialize.YAMLTree{
								Tree: sdktestutil.JSONOutput("previous-task", "test-output-key"),
							},
							"structuredOutput": serialize.YAMLTree{
								Tree: sdktestutil.JSONOutput("previous-task", "test-structured-output-key"),
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
	}

	tokenMap := tokenGenerator.GenerateAll(ctx, sc)

	currentTaskToken, found := tokenMap.ForStep("test", "current-task")
	require.True(t, found)

	h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()))

	// Request the whole spec.
	req, err := http.NewRequest(http.MethodGet, "/spec", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+currentTaskToken)

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	var r evaluate.JSONResultEnvelope
	require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&r))
	require.Equal(t, map[string]interface{}{
		"superParameter": "nice-value",
		"superSecret":    "test-secret-value",
		"superOutput":    "test-output-value",
		"structuredOutput": map[string]interface{}{
			"a":       "value",
			"another": "thing",
		},
		"superConnection": map[string]interface{}{
			"accessKeyID":     "testaccesskey",
			"secretAccessKey": "testsecretaccesskey",
		},
		"mergedConnection": map[string]interface{}{
			"accessKeyID":     "testaccesskey",
			"secretAccessKey": "testsecretaccesskey",
			"merge":           "me",
		},
		"superNormal": "test-normal-value",
	}, r.Value.Data)
	require.True(t, r.Complete)

	// Request a specific expression from the spec.
	req.URL.RawQuery = url.Values{"q": []string{"structuredOutput.a"}}.Encode()

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	r = evaluate.JSONResultEnvelope{}
	require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&r))
	require.Equal(t, "value", r.Value.Data)
	require.True(t, r.Complete)

	// Request a specific expression from the spec using the JSON path query language
	req.URL.RawQuery = url.Values{
		"q":    []string{"$.structuredOutput.a"},
		"lang": []string{"jsonpath"},
	}.Encode()

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	r = evaluate.JSONResultEnvelope{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&r))
	require.Equal(t, "value", r.Value.Data)
	require.True(t, r.Complete)
}

func TestSpecValidationCapture(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	var cases = []struct {
		description      string
		sc               *opt.SampleConfig
		validationReport *alertstest.ReporterRecorder
	}{
		{
			description: "missing spec schema",
			sc: &opt.SampleConfig{
				Connections: map[memory.ConnectionKey]map[string]interface{}{
					memory.ConnectionKey{Type: "aws", Name: "test-aws-connection"}: {
						"accessKeyID":     "testaccesskey",
						"secretAccessKey": "testsecretaccesskey",
					},
				},
				Secrets: map[string]string{
					"test-secret-key": "test-secret-value",
				},
				Runs: map[string]*opt.SampleConfigRun{
					"test": &opt.SampleConfigRun{
						Steps: map[string]*opt.SampleConfigStep{
							"current-task": &opt.SampleConfigStep{
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
			validationReport: &alertstest.ReporterRecorder{
				Tags: []trackers.Tag{
					{
						Key:   "relay.spec.validation-error",
						Value: "relaysh/image",
					},
				},
				Err: errors.NewSpecSchemaLookupError().WithCause(&spec.SchemaDoesNotExistError{Name: "relaysh/image"}),
			},
		},
		{
			description: "invalid spec schema",
			sc: &opt.SampleConfig{
				Connections: map[memory.ConnectionKey]map[string]interface{}{
					memory.ConnectionKey{Type: "kubernetes", Name: "test-kubernetes-connection"}: {
						"server":               "https://127.0.0.1:6443",
						"token":                "test",
						"certificateAuthority": "test",
					},
				},
				Runs: map[string]*opt.SampleConfigRun{
					"test": &opt.SampleConfigRun{
						Steps: map[string]*opt.SampleConfigStep{
							"current-task": &opt.SampleConfigStep{
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
			validationReport: &alertstest.ReporterRecorder{
				Tags: []trackers.Tag{
					{
						Key:   "relay.spec.validation-error",
						Value: "relaysh/kubernetes-step-kubectl",
					},
				},
				Err: errors.NewSpecSchemaValidationError().WithCause(&spec.SchemaValidationError{
					Cause: &typeutil.ValidationError{
						FieldErrors: []*typeutil.FieldValidationError{
							&typeutil.FieldValidationError{Context: "(root)", Field: "(root)", Description: "command is required", Type: "required"},
						},
					},
				}),
			},
		},
		{
			description: "valid spec schema",
			sc: &opt.SampleConfig{
				Connections: map[memory.ConnectionKey]map[string]interface{}{
					memory.ConnectionKey{Type: "kubernetes", Name: "test-kubernetes-connection"}: {
						"server":               "https://127.0.0.1:6443",
						"token":                "test",
						"certificateAuthority": "test",
					},
				},
				Runs: map[string]*opt.SampleConfigRun{
					"test": &opt.SampleConfigRun{
						Steps: map[string]*opt.SampleConfigStep{
							"current-task": &opt.SampleConfigStep{
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

			capturer := alertstest.NewCapturer()

			testutil.WithStepMetadataSchemaRegistry(t, filepath.Join("testdata/step-metadata.json"), func(reg spec.SchemaRegistry) {
				h := api.NewHandler(sample.NewAuthenticator(c.sc, tokenGenerator.Key()), api.WithSpecSchemaRegistry(reg))
				h = capturer.Middleware().Wrap(h)

				req, err := http.NewRequest(http.MethodGet, "/spec", nil)
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+currentTaskToken)

				resp := httptest.NewRecorder()
				h.ServeHTTP(resp, req)
				require.Equal(t, http.StatusOK, resp.Result().StatusCode, resp.Body.String())

				if c.validationReport != nil {
					require.Len(t, capturer.ReporterRecorders, 1)
					report := capturer.ReporterRecorders[0]

					require.Equal(t, c.validationReport, report)
				} else {
					require.Len(t, capturer.ReporterRecorders, 0)
				}
			})
		})
	}
}
