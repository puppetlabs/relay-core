package gcs

import (
	"context"
	"fmt"
	"net/url"
	"path"

	gcstorage "cloud.google.com/go/storage"
	"github.com/puppetlabs/horsehead/storage"
	"google.golang.org/api/googleapi"
)

type GCS struct {
	client     *gcstorage.Client
	bucketName string
	namePrefix string
}

func init() {
	storage.RegisterFactory("gs", New)
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
	key = path.Join(s.namePrefix, key)
	w := s.client.Bucket(s.bucketName).Object(key).NewWriter(ctx)
	defer func() {
		cerr := w.Close()
		if nil != cerr && nil == err {
			err = translateError(cerr, "PUT gc://%s/%s", s.bucketName, key)
		}
	}()
	w.ObjectAttrs.ContentType = opts.ContentType
	err = translateError(sink(w), "PUT gc://%s/%s", s.bucketName, key)
	return
}

func (s *GCS) Get(ctx context.Context, key string, src storage.Source, opts storage.GetOptions) (err error) {
	key = path.Join(s.namePrefix, key)
	r, rerr := s.client.Bucket(s.bucketName).Object(key).NewReader(ctx)
	if nil != rerr {
		return translateError(rerr, "GET gc://%s/%s", s.bucketName, key)
	}
	defer func() {
		rerr := r.Close()
		if nil != rerr && nil == err {
			err = translateError(rerr, "GET gc://%s/%s", s.bucketName, key)
		}
	}()
	meta := &storage.Meta{
		ContentType: r.ContentType(),
	}
	err = translateError(src(meta, r), "GET gc://%s/%s", s.bucketName, key)
	return
}

func (s *GCS) Delete(ctx context.Context, key string, opts storage.DeleteOptions) error {
	key = path.Join(s.namePrefix, key)
	return translateError(s.client.Bucket(s.bucketName).Object(key).Delete(ctx),
		"DELETE gc://%s/%s", s.bucketName, key)
}

func stripSlash(path string) string {
	if len(path) > 0 && path[0] == '/' {
		return path[1:]
	}
	return path
}

func newGCS(u url.URL, client *gcstorage.Client) (storage.BlobStore, error) {
	return &GCS{
		client:     client,
		bucketName: u.Hostname(),
		namePrefix: stripSlash(path.Clean(u.Path)),
	}, nil
}

func New(u url.URL) (storage.BlobStore, error) {
	client, err := gcstorage.NewClient(context.Background())
	if nil != err {
		return nil, err
	}
	return newGCS(u, client)
}
