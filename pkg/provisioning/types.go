package provisioning

import (
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

var PlatformAdapters = map[Platform]func(*models.K8sProvisionerSpec, string) K8sClusterAdapter{
	PlatformGCP: func(spec *models.K8sProvisionerSpec, workdir string) K8sClusterAdapter {
		support := gcpsupport.NewKopsSupport(gcpsupport.KopsSupportConfig{
			ProjectID:                spec.Project,
			ServiceAccountFileBase64: spec.Credentials.GCPServiceAccountFile,
			StateStoreName:           spec.StateStoreName,
			WorkDir:                  workdir,
		})
		return cloud.NewK8sClusterAdapter(spec, support, workdir)
	},
	PlatformAWS: func(spec *models.K8sProvisionerSpec, workdir string) K8sClusterAdapter {
		support := awssupport.NewKopsSupport(awssupport.KopsSupportConfig{
			AccessKeyID:     spec.Credentials.AWSAccessKeyID,
			SecretAccessKey: spec.Credentials.AWSSecretAccessKey,
			StateStoreName:  spec.StateStoreName,
			Region:          spec.Region,
			SSHPublicKey:    spec.Credentials.SSHPublicKey,
			WorkDir:         workdir,
		})
		return cloud.NewK8sClusterAdapter(spec, support, workdir)
	},
}
