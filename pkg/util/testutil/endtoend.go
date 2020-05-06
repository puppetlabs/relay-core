package testutil

import (
	"context"
	"fmt"
	"log"
	"path"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/puppetlabs/nebula-tasks/pkg/reconciler/workflow/obj"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	DefaultTektonPipelineVersion = "0.11.3"
)

type EndToEndEnvironment struct {
	RESTConfig              *rest.Config
	ControllerRuntimeClient client.Client
	Interface               kubernetes.Interface
}

func (e *EndToEndEnvironment) WithTestNamespace(t *testing.T, ctx context.Context, fn func(ns *corev1.Namespace)) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("relay-e2e-%s", uuid.New()),
			Labels: map[string]string{
				"testing.relay.sh/harness": "end-to-end",
			},
		},
	}
	require.NoError(t, e.ControllerRuntimeClient.Create(ctx, ns))
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		assert.NoError(t, e.ControllerRuntimeClient.Delete(ctx, ns))
	}()

	fn(ns)
}

type EndToEndEnvironmentOption func(ctx context.Context, e *EndToEndEnvironment) error

func EndToEndEnvironmentWithTekton(ctx context.Context, e *EndToEndEnvironment) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return doInstallTektonPipeline(ctx, e.ControllerRuntimeClient, viper.GetString("tekton_pipeline_version"))
}

var _ EndToEndEnvironmentOption = EndToEndEnvironmentWithTekton

func doEndToEndEnvironment(ctx context.Context, fn func(e *EndToEndEnvironment), opts ...EndToEndEnvironmentOption) (bool, error) {
	viper.SetEnvPrefix("relay_test_e2e")
	viper.AutomaticEnv()

	viper.SetDefault("label_nodes", false)
	viper.SetDefault("tekton_pipeline_version", DefaultTektonPipelineVersion)
	viper.SetDefault("disabled", false)

	if viper.GetBool("disabled") {
		return false, nil
	}

	env := &envtest.Environment{
		CRDDirectoryPaths: []string{
			path.Join(ModuleDirectory, "manifests/resources"),
		},
		AttachControlPlaneOutput: true,
		UseExistingCluster:       func(b bool) *bool { return &b }(true),
	}

	cfg, err := env.Start()
	if err != nil {
		return true, fmt.Errorf("failed to connect to Kubernetes cluster: %+v", err)
	}
	defer env.Stop()

	client, err := client.New(cfg, client.Options{
		Scheme: TestScheme,
	})
	if err != nil {
		return true, fmt.Errorf("failed to configure client: %+v", err)
	}

	if viper.GetBool("label_nodes") {
		nodes := &corev1.NodeList{}
		if err := client.List(ctx, nodes); err != nil {
			return true, fmt.Errorf("failed to list nodes to label: %+v", err)
		}

		for _, node := range nodes.Items {
			for name, value := range obj.PipelineRunPodNodeSelector {
				node.GetLabels()[name] = value
			}

			if err := client.Update(ctx, &node); err != nil {
				return true, fmt.Errorf("failed to update node %s: %+v", node.GetName(), err)
			}
		}
	}

	ifc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return true, fmt.Errorf("failed to configure client: %+v", err)
	}

	e := &EndToEndEnvironment{
		RESTConfig:              cfg,
		ControllerRuntimeClient: client,
		Interface:               ifc,
	}

	for _, opt := range opts {
		if err := opt(ctx, e); err != nil {
			return true, err
		}
	}

	fn(e)
	return true, nil
}

func WithEndToEndEnvironment(t *testing.T, ctx context.Context, fn func(e *EndToEndEnvironment), opts ...EndToEndEnvironmentOption) {
	enabled, err := doEndToEndEnvironment(ctx, fn)
	require.NoError(t, err)
	if !enabled {
		log.Println("end-to-end tests disabled by configuration")
	}
}

func RunEndToEnd(m *testing.M, fn func(e *EndToEndEnvironment), opts ...EndToEndEnvironmentOption) int {
	if enabled, err := doEndToEndEnvironment(context.Background(), fn); err != nil {
		log.Println(err)
		return 1
	} else if !enabled {
		log.Println("end-to-end tests disabled by configuration")
		return 0
	}

	return m.Run()
}
