package gcs

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/puppetlabs/nebula-tasks/pkg/storage"
	"github.com/stretchr/testify/assert"
)

func TestGCS(t *testing.T) {
	bucketName := os.Getenv("GCS_BUCKET")
	if 0 == len(bucketName) || 0 == len(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")) {
		t.Skip("Define the GCS_BUCKET and GOOGLE_APPLICATION_CREDENTIALS environment variables to enable GCS tests")
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 60 * time.Second)
	defer cancel()

	key, err := uuid.NewRandom()
	assert.Nil(t, err)

	content := []byte("TEST CONTENT")

	// Ensure that gcs.New conforms to the BlobStorage interface.
	var gcs storage.BlobStorage

	gcs, err = New(Options {
		BucketName: bucketName,
	})
	assert.Nil(t, err)
	serr := gcs.Put(storage.PutOptions {
		Context: ctx,
		Key:     key.String(),
		Sink:    func(w io.Writer) error {
			_, err := w.Write(content)
			return err
		},
		ContentType: "application/testing",
	})
	assert.Nil(t, serr)
	var buf bytes.Buffer
	serr = gcs.Get(storage.GetOptions {
		Context: ctx,
		Key:     key.String(),
		Src:     func(r io.Reader) error {
			_, err := io.Copy(&buf, r)
			return err
		},
	})
	assert.Nil(t, serr)
	assert.Equal(t, content, buf.Bytes())
	gcs.Delete(storage.DeleteOptions {
		Context: ctx,
		Key:     key.String(),
	})
}
