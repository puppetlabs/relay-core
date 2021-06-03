package vault

import (
	"context"
	"strings"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type ConnectionManager struct {
	client *KVV2Client
}

var _ model.ConnectionManager = &ConnectionManager{}

func (m *ConnectionManager) List(ctx context.Context) ([]*model.Connection, error) {
	// The type-name mapping and connection information share a keyspace (?!) so
	// this operation will list both. We filter them by checking whether the
	// second-level entries are effectively pointers to other data or not.
	//
	// TODO: Move these into separate keyspaces.
	candidateTypes, err := m.client.List(ctx)
	if err == model.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var l []*model.Connection

	for _, candidateType := range candidateTypes {
		candidateNames, err := m.client.In(candidateType).List(ctx)
		if err == model.ErrNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

		for _, candidateName := range candidateNames {
			c, err := m.Get(ctx, candidateType, candidateName)
			if err == model.ErrNotFound {
				continue
			} else if err != nil {
				return nil, err
			}

			l = append(l, c)
		}
	}

	return l, nil
}

func (m *ConnectionManager) Get(ctx context.Context, typ, name string) (*model.Connection, error) {
	connectionID, err := m.client.In(typ, name).ReadString(ctx)
	if err != nil {
		return nil, err
	}

	keys, err := m.client.In(connectionID).List(ctx)
	if err != nil {
		return nil, err
	}

	attrs := make(map[string]interface{}, len(keys))
	for _, key := range keys {
		value, err := m.client.In(connectionID, key).ReadString(ctx)
		if err == model.ErrNotFound {
			// Deleted from under us?
			continue
		} else if err != nil {
			return nil, err
		}

		attrs[key] = value
	}

	return &model.Connection{
		Type:       strings.TrimSuffix(typ, "/"),
		Name:       name,
		Attributes: attrs,
	}, nil
}

func NewConnectionManager(client *KVV2Client) *ConnectionManager {
	return &ConnectionManager{
		client: client,
	}
}
