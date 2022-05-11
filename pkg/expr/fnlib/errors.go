package fnlib

import (
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/gvalutil/pkg/eval"
)

var notExistsRule = errmark.RuleAny(
	errmark.RuleType(&eval.IndexOutOfBoundsError{}),
	errmark.RuleType(&eval.UnknownKeyError{}),
	errmark.RuleType(&eval.UnknownFieldError{}),
)
