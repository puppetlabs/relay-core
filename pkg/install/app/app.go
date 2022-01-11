package app

import (
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/ownerext"
)

const (
	webhookTLSDirPath     = "/var/run/secrets/puppet/relay/webhook-tls"
	metadataAPITLSDirPath = "/var/run/secrets/puppet/relay/tls"
)

var DependencyManager = ownerext.NewManager("installer.relay.sh/dependency-of")
