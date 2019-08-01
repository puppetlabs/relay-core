package storage

// Look here for various implementations:
// 	https://github.com/puppetlabs/nebula-libs/tree/master/storage"
// 

import (
	"context"
	"io"
)

type StorageErrorCode string
const (
	AuthError     StorageErrorCode = "AuthError"
	NotFoundError StorageErrorCode = "NotFoundError"
	TimeoutError  StorageErrorCode = "TimeoutError"
	UnknownError  StorageErrorCode = "UnknownError"
)

type StorageError struct {
	code StorageErrorCode
	msg string
	cause error
}

func (e *StorageError) Error() string {
	return e.msg
}

func (e *StorageError) Code() StorageErrorCode {
	return e.code
}

func (e *StorageError) Cause() error {
	return e.cause
}

func NewStorageError(code StorageErrorCode, msg string, cause error) *StorageError {
	return &StorageError {
		code:  code,
		msg:   msg,
		cause: cause,
	}
}

type StorageMeta struct {
	Key           string
	Metadata      map[string]string
	ContentLength int64
}

type StorageSink func(io.Writer) error
type StorageSrc func(io.Reader) error

type PutOptions struct {
	Context     context.Context
	Key         string
	Sink        StorageSink
	ContentType string
}

type GetOptions struct {
	Context context.Context
	Key     string
	Src     StorageSrc
}

type DeleteOptions struct {
	Context context.Context
	Key     string
}

type BlobStorage interface {
	Put(PutOptions) *StorageError
	Get(GetOptions) *StorageError
	Delete(DeleteOptions) *StorageError
}
