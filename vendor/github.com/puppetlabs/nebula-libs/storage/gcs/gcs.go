package gcs

import (
	"context"
	"fmt"
	"net/url"

	gcstorage "cloud.google.com/go/storage"
	"github.com/puppetlabs/horsehead/storage"
	"google.golang.org/api/googleapi"
)

type GCS struct {
	client     *gcstorage.Client
	bucketName string
}

func init() {
	storage.RegisterFactory("gcs", New)
}

// Translate a gcstorage error into a storage error.
func translateError(err error, format string, a ...interface{}) error {
	if nil == err {
		return nil
	}
	msg := fmt.Sprintf(format, a...)
	if context.Canceled == err || context.DeadlineExceeded == err {
		return storage.Errorf(
			err,
			storage.TimeoutError,
			"%s: %s", msg, err.Error())
	}
	if gcstorage.ErrObjectNotExist == err || gcstorage.ErrBucketNotExist == err {
		return storage.Errorf(
			err,
			storage.NotFoundError,
			"%s: %s", msg, err.Error())
	}
	if e, ok := err.(*googleapi.Error); ok {
		switch e.Code {
		case 404:
			return storage.Errorf(
				err,
				storage.NotFoundError,
				"%s: %s", msg, err.Error())
		case 401, 403:
			return storage.Errorf(
				err,
				storage.AuthError,
				"%s: %s", msg, err.Error())
		}
	}
	return storage.Errorf(
		err,
		storage.UnknownError,
		"%s: %s", msg, err.Error())
}

func (s *GCS) Put(ctx context.Context, key string, sink storage.Sink, opts storage.PutOptions) (err error) {
	w := s.client.Bucket(s.bucketName).Object(key).NewWriter(ctx)
	defer func() {
		cerr := w.Close()
		if nil != cerr && nil == err {
			err = translateError(cerr, "PUT gcs:///%s/%s", s.bucketName, key)
		}
	}()
	w.ObjectAttrs.ContentType = opts.ContentType
	err = translateError(sink(w), "PUT gcs:///%s/%s", s.bucketName, key)
	return
}

func (s *GCS) Get(ctx context.Context, key string, src storage.Source, opts storage.GetOptions) (err error) {
	r, rerr := s.client.Bucket(s.bucketName).Object(key).NewReader(ctx)
	if nil != rerr {
		return translateError(rerr, "GET gcs:///%s/%s", s.bucketName, key)
	}
	defer func() {
		rerr := r.Close()
		if nil != rerr && nil == err {
			err = translateError(rerr, "GET gcs:///%s/%s", s.bucketName, key)
		}
	}()
	meta := &storage.Meta {
		ContentType: r.ContentType(),
	}
	err = translateError(src(meta, r), "GET gcs:///%s/%s", s.bucketName, key)
	return
}

func (s *GCS) Delete(ctx context.Context, key string, opts storage.DeleteOptions) error {
	return translateError(s.client.Bucket(s.bucketName).Object(key).Delete(ctx),
		"DELETE gcs:///%s/%s", s.bucketName, key)
}

func newGCS(u url.URL, client *gcstorage.Client) (storage.BlobStore, error) {
	bucketNames := u.Query()["bucket"]
	if 1 != len(bucketNames) {
		return nil, fmt.Errorf("Invalid URL, must contain exactly one 'bucket=...' in the query string")
	}
	return &GCS{
		client:     client,
		bucketName: bucketNames[0],
	}, nil
}

func New(u url.URL) (storage.BlobStore, error) {
	client, err := gcstorage.NewClient(context.Background())
	if nil != err {
		return nil, err
	}
	return newGCS(u, client)
}
