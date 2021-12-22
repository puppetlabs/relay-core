//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen rbac:roleName=relay-install-operator-relaycores crd:preserveUnknownFields=false object paths=./... output:crd:artifacts:config=manifests/installer

package install
