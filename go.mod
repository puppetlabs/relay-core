module github.com/puppetlabs/nebula-tasks

go 1.13

require (
	cloud.google.com/go/storage v1.1.2 // indirect
	github.com/frankban/quicktest v1.7.3 // indirect
	github.com/golang/groupcache v0.0.0-20191002201903-404acd9df4cc // indirect
	github.com/google/uuid v1.1.1
	github.com/hashicorp/vault/api v1.0.4
	github.com/inconshreveable/log15 v0.0.0-20180818164646-67afb5ed74ec
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/puppetlabs/errawr-gen v1.0.1
	github.com/puppetlabs/errawr-go/v2 v2.2.0
	github.com/puppetlabs/horsehead/v2 v2.7.0
	github.com/puppetlabs/nebula-libs/storage/file/v2 v2.0.0
	github.com/puppetlabs/nebula-libs/storage/gcs/v2 v2.0.0
	github.com/puppetlabs/nebula-sdk v1.9.2
	github.com/stretchr/testify v1.4.0
	github.com/tektoncd/pipeline v0.11.0-rc2
	gonum.org/v1/gonum v0.6.2 // indirect
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.0
	k8s.io/code-generator v0.17.1
	k8s.io/klog v1.0.0
	knative.dev/pkg v0.0.0-20200207155214-fef852970f43
)

// Knative deps

replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.5
	knative.dev/pkg => knative.dev/pkg v0.0.0-20190909195211-528ad1c1dd62
	knative.dev/pkg/vendor/github.com/spf13/pflag => github.com/spf13/pflag v1.0.5
)

// Pin k8s deps to 1.15.5

replace (
	k8s.io/api => k8s.io/api v0.0.0-20191016110246-af539daaa43a
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115701-31ade1b30762
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016110837-54936ba21026
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b
)
