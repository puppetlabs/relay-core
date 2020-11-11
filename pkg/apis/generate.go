//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:preserveUnknownFields=false object rbac:roleName=relay-install-operator-relaycores paths=./... output:crd:artifacts:config=../../manifests/resources
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen rbac:roleName=relay-install-operator paths=../install/controller/... output:dir=../../manifests/resources

package apis
