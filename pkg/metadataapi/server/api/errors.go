package api

import (
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/model"
)

func ModelReadError(err error) errors.Error {
	switch err {
	case model.ErrNotFound:
		return errors.NewModelNotFoundError()
	case model.ErrRejected:
		return errors.NewModelAuthorizationError()
	default:
		return errors.NewModelReadError().WithCause(err)
	}
}

func ModelWriteError(err error) errors.Error {
	switch err {
	case model.ErrNotFound:
		return errors.NewModelNotFoundError()
	case model.ErrRejected:
		return errors.NewModelAuthorizationError()
	case model.ErrConflict:
		return errors.NewModelConflictError()
	default:
		return errors.NewModelWriteError().WithCause(err)
	}
}
