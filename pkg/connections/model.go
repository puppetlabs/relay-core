package connections

// Connection is a model for relay connections. It has a simple spec map that
// contains key/value pairs that get merged into the parse tree when workflow
// step specs are evaluated.
type Connection struct {
	Name string
	Type string
	Spec map[string]string
}
