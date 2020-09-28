package spec

import "encoding/json"

// StepMetadata is a subset of fields for relaysh step-metadata.json. It is
// used for decoding that payload into schema objects for step spec validation.
type StepMetadata struct {
	// Publish is the step image publishing metadata
	Publish struct {
		// Repository is the image repo for the step
		Repository string `json:"repository"`
	} `json:"publish"`
	// Schemas is a map of schemas available for the given step repository
	Schemas map[string]json.RawMessage `json:"schemas"`
}
