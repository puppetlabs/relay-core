//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:preserveUnknownFields=false object paths=../apis/relay.sh/... output:crd:artifacts:config=../../manifests/resources
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:preserveUnknownFields=false object paths=../apis/install.relay.sh/... output:crd:artifacts:config=../../manifests/installer

package apis
