module github.com/puppetlabs/relay-core

go 1.13

require (
	github.com/PaesslerAG/gval v1.1.0 // indirect
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/containerd/continuity v0.0.0-20200413184840-d3ef23f19fbb // indirect
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/gofrs/flock v0.7.1
	github.com/golang/protobuf v1.4.3
	github.com/gomarkdown/markdown v0.0.0-20200513213024-62c5e2c608cc
	github.com/google/go-containerregistry v0.2.1
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.7.4
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
	github.com/puppetlabs/errawr-gen v1.0.1
	github.com/puppetlabs/errawr-go/v2 v2.2.0
	github.com/puppetlabs/horsehead/v2 v2.16.0
	github.com/puppetlabs/leg/timeutil v0.2.0
	github.com/puppetlabs/paesslerag-gval v1.0.2-0.20191119012647-d2c694821b5b
	github.com/puppetlabs/paesslerag-jsonpath v0.1.2-0.20201115225516-4a6f3d111e98
	github.com/puppetlabs/relay-pls v0.0.0-20201125074651-13575df50b51
	github.com/rancher/remotedialer v0.2.5
	github.com/robfig/cron/v3 v3.0.1
	github.com/shurcooL/vfsgen v0.0.0-20181202132449-6a9ea43bcacd
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tektoncd/pipeline v0.20.1
	github.com/xeipuuv/gojsonschema v1.2.0
	go.opentelemetry.io/otel v0.16.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.16.0
	go.opentelemetry.io/otel/exporters/stdout v0.16.0
	golang.org/x/net v0.0.0-20201209123823-ac852fbbde11
	google.golang.org/grpc v1.34.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/square/go-jose.v2 v2.4.1
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.4.0
	knative.dev/caching v0.0.0-20200630172829-a78409990d76
	knative.dev/pkg v0.0.0-20210107022335-51c72e24c179
	knative.dev/serving v0.16.0
	sigs.k8s.io/controller-runtime v0.6.5
	sigs.k8s.io/controller-tools v0.4.0
)

replace (
	k8s.io/api => k8s.io/api v0.19.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.7
	k8s.io/client-go => k8s.io/client-go v0.19.7
	k8s.io/code-generator => k8s.io/code-generator v0.19.7
)
