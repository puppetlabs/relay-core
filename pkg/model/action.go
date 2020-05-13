package model

type ActionType struct {
	Singular string
	Plural   string
}

var (
	ActionTypeStep    = ActionType{Singular: "step", Plural: "steps"}
	ActionTypeTrigger = ActionType{Singular: "trigger", Plural: "triggers"}
)

type Action interface {
	Type() ActionType
	Hash() Hash
}
