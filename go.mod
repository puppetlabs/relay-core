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
	github.com/puppetlabs/nebula-sdk v1.11.0
	github.com/stretchr/testify v1.4.0
	github.com/tektoncd/pipeline v0.11.0-rc2
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.0
	k8s.io/klog v1.0.0
	knative.dev/pkg v0.0.0-20200207155214-fef852970f43
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/controller-tools v0.2.8
)
