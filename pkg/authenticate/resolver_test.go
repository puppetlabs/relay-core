package authenticate_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestAnyResolver(t *testing.T) {
	ctx := context.Background()

	var validators []authenticate.Validator
	var injectors []authenticate.Injector
	state := authenticate.NewInitializedAuthentication(&validators, &injectors)

	var tested int
	resolver := authenticate.NewAnyResolver([]authenticate.Resolver{
		// Resolver that should fail and no validators or injectors added.
		authenticate.ResolverFunc(func(ctx context.Context, state *authenticate.Authentication, raw authenticate.Raw) (*authenticate.Claims, error) {
			tested++

			state.AddValidator(authenticate.ValidatorFunc(func(ctx context.Context, c *authenticate.Claims) (bool, error) {
				assert.Fail(t, "erroring resolver validator called")
				return false, nil
			}))

			state.AddInjector(authenticate.InjectorFunc(func(ctx context.Context, c *authenticate.Claims) error {
				assert.Fail(t, "erroring resolver injector called")
				return nil
			}))

			return nil, &authenticate.NotFoundError{}
		}),

		// Resolver that should succeed.
		authenticate.ResolverFunc(func(ctx context.Context, state *authenticate.Authentication, raw authenticate.Raw) (*authenticate.Claims, error) {
			tested++

			state.AddValidator(authenticate.ValidatorFunc(func(ctx context.Context, c *authenticate.Claims) (bool, error) {
				return true, nil
			}))

			state.AddInjector(authenticate.InjectorFunc(func(ctx context.Context, c *authenticate.Claims) error {
				return nil
			}))

			return &authenticate.Claims{
				Claims: &jwt.Claims{
					Subject: string(raw),
				},
			}, nil
		}),
	})
	claims, err := resolver.Resolve(ctx, state, authenticate.Raw("my-auth-token"))
	require.NoError(t, err)
	require.Equal(t, "my-auth-token", claims.Subject)

	require.Equal(t, tested, 2)
	require.Len(t, validators, 1)
	require.Len(t, injectors, 1)

	for i, validator := range validators {
		ok, err := validator.Validate(ctx, claims)
		require.NoError(t, err, "validator %d", i)
		require.True(t, ok, "validator %d", i)
	}

	for i, injector := range injectors {
		require.NoError(t, injector.Inject(ctx, claims), "injector %d", i)
	}
}
