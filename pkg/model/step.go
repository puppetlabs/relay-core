package model

import (
	"crypto/sha1"
	"path"
)

type Step struct {
	Run  Run
	Name string
}

var _ Action = &Step{}

func (*Step) Type() ActionType {
	return ActionTypeStep
}

func (s *Step) Hash() Hash {
	return Hash(sha1.Sum([]byte(path.Join("runs", s.Run.ID, s.Type().Plural, s.Name))))
}

func IfStep(action Action, fn func(s *Step)) {
	if step, ok := action.(*Step); ok {
		fn(step)
	}
}
