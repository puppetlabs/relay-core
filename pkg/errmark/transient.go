package errmark

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
)

type TransientRule func(err error) bool

func TransientRuleExact(err, want error) bool {
	return err == want
}

func TransientAll(rules ...TransientRule) TransientRule {
	return func(err error) bool {
		for _, rule := range rules {
			if !rule(err) {
				return false
			}
		}

		return true
	}
}

var TransientAlways = TransientAll()

func TransientAny(rules ...TransientRule) TransientRule {
	return func(err error) bool {
		for _, rule := range rules {
			if rule(err) {
				return true
			}
		}

		return false
	}
}

var TransientIfConflict = TransientAny(
	errors.IsConflict,
	errors.IsAlreadyExists,
)

var TransientIfTimeout = TransientAny(
	errors.IsTimeout,
	errors.IsServerTimeout,
)

var TransientIfForbidden TransientRule = errors.IsForbidden

var TransientDefault = TransientAny(
	TransientIfConflict,
	TransientIfTimeout,
)

func TransientPredicate(rule TransientRule, pred func() bool) TransientRule {
	return func(err error) bool {
		if pred() {
			return rule(err)
		}

		return false
	}
}

type TransientError struct {
	Delegate error
}

var _ Marker = &TransientError{}

func (te *TransientError) Error() string {
	return fmt.Sprintf("transient: %+v", te.Delegate)
}

func (te *TransientError) Map(fn func(err error) error) error {
	te.Delegate = fn(te.Delegate)
	return te
}

func MarkTransient(err error, rules ...TransientRule) error {
	return MapFirst(err, func(err error) error {
		if _, ok := err.(*TransientError); ok {
			return err
		} else if TransientAny(rules...)(err) {
			return &TransientError{Delegate: err}
		}

		return err
	})
}

func IfTransient(err error, fn func(err error)) {
	if te, ok := err.(*TransientError); ok {
		fn(te.Delegate)
	}
}

func IfNotTransient(err error, fn func(err error)) {
	if _, ok := err.(*TransientError); !ok {
		fn(err)
	}
}
