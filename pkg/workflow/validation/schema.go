package validation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/puppetlabs/relay-core/pkg/util/typeutil"
	"github.com/xeipuuv/gojsonschema"
)

// Schema is a step spec schema. It provides a Validate method for ensuring at
// runtime, specs for steps meet the schema's requirements.
type Schema interface {
	// Validate takes the data to be validated as a byte array.
	Validate(data []byte) error
	// ValidateGo takes the data to be validated as an arbitrary go data structure.
	ValidateGo(data interface{}) error
}

// SchemaRegistry is a registry of spec schemas for steps.
type SchemaRegistry interface {
	// GetByStepRepository returns the spec Schema for repo. Implementations of
	// this interface might use the image repo as the repo string to look up
	// schemas.
	GetByImage(ref name.Reference) (Schema, error)
}

// JSONSchema implements Schema and uses json-schema to validate spec data.
type JSONSchema struct {
	schema *gojsonschema.Schema
}

func (j *JSONSchema) Validate(data []byte) error {
	return j.validate(gojsonschema.NewBytesLoader(data))
}

func (j *JSONSchema) ValidateGo(data interface{}) error {
	return j.validate(gojsonschema.NewGoLoader(data))
}

func (j *JSONSchema) validate(loader gojsonschema.JSONLoader) error {
	//	result, err := j.schema.Validate()
	result, err := j.schema.Validate(loader)
	if err != nil {
		return &SchemaValidationError{Cause: err}
	}

	if err := typeutil.ValidationErrorFromResult(result); err != nil {
		return &SchemaValidationError{Cause: err}
	}

	return nil
}

// StepMetadataSchemaRegistry is a registry that loads spec schemas for steps
// from a single file at a URL. An example of this file can be found in
// `testdata/step-metadata.json`.
type StepMetadataSchemaRegistry struct {
	LastResponse *http.Response

	registry     map[string]*gojsonschema.Schema
	metadataURL  *url.URL
	lastModified string
	client       *http.Client

	sync.RWMutex
}

// GetByImage takes an image repo ref and looks up the schema for its spec
// and returns a Schema for it. If the repo cannot be found, it returns
// SchemaDoesNotExistError.
func (s *StepMetadataSchemaRegistry) GetByImage(ref name.Reference) (Schema, error) {
	if err := s.fetchStepMetadata(); err != nil {
		return nil, err
	}

	s.RLock()
	defer s.RUnlock()

	repository := ref.Context()
	repo := repository.RepositoryStr()

	schema, ok := s.registry[repo]
	if !ok {
		return nil, &SchemaDoesNotExistError{Name: repo}
	}

	return &JSONSchema{schema: schema}, nil
}

func (s *StepMetadataSchemaRegistry) fetchStepMetadata() error {
	if s.LastResponse != nil && s.LastResponse.Header.Get("last-modified") != "" {
		req, err := http.NewRequest("HEAD", s.metadataURL.String(), nil)
		if err != nil {
			return err
		}

		req.Header.Set("if-modified-since", s.LastResponse.Header.Get("last-modified"))

		resp, err := s.client.Do(req)
		if err != nil {
			return err
		}

		switch resp.StatusCode {
		case http.StatusNotModified:
			s.LastResponse = resp

			return nil
		case http.StatusOK:
		default:
			return &StepMetadataFetchError{StatusCode: resp.StatusCode}
		}
	}

	resp, err := s.client.Get(s.metadataURL.String())
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &StepMetadataFetchError{StatusCode: resp.StatusCode}
	}

	if err := s.decodeStepMetadataIntoRegistry(resp.Body); err != nil {
		return err
	}

	s.LastResponse = resp

	return nil
}

func (s *StepMetadataSchemaRegistry) decodeStepMetadataIntoRegistry(r io.Reader) error {
	s.Lock()
	defer s.Unlock()

	sm := []*StepMetadata{}

	if err := json.NewDecoder(r).Decode(&sm); err != nil {
		return err
	}

	for _, step := range sm {
		if v, ok := step.Schemas["spec"]; ok {
			// test to see if we have an empty schema object ({})
			var j map[string]json.RawMessage

			if err := json.Unmarshal(v, &j); err != nil {
				return err
			}

			if len(j) > 0 {
				loader := gojsonschema.NewBytesLoader(v)

				schema, err := gojsonschema.NewSchema(loader)
				if err != nil {
					return err
				}

				s.registry[step.Publish.Repository] = schema
			}
		}
	}

	return nil
}

// NewStepMetadataSchemaRegistry returns a new StepMetadataSchemaRegistry for a
// given URL. An initial request for the repo file is made and could return an
// error if the file does not exist, or is otherwise broken.
func NewStepMetadataSchemaRegistry(u *url.URL, opts ...StepMetadataSchemaRegistryOption) (*StepMetadataSchemaRegistry, error) {
	reg := &StepMetadataSchemaRegistry{
		client:      http.DefaultClient,
		metadataURL: u,
		registry:    make(map[string]*gojsonschema.Schema),
	}

	for _, opt := range opts {
		opt(reg)
	}

	if err := reg.fetchStepMetadata(); err != nil {
		return nil, err
	}

	return reg, nil
}

// StepMetadataSchemaRegistryOption is for setting optional configuration on
// StepMetadataSchemaRegistry objects when using NewStepMetadataSchemaRegistry.
type StepMetadataSchemaRegistryOption func(*StepMetadataSchemaRegistry)

// WithStepMetadataSchemaRegistryClient sets the http.Client to use when
// StepMetadataSchemaRegistry is making requests to fetch step-metadata.json
// files.
func WithStepMetadataSchemaRegistryClient(client *http.Client) StepMetadataSchemaRegistryOption {
	return func(r *StepMetadataSchemaRegistry) {
		r.client = client
	}
}
