package provisioning

import (
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/cloud"
	awssupport "github.com/puppetlabs/nebula-tasks/pkg/provisioning/cloud/aws/support"
	gcpsupport "github.com/puppetlabs/nebula-tasks/pkg/provisioning/cloud/gcp/support"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/models"
)

type Platform int

const (
	PlatformGCP Platform = iota
	PlatformAWS
)

var PlatformMapping = map[string]Platform{
	"gcp": PlatformGCP,
	"aws": PlatformAWS,
}

var PlatformAdapters = map[Platform]func(spec *models.K8sProvisionerSpec) (K8sClusterAdapter, errors.Error){
	PlatformGCP: func(spec *models.K8sProvisionerSpec) (K8sClusterAdapter, errors.Error) {
		support := gcpsupport.NewKopsSupport(gcpsupport.KopsSupportConfig{
			ProjectID:                spec.Project,
			ServiceAccountFileBase64: spec.Credentials.GCPServiceAccountFile,
			StateStoreName:           spec.StateStoreName,
		})
		return cloud.NewK8sClusterAdapter(spec, support)
	},
	PlatformAWS: func(spec *models.K8sProvisionerSpec) (K8sClusterAdapter, errors.Error) {
		support := awssupport.NewKopsSupport(awssupport.KopsSupportConfig{
			AccessKeyID:     spec.Credentials.AWSAccessKeyID,
			SecretAccessKey: spec.Credentials.AWSSecretAccessKey,
			StateStoreName:  spec.StateStoreName,
			Region:          spec.Region,
			SSHPublicKey:    spec.Credentials.SSHPublicKey,
		})
		return cloud.NewK8sClusterAdapter(spec, support)
	},
}
