package obj

import (
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
)

func TransientIfRequired(err error) bool {
	return errmark.MarkTransientIf(err, errmark.RuleExact(ErrRequired))
}
