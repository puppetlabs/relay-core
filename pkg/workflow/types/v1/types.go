package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/puppetlabs/horsehead/v2/stringutil"
	"github.com/puppetlabs/relay-core/pkg/expr/parse"
	"github.com/puppetlabs/relay-core/pkg/expr/serialize"
	"github.com/puppetlabs/relay-core/pkg/manager/input"
	"github.com/robfig/cron/v3"
)

type WorkflowStepType string

type WorkflowTriggerSourceType string

type WorkflowStepApprovalStatus string

func (ws WorkflowStepApprovalStatus) String() string {
	return string(ws)
}

const (
	WorkflowStepTypeContainer WorkflowStepType = "container"
	WorkflowStepTypeApproval  WorkflowStepType = "approval"
	WorkflowStepTypeUnknown   WorkflowStepType = "unknown" // Unrecognized deserialized yaml values
)

const (
	WorkflowStepApprovalWaiting  WorkflowStepApprovalStatus = "waiting"
	WorkflowStepApprovalApproved WorkflowStepApprovalStatus = "approved"
	WorkflowStepApprovalRejected WorkflowStepApprovalStatus = "rejected"
)

func (wtst WorkflowTriggerSourceType) String() string {
	return string(wtst)
}

const (
	WorkflowTriggerSourceTypePush     WorkflowTriggerSourceType = "push"
	WorkflowTriggerSourceTypeSchedule WorkflowTriggerSourceType = "schedule"
	WorkflowTriggerSourceTypeWebhook  WorkflowTriggerSourceType = "webhook"
)

type YAMLWorkflowData struct {
	Version     string                `yaml:"version" json:"version"`
	APIVersion  string                `yaml:"apiVersion" json:"apiVersion"`
	Description string                `yaml:"description" json:"description"`
	Parameters  WorkflowParameters    `yaml:"parameters" json:"parameters,omitempty"`
	Steps       []YAMLWorkflowStep    `yaml:"steps" json:"steps"`
	Triggers    []YAMLWorkflowTrigger `yaml:"triggers" json:"triggers"`
}

type YAMLWorkflowTrigger struct {
	Name    string                     `yaml:"name" json:"name"`
	Source  YAMLWorkflowTriggerSource  `yaml:"source" json:"source"`
	Binding YAMLWorkflowTriggerBinding `yaml:"binding" json:"binding"`
	When    serialize.YAMLTree         `yaml:"when" json:"when,omitempty"`
}

type YAMLContainerMixin struct {
	Image     string                        `yaml:"image" json:"image,omitempty"`
	Spec      map[string]serialize.YAMLTree `yaml:"spec" json:"spec,omitempty"`
	Input     []string                      `yaml:"input" json:"input,omitempty"`
	InputFile string                        `yaml:"inputFile" json:"inputFile,omitempty"`
	Command   string                        `yaml:"command" json:"command,omitempty"`
	Args      []string                      `yaml:"args" json:"args,omitempty"`
}

type YAMLWorkflowStep struct {
	Name               string `json:"name"`
	Type               string `json:"type,omitempty"`
	YAMLContainerMixin `yaml:",inline"`
	DependsOn          stringutil.StringArray `yaml:"dependsOn" json:"depends_on,omitempty"`
	When               serialize.YAMLTree     `yaml:"when" json:"when,omitempty"`
}

type YAMLWorkflowTriggerBinding struct {
	Key        serialize.YAMLTree            `yaml:"key" json:"key,omitempty"`
	Parameters map[string]serialize.YAMLTree `yaml:"parameters" json:"parameters,omitempty"`
}

type YAMLWorkflowTriggerSource struct {
	Type                              string `yaml:"type" json:"type,omitempty"`
	YAMLPushWorkflowTriggerSource     `yaml:",inline"`
	YAMLScheduleWorkflowTriggerSource `yaml:",inline"`
	YAMLWebhookWorkflowTriggerSource  `yaml:",inline"`
}

type YAMLPushWorkflowTriggerSource struct {
	// TODO: Implement serialization for schema field types.
	Schema map[string]interface{} `yaml:"schema" json:"schema,omitempty"`
}

type YAMLScheduleWorkflowTriggerSource struct {
	Schedule string `yaml:"schedule" json:"schedule,omitempty"`
}

type YAMLWebhookWorkflowTriggerSource struct {
	YAMLContainerMixin `yaml:",inline"`
}

