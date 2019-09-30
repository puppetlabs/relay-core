package secrets

// Secret is a model that represents the key/value pair for a single secret.
type Secret struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// AccessGrant is a model that contains the metadata for accessing
// Scoped secrets.
type AccessGrant struct {
	BackendAddr string
	ScopedPath  string
}
