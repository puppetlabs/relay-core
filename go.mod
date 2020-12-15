module github.com/puppetlabs/relay-core

go 1.13

require (
	cloud.google.com/go v0.65.0
	cloud.google.com/go/storage v1.11.0 // indirect
	contrib.go.opencensus.io/exporter/ocagent v0.7.1-0.20200907061046-05415f1de66d // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.2.1-0.20200609204449-6bcf6f8577f0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.13.2 // indirect
	github.com/PaesslerAG/gval v1.1.0 // indirect
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/aws/aws-sdk-go v1.31.12 // indirect
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/containerd/continuity v0.0.0-20200413184840-d3ef23f19fbb // indirect
	github.com/go-logr/logr v0.1.0
	github.com/gofrs/flock v0.7.1
	github.com/golang/protobuf v1.4.3
	github.com/gomarkdown/markdown v0.0.0-20200513213024-62c5e2c608cc
	github.com/google/go-containerregistry v0.1.3
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.7.4
	github.com/grpc-ecosystem/grpc-gateway v1.14.8 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1
	github.com/hashicorp/go-retryablehttp v0.6.6
	github.com/hashicorp/vault v1.4.1
	github.com/hashicorp/vault-plugin-auth-jwt v0.6.2
	github.com/hashicorp/vault-plugin-secrets-kv v0.5.5
	github.com/hashicorp/vault/api v1.0.5-0.20200317185738-82f498082f02
	github.com/hashicorp/vault/sdk v0.1.14-0.20200429182704-29fce8f27ce4
	github.com/inconshreveable/log15 v0.0.0-20180818164646-67afb5ed74ec
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4 // indirect
	github.com/lib/pq v1.3.0 // indirect
	github.com/mitchellh/mapstructure v1.3.1
	github.com/opencontainers/runc v1.0.0-rc6 // indirect
	github.com/prometheus/client_golang v1.6.0 // indirect
	github.com/puppetlabs/errawr-gen v1.0.1
	github.com/puppetlabs/errawr-go/v2 v2.2.0
	github.com/puppetlabs/horsehead/v2 v2.16.0
	github.com/puppetlabs/paesslerag-gval v1.0.2-0.20191119012647-d2c694821b5b
	github.com/puppetlabs/paesslerag-jsonpath v0.1.2-0.20201115225516-4a6f3d111e98
	github.com/puppetlabs/relay-pls v0.0.0-20201125074651-13575df50b51
	github.com/rancher/remotedialer v0.2.5
	github.com/robfig/cron/v3 v3.0.1
	github.com/shurcooL/vfsgen v0.0.0-20181202132449-6a9ea43bcacd
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tektoncd/pipeline v0.16.3
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/net v0.0.0-20200904194848-62affa334b73
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/sys v0.0.0-20200915084602-288bc346aa39 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	google.golang.org/api v0.31.0 // indirect
	google.golang.org/genproto v0.0.0-20200914193844-75d14daec038
	google.golang.org/grpc v1.34.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/square/go-jose.v2 v2.4.1
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver v0.18.4
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/klog v1.0.0
	knative.dev/caching v0.0.0-20200630172829-a78409990d76
	knative.dev/pkg v0.0.0-20200702222342-ea4d6e985ba0
	knative.dev/serving v0.16.0
	sigs.k8s.io/controller-runtime v0.5.11
	sigs.k8s.io/controller-tools v0.4.0
)

replace (
	k8s.io/api => k8s.io/api v0.17.12
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.12
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.12
	k8s.io/client-go => k8s.io/client-go v0.17.12
)
