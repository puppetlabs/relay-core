module github.com/puppetlabs/nebula-tasks

go 1.12

require (
	cloud.google.com/go v0.43.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.7 // indirect
	github.com/aws/aws-sdk-go v1.25.0
	github.com/google/go-containerregistry v0.0.0-20191018211754-b77a90c667af // indirect
	github.com/google/uuid v1.1.1
	github.com/hashicorp/vault/api v1.0.4
	github.com/imdario/mergo v0.3.7
	github.com/inconshreveable/log15 v0.0.0-20180818164646-67afb5ed74ec
	github.com/markbates/inflect v1.0.4 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/puppetlabs/errawr-gen v1.0.0
	github.com/puppetlabs/errawr-go/v2 v2.1.0
	github.com/puppetlabs/horsehead v1.11.0
	github.com/puppetlabs/nebula-libs/storage/gcs v1.0.1
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/tektoncd/pipeline v0.7.0
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	google.golang.org/api v0.7.0
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gopkg.in/ini.v1 v1.48.0
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20190515023547-db5a9d1c40eb
	k8s.io/apimachinery v0.0.0-20190515023456-b74e4c97951f
	k8s.io/client-go v0.0.0-20190515063710-7b18d6600f6b
	k8s.io/code-generator v0.0.0-20191017183038-0b22993d207c
	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.14.5 // indirect
	knative.dev/pkg v0.0.0-20190925130640-d02c80dc6256
)
