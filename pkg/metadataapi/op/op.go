package op

import (
	"context"
	"os"

	cconfigmap "github.com/puppetlabs/nebula-tasks/pkg/conditionals/configmap"
	cmemory "github.com/puppetlabs/nebula-tasks/pkg/conditionals/memory"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs/configmap"
	omemory "github.com/puppetlabs/nebula-tasks/pkg/outputs/memory"
	smemory "github.com/puppetlabs/nebula-tasks/pkg/secrets/memory"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
	stconfigmap "github.com/puppetlabs/nebula-tasks/pkg/state/configmap"
	stmemory "github.com/puppetlabs/nebula-tasks/pkg/state/memory"
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
	StateManager() StateManager
	MetadataManager() MetadataManager
	SpecsManager() SpecsManager
	ConditionalsManager() ConditionalsManager
}

// DefaultManagerFactory is the default ManagerFactory implementation. It is very opinionated
// within the context of nebula and the workflow system.
type DefaultManagerFactory struct {
	sm  SecretsManager
	om  OutputsManager
	stm StateManager
	mm  MetadataManager
	spm SpecsManager
	cm  ConditionalsManager
}

// SecretsManager returns a configured SecretsManager implementation.
// See pkg/metadataapi/op/secretsmanager.go
func (m DefaultManagerFactory) SecretsManager() SecretsManager {
	return m.sm
}

// OutputsManager returns a configured OutputsManager based on values in Configuration type.
// See pkg/metadataapi/op/outputsmanager.go
func (m DefaultManagerFactory) OutputsManager() OutputsManager {
	return m.om
}

// StateManager returns a configured StateManager.
// See pkg/metadataapi/op/statemanager.go
func (m DefaultManagerFactory) StateManager() StateManager {
	return m.stm
}

// MetadataManager returns a configured MetadataManager used to get task metadata.
// See pkg/metadataapi/op/metadatamanager.go
func (m DefaultManagerFactory) MetadataManager() MetadataManager {
	return m.mm
}

// SpecsManager returns a configured SpecsManager.
// See pkg/metadataapi/op/specsmanager.go
func (m DefaultManagerFactory) SpecsManager() SpecsManager {
	return m.spm
}

// ConditionalsManager returns a configured ConditionalsManager.
// See pkg/metadataapi/op/conditionalsmanager.go
func (m DefaultManagerFactory) ConditionalsManager() ConditionalsManager {
	return m.cm
}

// NewDefaultManagerFactory creates and returns a new DefaultManagerFactory
func NewForKubernetes(ctx context.Context, cfg *config.MetadataServerConfig) (*DefaultManagerFactory, errors.Error) {
	kc, err := NewKubeclientFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	sm, err := vault.NewVaultWithKubernetesAuth(ctx, &vault.Config{
		Addr:                       cfg.VaultAddr,
		K8sAuthMountPath:           cfg.VaultAuthMountPath,
		K8sServiceAccountTokenPath: cfg.K8sServiceAccountTokenPath,
		Token:                      cfg.VaultToken,
		Role:                       cfg.VaultRole,
		ScopedSecretsPath:          cfg.ScopedSecretsPath,
		Logger:                     cfg.Logger,
	})
	if err != nil {
		return nil, err
	}

	om := configmap.New(kc, cfg.Namespace)
	stm := stconfigmap.New(kc, cfg.Namespace)
	mm := task.NewKubernetesMetadataManager(kc, cfg.Namespace)
	spm := task.NewKubernetesSpecManager(kc, cfg.Namespace)
	cm := cconfigmap.New(kc, cfg.Namespace)

	return &DefaultManagerFactory{
		sm:  NewEncodingSecretManager(sm),
		om:  om,
		stm: NewEncodeDecodingStateManager(stm),
		mm:  mm,
		spm: spm,
		cm:  cm,
	}, nil
}

type developmentPreConfig struct {
	Secrets          map[string]string         `yaml:"secrets"`
	TaskMetadata     map[string]*task.Metadata `yaml:"taskMetadata"`
	TaskSpecs        map[string]string         `yaml:"taskSpecs"`
	TaskConditionals map[string]string         `yaml:"taskConditionals"`
}

// NewForDev returns a new DefaultManagerFactory useful for development and running the metadata api server
// locally. The manager implementations are mostly non-persistent in-memory storage backends. No data will be written
// to disk and this should not be used in production.
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
	stm := stmemory.New()
	mm := task.NewPreconfiguredMetadataManager(preCfg.TaskMetadata)
	spm := task.NewPreconfiguredSpecManager(preCfg.TaskSpecs)
	cm := cmemory.New(preCfg.TaskConditionals)

	return &DefaultManagerFactory{
		sm:  NewEncodingSecretManager(sm),
		om:  om,
		stm: NewEncodeDecodingStateManager(stm),
		mm:  mm,
		spm: spm,
		cm:  cm,
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
