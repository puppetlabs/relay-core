package vault

import (
	"context"
	"path"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/leg/encoding/transfer"
	"github.com/puppetlabs/relay-core/pkg/model"
)

// KVV2Client accesses metadata and data from a KV V2 engine mount uniformly.
type KVV2Client struct {
	client     *vaultapi.Client
	enginePath string
	path       string
}

func (c *KVV2Client) In(sub ...string) *KVV2Client {
	return &KVV2Client{
		client:     c.client,
		enginePath: c.enginePath,
		path:       path.Join(c.path, path.Join(sub...)),
	}
}

func (c *KVV2Client) Read(ctx context.Context) (interface{}, error) {
	sec, err := c.client.Logical().Read(c.dataPath())
	if err != nil {
		return nil, err
	} else if sec == nil {
		return nil, model.ErrNotFound
	}

	data, ok := sec.Data["data"].(map[string]interface{})
	if !ok {
		return nil, model.ErrNotFound
	}

	value, found := data["value"]
	if !found {
		return nil, model.ErrNotFound
	}

	return value, nil
}

func (c *KVV2Client) ReadString(ctx context.Context) (string, error) {
	raw, err := c.Read(ctx)
	if err != nil {
		return "", err
	}

	encoded, ok := raw.(string)
	if !ok {
		// TODO: Should this be a different error?
		return "", model.ErrNotFound
	}

	b, err := transfer.DecodeFromTransfer(encoded)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (c *KVV2Client) List(ctx context.Context) ([]string, error) {
	ls, err := c.client.Logical().List(c.metadataPath())
	if err != nil {
		return nil, err
	} else if ls == nil {
		return nil, model.ErrNotFound
	}

	ki, ok := ls.Data["keys"].([]interface{})
	if !ok {
		return nil, model.ErrNotFound
	}

	keys := make([]string, len(ki))
	for i, k := range ki {
		keys[i], ok = k.(string)
		if !ok {
			// TODO: Should this be a different error?
			return nil, model.ErrNotFound
		}
	}

	return keys, nil
}

func (c *KVV2Client) dataPath() string {
	return path.Join(c.enginePath, "data", c.path)
}

func (c *KVV2Client) metadataPath() string {
	return path.Join(c.enginePath, "metadata", c.path)
}

func NewKVV2Client(delegate *vaultapi.Client, enginePath string) *KVV2Client {
	return &KVV2Client{
		client:     delegate,
		enginePath: enginePath,
	}
}
