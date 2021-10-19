package v1beta1

// Decorator describes a result for a concluded step. These can be added to
// steps to represent hints to UI's (web, cli, etc.) about data generated as a
// result of a step's run.
type Decorator struct {
	// Name is a way to identify the decorator
	Name string `json:"name"`

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
