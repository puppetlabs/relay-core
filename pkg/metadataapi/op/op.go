package op

import (
	"context"
	"crypto/sha1"
	"os"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	cconfigmap "github.com/puppetlabs/nebula-tasks/pkg/conditionals/configmap"
	cmemory "github.com/puppetlabs/nebula-tasks/pkg/conditionals/memory"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	comemory "github.com/puppetlabs/nebula-tasks/pkg/connections/memory"
	connvault "github.com/puppetlabs/nebula-tasks/pkg/connections/vault"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs/configmap"
	omemory "github.com/puppetlabs/nebula-tasks/pkg/outputs/memory"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
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
	ConnectionsManager() ConnectionsManager
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
	cm  ConnectionsManager
	om  OutputsManager
	stm StateManager
	mm  MetadataManager
	spm SpecsManager
	cdm ConditionalsManager
}

// SecretsManager returns a configured SecretsManager implementation.
// See pkg/metadataapi/op/secretsmanager.go
func (m DefaultManagerFactory) SecretsManager() SecretsManager {
	return m.sm
}

// ConnectionsManager returns a configured ConnectionsManager implementation.
// See pkg/metadataapi/op/secretsmanager.go
func (m DefaultManagerFactory) ConnectionsManager() ConnectionsManager {
	return m.cm
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
	return m.cdm
}

// NewDefaultManagerFactory creates and returns a new DefaultManagerFactory
func NewForKubernetes(ctx context.Context, cfg *config.MetadataServerConfig) (*DefaultManagerFactory, errors.Error) {
	kc, err := NewKubeclientFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	// decode the access grants
	grantBytes, gerr := transfer.DecodeFromTransfer(cfg.VaultAccessGrants)
	if gerr != nil {
		return nil, errors.NewServerRunError().WithCause(gerr).Bug()
	}

	grants, gerr := secrets.UnmarshalGrants(grantBytes)
	if gerr != nil {
		return nil, errors.NewServerRunError().WithCause(gerr).Bug()
	}

	if _, ok := grants["workflows"]; !ok {
		return nil, errors.NewServerVaultGrantNotFound("workflows")
	}

	if _, ok := grants["connections"]; !ok {
		return nil, errors.NewServerVaultGrantNotFound("connections")
	}

	sm, err := vault.NewVaultWithKubernetesAuth(ctx, grants["workflows"], &vault.Config{
		Addr:                       grants["workflows"].BackendAddr,
		K8sAuthMountPath:           cfg.VaultAuthMountPath,
		K8sServiceAccountTokenPath: cfg.K8sServiceAccountTokenPath,
		Token:                      cfg.VaultToken,
		Role:                       cfg.VaultRole,
		Logger:                     cfg.Logger,
	})
	if err != nil {
		return nil, err
	}

	forConns, err := vault.NewVaultWithKubernetesAuth(ctx, grants["connections"], &vault.Config{
		Addr:                       grants["connections"].BackendAddr,
		K8sAuthMountPath:           cfg.VaultAuthMountPath,
		K8sServiceAccountTokenPath: cfg.K8sServiceAccountTokenPath,
		Token:                      cfg.VaultToken,
		Role:                       cfg.VaultRole,
		Logger:                     cfg.Logger,
	})
	if err != nil {
		return nil, err
	}

	cm := connvault.New(forConns)

	om := configmap.New(kc, cfg.Namespace)
	stm := stconfigmap.New(kc, cfg.Namespace)
	mm := task.NewKubernetesMetadataManager(kc, cfg.Namespace)
	spm := task.NewKubernetesSpecManager(kc, cfg.Namespace)
	cdm := cconfigmap.New(kc, cfg.Namespace)

	return &DefaultManagerFactory{
		sm:  NewEncodingSecretManager(sm),
		cm:  NewEncodingConnectionManager(cm),
		om:  om,
		stm: NewEncodeDecodingStateManager(stm),
		mm:  mm,
		spm: spm,
		cdm: cdm,
	}, nil
}

type developmentPreConfig struct {
	Secrets      map[string]string            `yaml:"secrets"`
	Connections  map[string]map[string]string `yaml:"connections"`
	TaskMetadata map[string]struct {
		Name string `yaml:"name"`
	} `yaml:"taskMetadata"`
	TaskSpecs        map[string]string `yaml:"taskSpecs"`
	TaskConditionals map[string]string `yaml:"taskConditionals"`
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

	mds := make(map[string]*task.Metadata, len(preCfg.TaskMetadata))
	for name, md := range preCfg.TaskMetadata {
		mds[name] = &task.Metadata{
			Hash: sha1.Sum([]byte(md.Name)),
		}
	}

	sm := smemory.New(preCfg.Secrets)
	cm := comemory.New(preCfg.Connections)
	om := omemory.New()
	stm := stmemory.New()
	mm := task.NewPreconfiguredMetadataManager(mds)
	spm := task.NewPreconfiguredSpecManager(preCfg.TaskSpecs)
	cdm := cmemory.New(preCfg.TaskConditionals)

	return &DefaultManagerFactory{
		sm:  NewEncodingSecretManager(sm),
		cm:  NewEncodingConnectionManager(cm),
		om:  om,
		stm: NewEncodeDecodingStateManager(stm),
		mm:  mm,
		spm: spm,
		cdm: cdm,
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
