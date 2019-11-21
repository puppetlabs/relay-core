package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/puppetlabs/horsehead/v2/storage"
)

const (
	// DefaultFilePermissions is the default permission octal for the file that
	// holds the blob content.
	DefaultFilePermissions = os.FileMode(0640)

	// DefaultDirPermissions is the default permission octal for all directories created
	// as part of a key's path.
	DefaultDirPermissions = os.FileMode(0740)
)

func init() {
	storage.RegisterFactory("file", New)
}

func New(u url.URL) (storage.BlobStore, error) {
	stat, err := os.Stat(u.Path)
	if err != nil {
		return nil, fmt.Errorf("stat(%s) failed: %s", u.Path, err.Error())
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("%s must be a directory", u.Path)
	}
	fs := &Filesystem{
		blobPath:        filepath.Join(u.Path, "blob"),
		metaPath:        filepath.Join(u.Path, "meta"),
		filePermissions: DefaultFilePermissions,
		dirPermissions:  DefaultDirPermissions,
	}
	if arr := u.Query()["filePermissions"]; len(arr) > 0 {
		perm, err := strconv.ParseUint(arr[0], 8, 32)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse filePermissions=%s (%s)", arr[0], err.Error())
		}
		fs.filePermissions = os.FileMode(perm)
	}
	if arr := u.Query()["dirPermissions"]; len(arr) > 0 {
		perm, err := strconv.ParseUint(arr[0], 8, 32)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse dirPermissions=%s (%s)", arr[0], err.Error())
		}
		fs.dirPermissions = os.FileMode(perm)
	}
	// Ensure the blob/ and meta/ dirs exist:
	if err := os.MkdirAll(fs.blobPath, fs.dirPermissions); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(fs.metaPath, fs.dirPermissions); err != nil {
		return nil, err
	}
	return fs, nil
}

type Filesystem struct {
	blobPath        string
	metaPath        string
	filePermissions os.FileMode
	dirPermissions  os.FileMode
}

func translateError(err error, format string, a ...interface{}) error {
	if nil == err || io.EOF == err {
		return nil
	}
	msg := fmt.Sprintf(format, a...)
	if context.Canceled == err || context.DeadlineExceeded == err {
		return storage.Errorf(
			err,
			storage.TimeoutError,
			"%s: %s", msg, err.Error())
	}
	if os.IsPermission(err) {
		return storage.Errorf(
			err,
			storage.AuthError,
			"%s: %s", msg, err.Error())
	}
	if os.IsNotExist(err) {
		return storage.Errorf(
			err,
			storage.NotFoundError,
			"%s: %s", msg, err.Error())
	}
	return storage.Errorf(
		err,
		storage.UnknownError,
		"%s: %s", msg, err.Error())
}

func isErrInvalid(err error) bool {
	if err == os.ErrInvalid {
		return true
	}
	if syserr, ok := err.(*os.SyscallError); ok && syserr.Err == syscall.EINVAL {
		return true
	}
	return false
}

type Meta struct {
	ContentType string `json:"Content-Type"`
}

func (fs *Filesystem) Put(ctx context.Context, key string, sink storage.Sink, opts storage.PutOptions) (err error) {
	if key == "" {
		return storage.Errorf(nil, storage.UnknownError, "key must be non-empty")
	}

	dir, _ := filepath.Split(key)

	if dir != "" {
		fdir := filepath.Join(fs.blobPath, dir)
		if rerr := os.MkdirAll(fdir, fs.dirPermissions); rerr != nil {
			return translateError(rerr, "mkdir -p %s", fdir)
		}
		fdir = filepath.Join(fs.metaPath, dir)
		if rerr := os.MkdirAll(fdir, fs.dirPermissions); rerr != nil {
			return translateError(rerr, "mkdir -p %s", fdir)
		}
	}

	if true {
		path := filepath.Join(fs.metaPath, key)

		f, rerr := os.OpenFile(path, os.O_RDWR|os.O_CREATE, fs.filePermissions)
		if rerr != nil {
			return translateError(rerr, "open(%s)", path)
		}
		defer func() {
			rerr := f.Close()
			if nil == err {
				err = translateError(rerr, "close(%s)", path)
			}
		}()
		rerr = json.NewEncoder(f).Encode(&Meta{
			ContentType: opts.ContentType,
		})
		if rerr != nil {
			return translateError(rerr, "write(%s)", path)
		}
	}

	path := filepath.Join(fs.blobPath, key)

	f, rerr := os.OpenFile(path, os.O_RDWR|os.O_CREATE, fs.filePermissions)
	if rerr != nil {
		return translateError(rerr, "open(%s)", path)
	}
	defer func() {
		rerr := f.Close()
		if nil == err {
			err = translateError(rerr, "close(%s)", path)
		}
	}()

	if rerr := sink(f); rerr != nil {
		return translateError(rerr, "write(%s)", path)
	}

	return nil
}

