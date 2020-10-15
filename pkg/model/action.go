package model

// ActionPodStepContainerName is used to set the container name for step and
// trigger on all action pods.
//
// TODO: move all references of "step-step" to this const
const ActionPodStepContainerName = "step-step"

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
