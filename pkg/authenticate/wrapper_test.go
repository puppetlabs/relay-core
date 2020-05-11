package authenticate_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/stretchr/testify/require"
)

func TestChainWrapper(t *testing.T) {
	ctx := context.Background()

	wrapper := authenticate.NewChainWrapper(
		authenticate.WrapperFunc(func(ctx context.Context, raw authenticate.Raw) (authenticate.Raw, error) {
			return authenticate.Raw(fmt.Sprintf("%s-a", string(raw))), nil
		}),
		authenticate.WrapperFunc(func(ctx context.Context, raw authenticate.Raw) (authenticate.Raw, error) {
			return authenticate.Raw(fmt.Sprintf("%s-b", string(raw))), nil
		}),
	)
	raw, err := wrapper.Wrap(ctx, authenticate.Raw("initial"))
	require.NoError(t, err)
	require.Equal(t, authenticate.Raw("initial-a-b"), raw)
}
