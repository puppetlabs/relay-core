package v1beta1

type DecoratorType string

const (
	// DecoratorTypeLink is a reference to a URI. This informs a UI to display
	// the decorator as a link.
	DecoratorTypeLink DecoratorType = "relay.sh/decorator-link"
)

// Decorator describes a result for a concluded step. These can be added to
// steps to represent hints to UI's (web, cli, etc.) about data generated as a
// result of a step's run.
type Decorator struct {
	// Name is a way to identify the decorator
	Name string `json:"name"`

	// Type determines how the decorator's data field is parsed. This might
	// mean validating the data's structure against a schema.
	Type DecoratorType `json:"type,omitempty"`

	// Link is a link-type decorator.
	// +optional
	Link *DecoratorLink `json:"link,omitempty"`
}

// DecoratorLink holds the value for a link-type result.
type DecoratorLink struct {
	// Description describes what the URI points to.
	Description string `json:"description"`

	// URI must follow the syntax outlined in
	// https://datatracker.ietf.org/doc/html/rfc3986 and must be parsable by
	// Go's url.Parse function.
	URI string `json:"uri"`
}
