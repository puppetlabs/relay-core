package obj_test

import (
	"context"
	"os"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/reconciler/workflow/obj"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var e2e *testutil.EndToEndEnvironment

func Client(t *testing.T) client.Client {
	require.NotNil(t, e2e)
	return e2e.ControllerRuntimeClient
}

func WithTestNamespace(t *testing.T, ctx context.Context, fn func(ns *obj.Namespace)) {
	require.NotNil(t, e2e)
	e2e.WithTestNamespace(t, ctx, func(ns *corev1.Namespace) {
		fn(&obj.Namespace{Name: ns.GetName(), Object: ns})
	})
}

func TestMain(m *testing.M) {
	os.Exit(testutil.RunEndToEnd(m, func(e *testutil.EndToEndEnvironment) {
		e2e = e
	}))
}
