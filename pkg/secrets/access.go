package secrets

import (
	"context"

	v1 "k8s.io/api/core/v1"
)

// AccessGranter grants access to secrets.
// The actual implementations of AccessGranter know how to craft
// policies for the secrets backend they were created for.
type AccessGranter interface {
	// GrantAccessForPath grants access to secrets stored under path.
	GrantAccessForPaths(ctx context.Context, paths map[string]string) (map[string]*AccessGrant, error)
}

// AccessRevoker revokes access to secrets created with an AccessGranter.
type AccessRevoker interface {
	// RevokeAllAccess deletes scoped readonly policies and any auth
	// roles created for accessing secrets.
	RevokeAllAccess(ctx context.Context) error
}

type AuthAccessManager interface {
	ServiceAccountAccessGranter(*v1.ServiceAccount) (AccessGranter, error)
	ServiceAccountAccessRevoker(*v1.ServiceAccount) (AccessRevoker, error)
}