type truncatedReader struct {
	wrap   io.Reader
	length int64
}

func (r *truncatedReader) Read(p []byte) (int, error) {
	if r.length <= 0 {
		return 0, io.EOF
	}
	if r.length < int64(len(p)) {
		p = p[0:r.length]
	}
	n, err := r.wrap.Read(p)
	r.length -= int64(n)
	return n, err
}

func (fs *Filesystem) Get(ctx context.Context, key string, src storage.Source, opts storage.GetOptions) (err error) {
	if key == "" {
		return storage.Errorf(nil, storage.UnknownError, "key must be non-empty")
	}

	var meta storage.Meta
	if true {
		path := filepath.Join(fs.metaPath, key)

		f, rerr := os.Open(path)
		if rerr != nil {
			return translateError(rerr, "open(%s)", path)
		}
		defer func() {
			rerr := f.Close()
			if nil == err {
				err = translateError(rerr, "close(%s)", path)
			}
		}()
		var m Meta
		rerr = json.NewDecoder(f).Decode(&m)
		if rerr != nil {
			return translateError(rerr, "read(%s)", path)
		}
		meta.ContentType = m.ContentType
	}

	path := filepath.Join(fs.blobPath, key)

	f, rerr := os.Open(path)
	if rerr != nil {
		return translateError(rerr, "open(%s)", path)
	}
	defer func() {
		rerr := f.Close()
		if nil == err {
			err = translateError(rerr, "close(%s)", path)
		}
	}()

	fi, rerr := f.Stat()
	if nil != rerr {
		return translateError(rerr, "stat(%s)", path)
	}
	meta.Size = fi.Size()

	var reader io.Reader
	if opts.Offset < 0 {
		if opts.Length > 0 {
			return storage.Errorf(
				nil,
				storage.UnknownError,
				"Length must be -1 if Offset is negative in storage.GetOptions")
		}
		_, rerr := f.Seek(opts.Offset, 2)
		// We must ignore invalid argument error, since -100 on a 10 byte file
		// should simply seek to the beginning of the file.
		if nil != rerr && isErrInvalid(rerr) {
			return translateError(rerr, "seek(%d, 2)", opts.Offset)
		}
		meta.Offset = opts.Offset + meta.Size
		if meta.Offset < 0 {
			meta.Offset = 0
		}
	} else if opts.Offset > 0 {
		_, rerr := f.Seek(opts.Offset, 0)
		if nil != rerr {
			return translateError(rerr, "seek(%d, 0)", opts.Offset)
		}
		meta.Offset = opts.Offset
	}
	if opts.Length > 0 {
		reader = &truncatedReader{
			wrap:   f,
			length: opts.Length,
		}
	} else {
		reader = f
	}
	if rerr := src(&meta, reader); rerr != nil {
		return translateError(rerr, "read(%s)", path)
	}
	return nil
}

func (fs *Filesystem) Delete(ctx context.Context, key string, opts storage.DeleteOptions) error {
	if key == "" {
		return storage.Errorf(nil, storage.UnknownError, "key must be non-empty")
	}

	if true {
		path := filepath.Join(fs.metaPath, key)

		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return translateError(err, "remove(%s)", path)
		}
	}

	path := filepath.Join(fs.blobPath, key)

	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return translateError(err, "remove(%s)", path)
	}

	return nil
}
