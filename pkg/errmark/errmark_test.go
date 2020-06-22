package errmark_test

import (
	"fmt"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/errmark"
	"github.com/stretchr/testify/require"
)

func TestIfAll(t *testing.T) {
	tests := []struct {
		Error        error
		IfFuncs      []errmark.IfFunc
		ExpectedCall bool
	}{
		{
			Error:        fmt.Errorf("boom"),
			IfFuncs:      []errmark.IfFunc{errmark.IfNotTransient, errmark.IfNotUser},
			ExpectedCall: true,
		},
		{
			Error:        errmark.MarkUser(fmt.Errorf("boom")),
			IfFuncs:      []errmark.IfFunc{errmark.IfNotTransient, errmark.IfNotUser},
			ExpectedCall: false,
		},
		{
			Error:        errmark.MarkUser(fmt.Errorf("boom")),
			IfFuncs:      []errmark.IfFunc{errmark.IfNotUser, errmark.IfNotTransient},
			ExpectedCall: false,
		},
		{
			Error: errmark.MapLast(errmark.MarkUser(fmt.Errorf("boom")), func(err error) error {
				return fmt.Errorf("inner: %+v", err)
			}),
			IfFuncs:      []errmark.IfFunc{errmark.IfUser},
			ExpectedCall: true,
		},
	}
	for _, test := range tests {
		t.Run(test.Error.Error(), func(t *testing.T) {
			var called bool
			errmark.IfAll(test.Error, test.IfFuncs, func(err error) {
				called = true
			})
			require.Equal(t, test.ExpectedCall, called)
		})
	}
}
