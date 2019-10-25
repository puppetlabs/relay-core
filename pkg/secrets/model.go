package secrets

// Secret is a model that represents the key/value pair for a single secret.
type Secret struct {
	Key   string
	Value string
}

// AccessGrant is a model that contains the metadata for accessing
// Scoped secrets.
type AccessGrant struct {
	BackendAddr string
	ScopedPath  string
}
