module github.com/puppetlabs/nebula-tasks

go 1.13

require (
	cloud.google.com/go/storage v1.1.2 // indirect
	github.com/containerd/continuity v0.0.0-20200413184840-d3ef23f19fbb // indirect
	github.com/frankban/quicktest v1.7.3 // indirect
	github.com/gofrs/flock v0.7.1
	github.com/golang/groupcache v0.0.0-20191002201903-404acd9df4cc // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-retryablehttp v0.6.2
	github.com/hashicorp/vault v1.4.1
	github.com/hashicorp/vault-plugin-auth-jwt v0.6.2
	github.com/hashicorp/vault-plugin-secrets-kv v0.5.5
	github.com/hashicorp/vault/api v1.0.5-0.20200317185738-82f498082f02
	github.com/hashicorp/vault/sdk v0.1.14-0.20200429182704-29fce8f27ce4
	github.com/inconshreveable/log15 v0.0.0-20180818164646-67afb5ed74ec
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4 // indirect
	github.com/opencontainers/runc v1.0.0-rc6 // indirect
	github.com/puppetlabs/errawr-gen v1.0.1
	github.com/puppetlabs/errawr-go/v2 v2.2.0
	github.com/puppetlabs/horsehead/v2 v2.7.0
	github.com/puppetlabs/nebula-libs/storage/file/v2 v2.0.0
	github.com/puppetlabs/nebula-libs/storage/gcs/v2 v2.0.0
	github.com/puppetlabs/nebula-sdk v1.12.3
	github.com/rancher/remotedialer v0.2.5
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.5.1
	github.com/tektoncd/pipeline v0.12.0
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2
	gopkg.in/square/go-jose.v2 v2.4.1
	gopkg.in/yaml.v3 v3.0.0-20191010095647-fc94e3f71652
	k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/klog v1.0.0
	knative.dev/caching v0.0.0-20200116200605-67bca2c83dfa
	knative.dev/pkg v0.0.0-20200306230727-a56a6ea3fa56
	knative.dev/serving v0.13.0
	sigs.k8s.io/controller-runtime v0.5.2
	sigs.k8s.io/controller-tools v0.2.8
)
