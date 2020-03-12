package workflow

import (
	"context"
	"time"

	nebclientset "github.com/puppetlabs/nebula-tasks/pkg/generated/clientset/versioned"
	nebinformers "github.com/puppetlabs/nebula-tasks/pkg/generated/informers/externalversions"
	nebv1informers "github.com/puppetlabs/nebula-tasks/pkg/generated/informers/externalversions/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
	tekclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tekinformers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	pipelinev1beta1informers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const DefaultResyncPeriod = time.Second * 10

type DependencyManager struct {
	KubeClient            kubernetes.Interface
	NebulaClient          nebclientset.Interface
	TektonClient          tekclientset.Interface
	NebulaInformerFactory nebinformers.SharedInformerFactory
	TektonInformerFactory tekinformers.SharedInformerFactory
}

func (d DependencyManager) WorkflowRunInformer() nebv1informers.WorkflowRunInformer {
	return d.NebulaInformerFactory.Nebula().V1().WorkflowRuns()
}

func (d DependencyManager) PipelineRunInformer() pipelinev1beta1informers.PipelineRunInformer {
	return d.TektonInformerFactory.Tekton().V1beta1().PipelineRuns()
}

func NewDependencyManager(kcfg *rest.Config) (*DependencyManager, error) {
	kubeclient, err := kubernetes.NewForConfig(kcfg)
	if err != nil {
		return nil, err
	}

	nebclient, err := nebclientset.NewForConfig(kcfg)
	if err != nil {
		return nil, err
	}

	tekclient, err := tekclientset.NewForConfig(kcfg)
	if err != nil {
		return nil, err
	}

	d := &DependencyManager{
		KubeClient:            kubeclient,
		NebulaClient:          nebclient,
		TektonClient:          tekclient,
		NebulaInformerFactory: nebinformers.NewSharedInformerFactory(nebclient, DefaultResyncPeriod),
		TektonInformerFactory: tekinformers.NewSharedInformerFactory(tekclient, DefaultResyncPeriod),
	}

	return d, nil
}

type SecretAuthAccessManager interface {
	GrantScopedAccess(ctx context.Context, workflowID, namespace, serviceAccount string) (*secrets.AccessGrant, error)
	RevokeScopedAccess(ctx context.Context, namespace string) error
}
