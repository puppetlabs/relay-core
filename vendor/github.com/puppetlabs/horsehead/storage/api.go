package storage

// Look here for various implementations:
// 	https://github.com/puppetlabs/nebula-libs/tree/master/storage"
//

import (
	"context"
	"fmt"
	"io"
)

type ErrorCode string

const (
	AuthError     ErrorCode = "AuthError"
	NotFoundError ErrorCode = "NotFoundError"
	TimeoutError  ErrorCode = "TimeoutError"
	UnknownError  ErrorCode = "UnknownError"
)

type errorImpl struct {
	message string
	code    ErrorCode
	cause   error
}

func (e *errorImpl) Error() string {
	return e.message
}

func (e *errorImpl) Unwrap() error {
	return e.cause
}

func Errorf(cause error, code ErrorCode, format string, a ...interface{}) error {
	return &errorImpl{
		code:    code,
		message: fmt.Sprintf(format, a...),
		cause:   cause,
	}
}

func IsAuthError(err error) bool {
	e, ok := err.(*errorImpl)
	return ok && e.code == AuthError
}

func IsNotFoundError(err error) bool {
	e, ok := err.(*errorImpl)
	return ok && e.code == NotFoundError
}

func IsTimeoutError(err error) bool {
	e, ok := err.(*errorImpl)
	return ok && e.code == TimeoutError
}

type Sink func(io.Writer) error
type Source func(*Meta, io.Reader) error

type Meta struct {
	// ContentType of the blob
	ContentType string
	// Offset within the blob that the read begins
	Offset int64
	// Size is the total size of the blob
	Size int64
}
type PutOptions struct {
	// ContentType of the blob
	ContentType string
}
type GetOptions struct {
	// Offset is the byte offset to begin at, it may be negative
	// to specify an offset from the end of the blob.
	Offset int64

	// Length is the maximum number of bytes to return, if <= 0
	// then all bytes after Offset are returned. The Length must
	// be <= 0 if Offset is negative due to the limitations of
	// HTTP range requests.
	Length int64
}
type DeleteOptions struct {
	// TODO: Support conditional deletes?
}

type BlobStore interface {
	Put(ctx context.Context, key string, sink Sink, opts PutOptions) error
	Get(ctx context.Context, key string, source Source, opts GetOptions) error
	Delete(ctx context.Context, key string, opts DeleteOptions) error
}
