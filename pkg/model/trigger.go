package model

import (
	"crypto/sha1"
	"path"
)

type Trigger struct {
	Name string
}

var _ Action = &Trigger{}

func (*Trigger) Type() ActionType {
	return ActionTypeTrigger
}

func (t *Trigger) Hash() Hash {
	return Hash(sha1.Sum([]byte(path.Join(t.Type().Plural, t.Name))))
}

func IfTrigger(action Action, fn func(t *Trigger)) {
	if trigger, ok := action.(*Trigger); ok {
		fn(trigger)
	}
}