type WorkflowDataTriggerSource struct {
	Type    string `yaml:"type" json:"type,omitempty"`
	Variant WorkflowTriggerSourceVariant
}

type PushWorkflowTriggerSource struct {
	// TODO: Implement serialization for schema field types.
	Schema map[string]interface{} `yaml:"schema" json:"schema,omitempty"`
}

type ScheduleWorkflowTriggerSource struct {
	Schedule string `yaml:"schedule" json:"schedule,omitempty"`
}

func (swts *ScheduleWorkflowTriggerSource) Next(from time.Time) (time.Time, error) {
	sched, err := cron.ParseStandard(swts.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("TODO")
		//return time.Time{}, errors.NewWorkflowTriggerScheduleFormatError(swts.Schedule, err.Error())
	}

	return sched.Next(from), nil
}

type WebhookWorkflowTriggerSource struct {
	ContainerMixin
}

type WorkflowTriggerSourceVariant interface {
	workflowTriggerSourceVariant()
	TriggerSourceType() string
}

func (*PushWorkflowTriggerSource) workflowTriggerSourceVariant() {}

func (wtsv *PushWorkflowTriggerSource) TriggerSourceType() string {
	return "push"
}

func (*ScheduleWorkflowTriggerSource) workflowTriggerSourceVariant() {}

func (wtsv *ScheduleWorkflowTriggerSource) TriggerSourceType() string {
	return "schedule"
}

func (*WebhookWorkflowTriggerSource) workflowTriggerSourceVariant() {}

func (wtsv *WebhookWorkflowTriggerSource) TriggerSourceType() string {
	return "webhook"
}

type WorkflowData struct {
	Version     string                 `yaml:"version" json:"version"`
	Description string                 `yaml:"description" json:"description"`
	Parameters  WorkflowParameters     `yaml:"parameters" json:"parameters,omitempty"`
	Steps       []*WorkflowStep        `yaml:"steps" json:"steps"`
	Triggers    []*WorkflowDataTrigger `yaml:"triggers" json:"triggers"`
}

type WorkflowDataTrigger struct {
	Name    string                      `yaml:"name" json:"name"`
	Source  *WorkflowDataTriggerSource  `yaml:"source" json:"source,omitempty"`
	Binding *WorkflowDataTriggerBinding `yaml:"binding" json:"binding,omitempty"`
	When    serialize.JSONTree          `yaml:"when" json:"when,omitempty"`
}

type WorkflowDataTriggerBinding struct {
	Key        serialize.JSONTree `yaml:"key" json:"key,omitempty"`
	Parameters ExpressionMap      `yaml:"parameters" json:"parameters,omitempty"`
}

type WorkflowParameters map[string]*WorkflowParameter

type WorkflowParameter struct {
	Default     interface{} `json:"default"`
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type,omitempty"`
}

type WorkflowRunParameters map[string]*WorkflowRunParameter

type WorkflowRunParameter struct {
	Value interface{} `json:"value"`
}

type ExpressionMap map[string]serialize.JSONTree

func (em ExpressionMap) AsParseTree() parse.Tree {
	m := make(map[string]interface{}, len(em))
	for k, v := range em {
		m[k] = v.Tree
	}
	return parse.Tree(m)
}

type WorkflowStepVariant interface {
	// This private marker method prevents out-of-package implementation of this type, making it an actual variant type.
	workflowStepVariant()
	StepType() string
}

type ContainerMixin struct {
	Image     string        `yaml:"image" json:"image"`
	Spec      ExpressionMap `yaml:"spec" json:"spec,omitempty"`
	InputFile string        `yaml:"inputFile" json:"inputFile,omitempty"`
	Input     []string      `yaml:"input" json:"input,omitempty"`
	Command   string        `yaml:"command" json:"command,omitempty"`
	Args      []string      `yaml:"args" json:"args,omitempty"`

	inputFileLoaded bool
}

func (c *ContainerMixin) LoadInputFile(ctx context.Context, im input.FileManager) error {
	if c.inputFileLoaded || c.InputFile == "" {
		return nil
	}

	inputFileReader, err := im.GetByURL(ctx, c.InputFile)
	if err != nil {
		return err
	}

	content, gerr := ioutil.ReadAll(inputFileReader)
	if gerr != nil {
		return fmt.Errorf("TODO")
		//return errors.NewWorkflowInputFileContentError().WithCause(gerr)
	}

	c.Input = []string{string(content)}
	c.inputFileLoaded = true
	return nil
}

