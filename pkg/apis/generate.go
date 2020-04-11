//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:preserveUnknownFields=false object paths=./... output:crd:artifacts:config=../../manifests/resources

package apis
