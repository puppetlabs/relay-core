package op

import (
	"context"
	"os"

	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs/configmap"
	omemory "github.com/puppetlabs/nebula-tasks/pkg/outputs/memory"
	smemory "github.com/puppetlabs/nebula-tasks/pkg/secrets/memory"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
	"github.com/puppetlabs/nebula-tasks/pkg/task"

	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// ManagerFactory provides data access managers for various external services
// where data resides.
type ManagerFactory interface {
	SecretsManager() SecretsManager
	OutputsManager() OutputsManager
	MetadataManager() MetadataManager
	SpecsManager() SpecsManager
}

// DefaultManagerFactory is the default ManagerFactory implementation. It is very opinionated
// within the context of nebula and the workflow system.
type DefaultManagerFactory struct {
	sm  SecretsManager
	om  OutputsManager
	mm  MetadataManager
	spm SpecsManager
}

// SecretsManager creates and returns a new SecretsManager implementation.
func (m DefaultManagerFactory) SecretsManager() SecretsManager {
	return m.sm
}

// OutputsManager creates and returns a new OutputsManager based on values in Configuration type.
func (m DefaultManagerFactory) OutputsManager() OutputsManager {
	return m.om
}

func (m DefaultManagerFactory) MetadataManager() MetadataManager {
	return m.mm
}

func (m DefaultManagerFactory) SpecsManager() SpecsManager {
	return m.spm
}

// NewDefaultManagerFactory creates and returns a new DefaultManagerFactory
func NewForKubernetes(ctx context.Context, cfg *config.MetadataServerConfig) (*DefaultManagerFactory, errors.Error) {
	kc, err := NewKubeclientFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	sm, err := vault.NewVaultWithKubernetesAuth(ctx, &vault.Config{
		Addr:                       cfg.VaultAddr,
		K8sServiceAccountTokenPath: cfg.K8sServiceAccountTokenPath,
		Token:                      cfg.VaultToken,
		Role:                       cfg.VaultRole,
		Bucket:                     cfg.WorkflowID,
		EngineMount:                cfg.VaultEngineMount,
		Logger:                     cfg.Logger,
	})
	if err != nil {
		return nil, err
	}

	om := configmap.New(kc, cfg.Namespace)
	mm := task.NewKubernetesMetadataManager(kc, cfg.Namespace)
	spm := task.NewKubernetesSpecManager(kc, cfg.Namespace)

	return &DefaultManagerFactory{
		sm:  sm,
		om:  om,
		mm:  mm,
		spm: spm,
	}, nil
}

type developmentPreConfig struct {
	Secrets      map[string]string         `yaml:"secrets"`
	TaskMetadata map[string]*task.Metadata `yaml:"taskMetadata"`
	TaskSpecs    map[string]string         `yaml:"taskSpecs"`
}

func NewForDev(ctx context.Context, cfg *config.MetadataServerConfig) (*DefaultManagerFactory, errors.Error) {
	var preCfg developmentPreConfig

	f, err := os.Open(cfg.DevelopmentPreConfigPath)
	if err != nil {
		return nil, errors.NewServerPreConfigDecodingError().WithCause(err)
	}

	if err := yaml.NewDecoder(f).Decode(&preCfg); err != nil {
		return nil, errors.NewServerPreConfigDecodingError().WithCause(err)
	}

	sm := smemory.New(preCfg.Secrets)
	om := omemory.New()
	mm := task.NewPreconfiguredMetadataManager(preCfg.TaskMetadata)
	spm := task.NewPreconfiguredSpecManager(preCfg.TaskSpecs)

	return &DefaultManagerFactory{
		sm:  sm,
		om:  om,
		mm:  mm,
		spm: spm,
	}, nil
}

func NewKubeclientFromConfig(cfg *config.MetadataServerConfig) (kubernetes.Interface, errors.Error) {
	kcfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.KubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: cfg.K8sMasterURL}},
	)

	kcc, err := kcfg.ClientConfig()
	if err != nil {
		return nil, errors.NewKubernetesManagerSetupError().WithCause(err)
	}

	kubeclient, err := kubernetes.NewForConfig(kcc)
	if err != nil {
		return nil, errors.NewKubernetesManagerSetupError().WithCause(err)
	}

	return kubeclient, nil
}
