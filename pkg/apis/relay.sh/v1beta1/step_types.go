package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Log struct {
	// Name of the associated log.
	//
	// +optional
	Name string `json:"name"`

	// Context of the associated log.
	//
	// +optional
	Context string `json:"context"`
}

type StepOutput struct {
	// Name is the name of this output.
	Name string `json:"name"`

	// Sensitive is whether this output contains sensitive or privileged data.
	// If this output is sensitive, the value will not be set.
	//
	// +optional
	Sensitive bool `json:"sensitive"`

	// Value is the value provided by the step for the output.
	//
	// +optional
	Value *Unstructured `json:"value"`
}

// WhenEvaluationStepMessageSource indicates that a step message came from the
// runtime processing of its when conditions.
type WhenEvaluationStepMessageSource struct {
	// Path is the location within the when condition to the expression that
	// originated the message, expressed as a JSON Pointer (RFC 6901).
	//
	// If not specified, or empty, the path refers to the entire when condition.
	Path string `json:"path"`

	// Expression is the actual data of the expression that failed to evaluate,
	// copied verbatim from the workflow definition and relative to the path
	// field, if it is available.
	//
	// +optional
	Expression *Unstructured `json:"expression,omitempty"`
}

// SpecValidationStepMessageSource indicates that a step message came from the
// runtime validation of its spec.
type SpecValidationStepMessageSource struct {
	// Path is the location within the spec to the expression that originated
	// the message, expressed as a JSON Pointer (RFC 6901).
	//
	// If not specified, or empty, the path refers to the entire spec.
	Path string `json:"path"`

	// Expression is the actual data of the expression that failed to validate,
	// copied verbatim from the workflow definition and relative to the path
	// field, if it is available.
	//
	// +optional
	Expression *Unstructured `json:"expression,omitempty"`

	// Schema is the JSON Schema that caused the validation error relative to
	// the path field.
	//
	// +optional
	Schema *Unstructured `json:"schema,omitempty"`
}

// LogStepMessageSource indicates that a step message originated in the
// execution of a step's container itself.
type LogStepMessageSource struct{}

// StepMessageSource is the origin of a step message.
//
// At most one of the fields may be specified at any given time. The behavior is
// undefined if no source is specified or if more than one is specified.
type StepMessageSource struct {
	// WhenEvaluation is a source used by runtime processing of when conditions.
	//
	// +optional
	WhenEvaluation *WhenEvaluationStepMessageSource `json:"whenEvaluation,omitempty"`

	// SpecValidation is a source used by the runtime validation of the spec.
	//
	// +optional
	SpecValidation *SpecValidationStepMessageSource `json:"specValidation,omitempty"`

	// Log is a source used by the logging APIs exposed to step authors.
	//
	// +optional
	Log *LogStepMessageSource `json:"log,omitempty"`
}

type StepMessageSeverity string

const (
	// StepMessageSeverityTrace is for highly detailed messages that should
	// normally be hidden but could be inspected for debugging purposes.
	StepMessageSeverityTrace StepMessageSeverity = "Trace"

	// StepMessageSeverityInformational is for messages that alert a user to a
	// particular condition in a step's execution that does not require any user
	// action.
	StepMessageSeverityInformational StepMessageSeverity = "Informational"

	// StepMessageSeverityWarning is for important messages reporting on an
	// anomolous condition encountered by a step, but that do not necessarily
	// require user intervention.
	StepMessageSeverityWarning StepMessageSeverity = "Warning"

	// StepMessageSeverityError is for messages that indicate the step
	// encountered an uncorrectable situation. The step will usually fail, and
	// the user may need to intervene before retrying.
	StepMessageSeverityError StepMessageSeverity = "Error"
)

type StepMessage struct {
	// Source is the origin of the message.
	Source StepMessageSource `json:"source"`

	// Severity indicates the importance of this message.
	//
	// +kubebuilder:validation:Enum=Trace;Informational;Warning;Error
	Severity StepMessageSeverity `json:"severity"`

	// ObservationTime is the time that the causal event for the message
	// occurred.
	//
	// This may be different from the time the message was actually added to the
	// Run object.
	ObservationTime metav1.Time `json:"observationTime"`

	// Short is an abbreviated description of the message (if available) that
	// could be shown as a title or in space-constrained user interfaces.
	//
	// If not specified, the first few characters of the details will be used
	// instead.
	//
	// +kubebuilder:validation:MaxLength=24
	// +optional
	Short string `json:"short,omitempty"`

	// Details is the text content of the message to show to an interested user.
	//
	// +kubebuilder:validation:MaxLength=1024
	Details string `json:"details"`
}

type StepConditionType string

const (
	// StepCompleted indicates whether a step has finished executing.
	StepCompleted StepConditionType = "Completed"

	// StepSkipped indicates a step has been skipped.
	StepSkipped StepConditionType = "Skipped"

	// StepSucceeded indicates a step has succeeded.
	StepSucceeded StepConditionType = "Succeeded"
)

type StepCondition struct {
	Condition `json:",inline"`

	// Type is the identifier for this condition.
	//
	// +kubebuilder:validation:Enum=Completed;Skipped;Succeeded
	Type StepConditionType `json:"type"`
}

type StepStatus struct {
	// Name is the name of this step.
	Name string `json:"name"`

	// Outputs are each of the outputs provided by this step, if available.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	Outputs []*StepOutput `json:"outputs,omitempty"`

	// +optional
	// +listType=map
	// +listMapKey=name
	Decorators []*Decorator `json:"decorators,omitempty"`

	// Messages provide additional human-oriented context information about a
	// step's execution.
	//
	// +optional
	Messages []*StepMessage `json:"messages,omitempty"`

	// StartTime is the time this step began executing.
	//
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time this step ended, whether successful or not.
	//
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// InitializationTime is the time taken to initialize the step.
	//
	// +optional
	InitializationTime *metav1.Time `json:"initTime,omitempty"`

	// Associated logs for this step.
	//
	// +optional
	Logs []*Log `json:"logs,omitempty"`

	// Conditions are the possible observable conditions for this step.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []StepCondition `json:"conditions,omitempty"`
}
