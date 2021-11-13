package app

import (
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/ownerext"
	"github.com/puppetlabs/leg/k8sutil/pkg/norm"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	webhookTLSDirPath     = "/var/run/secrets/puppet/relay/webhook-tls"
	jwtSigningKeyDirPath  = "/var/run/secrets/puppet/relay/jwt"
	jwtSigningKeyPath     = "/var/run/secrets/puppet/relay/jwt/private-key.pem"
	metadataAPITLSDirPath = "/var/run/secrets/puppet/relay/tls"
)

var DependencyManager = ownerext.NewManager("installer.relay.sh/dependency-of")

// TODO share this with pkg/operator/app
func SuffixObjectKey(key client.ObjectKey, suffix string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: key.Namespace,
		Name:      norm.MetaNameSuffixed(key.Name, "-"+suffix),
	}
}