type ContainerWorkflowStep struct {
	ContainerMixin
}

func (*ContainerWorkflowStep) workflowStepVariant() {}

func (sv *ContainerWorkflowStep) StepType() string {
	return "container"
}

type ApprovalWorkflowStep struct{}

func (*ApprovalWorkflowStep) workflowStepVariant() {}

func (sv *ApprovalWorkflowStep) StepType() string {
	return "approval"
}

type WorkflowStep struct {
	Name      string             `yaml:"name" json:"name"`
	DependsOn []string           `yaml:"dependsOn" json:"depends_on"`
	When      serialize.JSONTree `yaml:"when" json:"when,omitempty"`
	Variant   WorkflowStepVariant
}

type WorkflowTriggerSource struct {
	Type    string `json:"type,omitempty"`
	Variant WorkflowTriggerSourceVariant
}

func (wts *WorkflowTriggerSource) UnmarshalJSON(data []byte) error {
	type common struct {
		Type string `json:"type"`
	}

	var c common
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}

	wts.Type = c.Type
	switch c.Type {
	case WorkflowTriggerSourceTypePush.String():
		wts.Type = c.Type
		wts.Variant = &PushWorkflowTriggerSource{}
		if err := json.Unmarshal(data, wts.Variant); err != nil {
			return err
		}
	case WorkflowTriggerSourceTypeSchedule.String():
		wts.Variant = &ScheduleWorkflowTriggerSource{}
		if err := json.Unmarshal(data, wts.Variant); err != nil {
			return err
		}
	case WorkflowTriggerSourceTypeWebhook.String():
		wts.Variant = &WebhookWorkflowTriggerSource{}
		if err := json.Unmarshal(data, wts.Variant); err != nil {
			return err
		}
	}

	return nil
}

func (wts WorkflowTriggerSource) MarshalJSON() ([]byte, error) {
	type common struct {
		Type string `json:"type"`
	}

	var data interface{}
	switch variant := wts.Variant.(type) {
	case *PushWorkflowTriggerSource:
		data = struct {
			common
			*PushWorkflowTriggerSource
		}{
			common:                    common{Type: WorkflowTriggerSourceTypePush.String()},
			PushWorkflowTriggerSource: variant,
		}
	case *ScheduleWorkflowTriggerSource:
		data = struct {
			common
			*ScheduleWorkflowTriggerSource
		}{
			common:                        common{Type: WorkflowTriggerSourceTypeSchedule.String()},
			ScheduleWorkflowTriggerSource: variant,
		}
	case *WebhookWorkflowTriggerSource:
		data = struct {
			common
			*WebhookWorkflowTriggerSource
		}{
			common:                       common{Type: WorkflowTriggerSourceTypeWebhook.String()},
			WebhookWorkflowTriggerSource: variant,
		}
	}

	return json.Marshal(data)
}

func (ws *WorkflowStep) UnmarshalJSON(data []byte) error {
	type common struct {
		Name      string             `json:"name"`
		Type      WorkflowStepType   `json:"type"`
		DependsOn []string           `json:"depends_on"`
		When      serialize.JSONTree `json:"when"`
	}

	var c common
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}

	ws.Name = c.Name
	ws.DependsOn = c.DependsOn
	ws.When = c.When

	switch c.Type {
	case WorkflowStepTypeApproval:
		ws.Variant = &ApprovalWorkflowStep{}
	default:
		ws.Variant = &ContainerWorkflowStep{}
		if err := json.Unmarshal(data, ws.Variant); err != nil {
			return err
		}
	}
	return nil
}

func (ws WorkflowStep) MarshalJSON() ([]byte, error) {
	type common struct {
		Name      string             `json:"name"`
		Type      WorkflowStepType   `json:"type"`
		DependsOn []string           `json:"depends_on"`
		When      serialize.JSONTree `json:"when"`
	}

	var es interface{}
	switch variant := ws.Variant.(type) {
	case *ContainerWorkflowStep:
		es = struct {
			common
			*ContainerWorkflowStep
		}{
			common:                common{Name: ws.Name, Type: "container", DependsOn: ws.DependsOn},
			ContainerWorkflowStep: variant,
		}
	case *ApprovalWorkflowStep:
		es = common{Name: ws.Name, Type: "approval", DependsOn: ws.DependsOn}
	}
	return json.Marshal(es)
}
