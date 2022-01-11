module github.com/puppetlabs/relay-core

go 1.16

require (
	github.com/PaesslerAG/gval v1.1.1-0.20210429131240-4f5f9c091d78
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496 // indirect
	github.com/certifi/gocertifi v0.0.0-20180118203423-deb3ae2ef261 // indirect
	github.com/generikvault/gvalstrings v0.0.0-20180926130504-471f38f0112a
	github.com/gofrs/flock v0.8.1
	github.com/golang/protobuf v1.5.2
	github.com/gomarkdown/markdown v0.0.0-20200513213024-62c5e2c608cc
	github.com/google/go-containerregistry v0.4.1-0.20210128200529-19c2b639fab1
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/hashicorp/hcl2 v0.0.0-20191002203319-fb75b3253c80
	github.com/hashicorp/vault v1.8.2
	github.com/hashicorp/vault-plugin-auth-jwt v0.10.1
	github.com/hashicorp/vault-plugin-secrets-kv v0.9.0
	github.com/hashicorp/vault/api v1.1.2-0.20210713235431-1fc8af4c041f
	github.com/hashicorp/vault/sdk v0.2.2-0.20210825150427-9b1f4d486f5d
	github.com/inconshreveable/log15 v0.0.0-20201112154412-8562bdadbbac
	github.com/keybase/go-crypto v0.0.0-20200123153347-de78d2cb44f4 // indirect
	github.com/mitchellh/mapstructure v1.4.1
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/puppetlabs/errawr-gen v1.0.1
	github.com/puppetlabs/errawr-go/v2 v2.2.0
	github.com/puppetlabs/leg/datastructure v0.1.0
	github.com/puppetlabs/leg/encoding v0.1.0
	github.com/puppetlabs/leg/errmap v0.1.0
	github.com/puppetlabs/leg/gvalutil v0.2.0
	github.com/puppetlabs/leg/hashutil v0.1.0
	github.com/puppetlabs/leg/httputil v0.1.4
	github.com/puppetlabs/leg/instrumentation v0.1.4
	github.com/puppetlabs/leg/jsonutil v0.2.2
	github.com/puppetlabs/leg/k8sutil v0.6.4
	github.com/puppetlabs/leg/logging v0.1.0
	github.com/puppetlabs/leg/mainutil v0.1.2
	github.com/puppetlabs/leg/scheduler v0.1.4
	github.com/puppetlabs/leg/storage v0.1.1
	github.com/puppetlabs/leg/timeutil v0.4.2
	github.com/puppetlabs/pvpool v0.3.0
	github.com/puppetlabs/relay-client-go/client v0.4.4
	github.com/puppetlabs/relay-pls v0.0.0-20201125074651-13575df50b51
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tektoncd/pipeline v0.22.0
	github.com/xeipuuv/gojsonschema v1.2.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.20.0
	go.opentelemetry.io/otel/exporters/stdout v0.20.0
	go.opentelemetry.io/otel/metric v0.20.0
	golang.org/x/net v0.0.0-20210510120150-4163338589ed
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gotest.tools/gotestsum v1.7.0
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/klog/v2 v2.5.0
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	knative.dev/caching v0.0.0-20210215030244-1212288570f0
	knative.dev/pkg v0.0.0-20210215165523-84c98f3c3e7a
	knative.dev/serving v0.21.0
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/controller-tools v0.4.1
)

replace (
	k8s.io/api => k8s.io/api v0.19.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.7
	k8s.io/client-go => k8s.io/client-go v0.19.7
)
