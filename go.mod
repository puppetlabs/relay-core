module github.com/puppetlabs/nebula-tasks

go 1.12

require (
	cloud.google.com/go v0.43.0
	github.com/Azure/azure-sdk-for-go v32.0.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/to v0.2.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.1.0 // indirect
	github.com/Azure/go-autorest/tracing v0.2.0 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Microsoft/go-winio v0.4.13 // indirect
	github.com/aws/aws-sdk-go v1.22.0
	github.com/blang/semver v3.6.1+incompatible // indirect
	github.com/denverdino/aliyungo v0.0.0-20190730233141-daf435c01246 // indirect
	github.com/digitalocean/godo v1.1.1-0.20170706200301-34840385860d // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/engine-api v0.4.0 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/go-ini/ini v1.44.0 // indirect
	github.com/google/go-containerregistry v0.0.0-20190729175742-ef12d49c8daf // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gophercloud/gophercloud v0.3.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/vault/api v1.0.2
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/knative/pkg v0.0.0-20190624141606-d82505e6c5b4 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.1.0
	github.com/markbates/inflect v1.0.4 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/miekg/dns v1.1.15 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/pkg/sftp v1.10.0 // indirect
	github.com/prometheus/common v0.6.0 // indirect
	github.com/puppetlabs/errawr-gen v1.0.0
	github.com/puppetlabs/errawr-go/v2 v2.1.0
	github.com/puppetlabs/horsehead v1.4.0
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/tektoncd/pipeline v0.4.0
	github.com/vmware/govmomi v0.21.0 // indirect
	go.uber.org/zap v1.9.1 // indirect
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7 // indirect
	golang.org/x/sys v0.0.0-20190626221950-04f50cda93cb // indirect
	google.golang.org/api v0.7.0
	google.golang.org/grpc v1.22.1 // indirect
	gopkg.in/gcfg.v1 v1.2.3 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.44.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	k8s.io/api v0.0.0-20190805141119-fdd30b57c827
	k8s.io/apiextensions-apiserver v0.0.0-20190802061903-25691aabac0a // indirect
	k8s.io/apimachinery v0.0.0-20190806215851-162a2dabc72f
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/code-generator v0.0.0-20190802060428-9d064f9b93f4
	k8s.io/csi-api v0.0.0-20190313123203-94ac839bf26c // indirect
	k8s.io/klog v0.3.3
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1 // indirect
)

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.2
	github.com/mholt/caddy => github.com/caddyserver/caddy v1.0.1
	github.com/miekg/coredns/middleware/pkg/dnsutil => github.com/coredns/coredns/middleware/pkg/dnsutil v0.0.0-20170910182647-1b60688dc8f7
	k8s.io/api => k8s.io/api v0.0.0-20190805141119-fdd30b57c827
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190805143126-cdb999c96590
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190805142138-368b2058237c
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190805141520-2fe0317bcee0
)
