package obj_test

import (
	"context"
	"encoding/json"
	"net/url"
	"path"
	"testing"

	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/obj"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWorkflowRunDepsConfigureAnnotate(t *testing.T) {
	ctx := context.Background()

	WithTestNamespace(t, ctx, func(namespace *obj.Namespace) {
		cl := Client(t)

		require.NoError(t, cl.Create(ctx, &nebulav1.WorkflowRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-test-run",
				Namespace: namespace.Name,
			},
			Spec: nebulav1.WorkflowRunSpec{
				Name: "my-workflow-run-1234",
				Workflow: nebulav1.Workflow{
					Name: "my-workflow",
					Steps: []*nebulav1.WorkflowStep{
						{
							Name: "my-test-step",
						},
					},
				},
			},
		}))

		run := obj.NewWorkflowRun(client.ObjectKey{
			Namespace: namespace.Name,
			Name:      "my-test-run",
		})

		ok, err := run.Load(ctx, cl)
		require.NoError(t, err)
		require.True(t, ok)

		deps, err := obj.ApplyWorkflowRunDeps(
			ctx,
			cl,
			run,
			authenticate.IssuerFunc(func(ctx context.Context, claims *authenticate.Claims) (authenticate.Raw, error) {
				tok, err := json.Marshal(claims)
				require.NoError(t, err)

				return authenticate.Raw(tok), nil
			}),
			&url.URL{Scheme: "https", Host: "metadata-api.example.com"},
		)
		require.NoError(t, err)

		ws := run.Object.Spec.Workflow.Steps[0]

		var md metav1.ObjectMeta
		require.NoError(t, deps.AnnotateStepToken(ctx, &md, ws))

		tok := md.GetAnnotations()[authenticate.KubernetesTokenAnnotation]
		require.NotEmpty(t, tok)

		var claims authenticate.Claims
		require.NoError(t, json.Unmarshal([]byte(tok), &claims))

		sat, err := deps.MetadataAPIServiceAccount.DefaultTokenSecret.Token()
		require.NoError(t, err)
		require.NotEmpty(t, sat)

		assert.Equal(t, authenticate.ControllerIssuer, claims.Issuer)
		assert.Equal(t, jwt.Audience{authenticate.MetadataAPIAudienceV1}, claims.Audience)
		assert.Equal(t, path.Join("steps", obj.ModelStep(run, ws).Hash().HexEncoding()), claims.Subject)
		assert.NotNil(t, claims.Expiry)
		assert.NotNil(t, claims.NotBefore)
		assert.NotNil(t, claims.IssuedAt)
		assert.Equal(t, namespace.Name, claims.KubernetesNamespaceName)
		assert.Equal(t, string(namespace.Object.GetUID()), claims.KubernetesNamespaceUID)
		assert.Equal(t, sat, claims.KubernetesServiceAccountToken)
		assert.Equal(t, run.Object.Spec.Name, claims.RelayRunID)
		assert.Equal(t, ws.Name, claims.RelayName)
		assert.Equal(t, deps.ImmutableConfigMap.Key.Name, claims.RelayKubernetesImmutableConfigMapName)
		assert.Equal(t, deps.MutableConfigMap.Key.Name, claims.RelayKubernetesMutableConfigMapName)
	})
}
