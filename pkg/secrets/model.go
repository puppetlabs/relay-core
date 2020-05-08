package secrets

import "encoding/json"

// Secret is a model that represents the key/value pair for a single secret.
type Secret struct {
	Key   string
	Value string
}

// AccessGrant is a model that contains the metadata for accessing
// Scoped secrets.
type AccessGrant struct {
	BackendAddr string `json:"backend_addr"`
	MountPath   string `json:"mount_path"`
	ScopedPath  string `json:"scoped_path"`
}

func UnmarshalGrants(b []byte) (map[string]*AccessGrant, error) {
	var t map[string]*AccessGrant

	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}

	return t, nil
}

func MarshalGrants(grants map[string]*AccessGrant) ([]byte, error) {
	return json.Marshal(grants)
}
