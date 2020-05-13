package authenticate_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/stretchr/testify/require"
)

func TestChainIntermediary(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		Name          string
		Intermediary  *authenticate.ChainIntermediary
		ExpectedRaw   authenticate.Raw
		ExpectedError error
	}{
		{
			Name: "Multiple chaining",
			Intermediary: authenticate.NewChainIntermediary(
				authenticate.Raw("test"),
				authenticate.ChainIntermediaryFunc(func(ctx context.Context, prev authenticate.Raw) (authenticate.Intermediary, error) {
					return authenticate.Raw(fmt.Sprintf("%s-step-a", string(prev))), nil
				}),
				authenticate.ChainIntermediaryFunc(func(ctx context.Context, prev authenticate.Raw) (authenticate.Intermediary, error) {
					return authenticate.Raw(fmt.Sprintf("%s-step-b", string(prev))), nil
				}),
			),
			ExpectedRaw: authenticate.Raw("test-step-a-step-b"),
		},
		{
			Name: "Initial error",
			Intermediary: authenticate.NewChainIntermediary(
				authenticate.IntermediaryFunc(func(ctx context.Context, state *authenticate.Authentication) (authenticate.Raw, error) {
					return nil, errors.New("initial")
				}),
				authenticate.ChainIntermediaryFunc(func(ctx context.Context, prev authenticate.Raw) (authenticate.Intermediary, error) {
					return authenticate.Raw(fmt.Sprintf("%s-step-a", string(prev))), nil
				}),
				authenticate.ChainIntermediaryFunc(func(ctx context.Context, prev authenticate.Raw) (authenticate.Intermediary, error) {
					return authenticate.Raw(fmt.Sprintf("%s-step-b", string(prev))), nil
				}),
			),
			ExpectedError: errors.New("initial"),
		},
		{
			Name: "Chain error",
			Intermediary: authenticate.NewChainIntermediary(
				authenticate.Raw("test"),
				authenticate.ChainIntermediaryFunc(func(ctx context.Context, prev authenticate.Raw) (authenticate.Intermediary, error) {
					return nil, errors.New("step-a")
				}),
				authenticate.ChainIntermediaryFunc(func(ctx context.Context, prev authenticate.Raw) (authenticate.Intermediary, error) {
					return authenticate.Raw(fmt.Sprintf("%s-step-b", string(prev))), nil
				}),
			),
			ExpectedError: errors.New("step-a"),
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			raw, err := test.Intermediary.Next(ctx, authenticate.NewAuthentication())
			require.Equal(t, test.ExpectedError, err)
			require.Equal(t, test.ExpectedRaw, raw)
		})
	}
}
