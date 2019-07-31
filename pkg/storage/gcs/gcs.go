package gcs

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/storage"
	gcstorage "cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
)

type Options struct {
	BucketName string
}

type GCS struct {
	client     *gcstorage.Client
	bucketName string
}

func translateError(err error) *storage.StorageError {
	if nil == err {
		return nil
	}
	if context.Canceled == err || context.DeadlineExceeded == err {
		return storage.NewStorageError(
			storage.TimeoutError,
			err.Error(),
			err)
	}
	if gcstorage.ErrObjectNotExist == err || gcstorage.ErrBucketNotExist == err {
		return storage.NewStorageError(
			storage.NotFoundError,
			err.Error(),
			err)
	}
	if e, ok := err.(*googleapi.Error); ok {
		switch e.Code {
		case 404:
			return storage.NewStorageError(
				storage.NotFoundError,
				err.Error(),
				err)
		case 401:
			return storage.NewStorageError(
				storage.AuthError,
				err.Error(),
				err)
		case 403:
			return storage.NewStorageError(
				storage.AuthError,
				err.Error(),
				err)
		}
	}
	return storage.NewStorageError(
		storage.UnknownError,
		err.Error(),
		err)
}

func (s *GCS) Put(opts storage.PutOptions) (err *storage.StorageError) {
	w := s.client.Bucket(s.bucketName).Object(opts.Key).NewWriter(opts.Context)
	defer func() {
		cerr := w.Close()
		if nil != cerr && nil == err {
			err = translateError(cerr)
		}
	}()
	w.ObjectAttrs.ContentType = opts.ContentType
	err = translateError(opts.Sink(w))
	return
}

func (s *GCS) Get(opts storage.GetOptions) (err *storage.StorageError) {
	r, rerr := s.client.Bucket(s.bucketName).Object(opts.Key).NewReader(opts.Context)
	if nil != rerr {
		return translateError(rerr)
	}
	defer func() {
		rerr := r.Close()
		if nil != rerr && nil == err {
			err = translateError(rerr)
		}
	}()
	err = translateError(opts.Src(r))
	return
}

func (s *GCS) Delete(opts storage.DeleteOptions) *storage.StorageError {
	return translateError(s.client.Bucket(s.bucketName).Object(opts.Key).Delete(opts.Context))
}


func New(opts Options) (*GCS, error) {
	client, err := gcstorage.NewClient(context.Background())
	if err != nil {
		return nil, err
	}
	return &GCS {
		client:     client,
		bucketName: opts.BucketName,
	}, nil
}
