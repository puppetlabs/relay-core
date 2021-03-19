package v1

import (
	"bytes"
	"context"
	"io"

	"github.com/puppetlabs/relay-core/pkg/expr/serialize"
	yaml "gopkg.in/yaml.v3"
)

// Decoder takes a byte slice of serialized workflow data and decodes it into a
// WorkflowData struct
type Decoder interface {
	Decode(ctx context.Context, data []byte) (*WorkflowData, error)
}

// StremStreamingDecoder returns WorkflowData struct from a stream of data.
// Most implementations of this will take an io.Reader or io.ReadCloser and a
// Decoder.
type StreamingDecoder interface {
	// DecodeStream returns io.EOF when the reader is empty. If the error is
	// nil, then additional calls to DecodeStream will return more WorkflowData
	// objects if the implementation supports streaming multiple objects.
	DecodeStream(ctx context.Context) (*WorkflowData, error)
	Close() error
}

// YAMLDecoder expects a YAML data payload and decodes it into a WorkflowData struct.
type YAMLDecoder struct{}

func (d *YAMLDecoder) Decode(ctx context.Context, data []byte) (*WorkflowData, error) {
	ywd := &YAMLWorkflowData{}
	if err := yaml.Unmarshal(data, ywd); err != nil {
		return nil, &WorkflowFileFormatError{Cause: err}
	}

	wd, err := yamlWorkflowDataToWorkflowData(ywd)
	if err != nil {
		return nil, err
	}

	return wd, nil
}

type documentStreamingDecoder struct {
	reader  io.ReadCloser
	decoder Decoder
	buf     *bytes.Buffer
}

func (d *documentStreamingDecoder) DecodeStream(ctx context.Context) (*WorkflowData, error) {
	var eof error

	_, err := d.buf.ReadFrom(d.reader)
	if err != nil {
		if err == io.EOF {
			eof = err
		} else {
			return nil, err
		}
	}

	wd, err := d.decoder.Decode(ctx, d.buf.Bytes())
	if err != nil {
		return nil, err
	}

	return wd, eof
}

func (d *documentStreamingDecoder) Close() error {
	return d.reader.Close()
}

// NewDocumentStreamingDecoder takes an io.ReadCloser and a Decoder and returns
// a new StreamingDecoder.
func NewDocumentStreamingDecoder(r io.ReadCloser, decoder Decoder) StreamingDecoder {
	return &documentStreamingDecoder{
		reader:  r,
		decoder: decoder,
		buf:     &bytes.Buffer{},
	}
}

func yamlWorkflowDataToWorkflowData(ywd *YAMLWorkflowData) (*WorkflowData, error) {
	env := &WorkflowData{
		APIVersion:  ywd.APIVersion,
		Description: ywd.Description,
		Name:        ywd.Name,
		Parameters:  ywd.Parameters,
	}

	for _, step := range ywd.Steps {
		stepType, err := makeStepType(step)
		if err != nil {
			return nil, err
		}
		switch stepType {
		case WorkflowStepTypeApproval:
			approval := map[string]interface{}{
				"$fn.equals": []interface{}{
					map[string]interface{}{"$type": "Answer", "askRef": step.Name, "name": string(WorkflowStepTypeApproval)},
					string(WorkflowStepApprovalApproved),
				},
			}

			when := make([]interface{}, 0)
			when = append(when, approval)

			// pre-existing conditions should always be supported...
			if step.When.Tree != nil {
				existing, ok := step.When.Tree.([]interface{})
				if ok {
					for _, condition := range existing {
						when = append(when, condition)
					}
				} else {
					when = append(when, step.When.Tree)
				}
			}

			env.Steps = append(env.Steps, &WorkflowStep{
				Name:      step.Name,
				DependsOn: step.DependsOn,
				When:      serialize.JSONTree{Tree: when},
				Variant:   &ApprovalWorkflowStep{},
			})
		default:
			env.Steps = append(env.Steps, &WorkflowStep{
				Name:      step.Name,
				DependsOn: step.DependsOn,
				When:      serialize.JSONTree(step.When),
				Variant: &ContainerWorkflowStep{
					ContainerMixin: ContainerMixin{
						Image:     step.Image,
						Spec:      makeJSONTreeMap(step.Spec),
						Env:       makeJSONTreeMap(step.Env),
						InputFile: step.InputFile,
						Input:     step.Input,
						Command:   step.Command,
						Args:      step.Args,
					},
				},
			})
		}
	}

	for _, trigger := range ywd.Triggers {
		et := &WorkflowDataTrigger{
			Name: trigger.Name,
			Binding: &WorkflowDataTriggerBinding{
				Key:        serialize.JSONTree(trigger.Binding.Key),
				Parameters: makeJSONTreeMap(trigger.Binding.Parameters),
			},
			When: serialize.JSONTree(trigger.When),
		}

		switch trigger.Source.Type {
		case WorkflowTriggerSourceTypePush.String():
			et.Source = &WorkflowDataTriggerSource{
				Type: WorkflowTriggerSourceTypePush.String(),
				Variant: &PushWorkflowTriggerSource{
					Schema: trigger.Source.Schema,
				},
			}
		case WorkflowTriggerSourceTypeSchedule.String():
			et.Source = &WorkflowDataTriggerSource{
				Type: WorkflowTriggerSourceTypeSchedule.String(),
				Variant: &ScheduleWorkflowTriggerSource{
					Schedule: trigger.Source.Schedule,
				},
			}
		case WorkflowTriggerSourceTypeWebhook.String():
			et.Source = &WorkflowDataTriggerSource{
				Type: WorkflowTriggerSourceTypeWebhook.String(),
				Variant: &WebhookWorkflowTriggerSource{
					ContainerMixin: ContainerMixin{
						Image:     trigger.Source.Image,
						Spec:      makeJSONTreeMap(trigger.Source.Spec),
						Env:       makeJSONTreeMap(trigger.Source.Env),
						InputFile: trigger.Source.InputFile,
						Input:     trigger.Source.Input,
						Command:   trigger.Source.Command,
						Args:      trigger.Source.Args,
					},
				},
			}
		}

		env.Triggers = append(env.Triggers, et)
	}

	return env, nil
}

func makeStepType(step YAMLWorkflowStep) (WorkflowStepType, error) {
	switch step.Type {
	case "", "container":
		return WorkflowStepTypeContainer, nil
	case "approval":
		return WorkflowStepTypeApproval, nil
	default:
		return WorkflowStepTypeUnknown, &WorkflowStepInvalidError{Name: step.Name, Type: step.Type}
	}
}

func makeJSONTreeMap(ym map[string]serialize.YAMLTree) map[string]serialize.JSONTree {
	if ym == nil {
		return nil
	}

	jm := make(map[string]serialize.JSONTree, len(ym))
	for k, v := range ym {
		jm[k] = serialize.JSONTree(v)
	}

	return jm
}
