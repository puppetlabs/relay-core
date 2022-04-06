package file

import (
	"io"
	"io/fs"
	"os"
)

func Copy(source, destination string, permissions fs.FileMode) error {
	s, err := os.Open(source)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE, permissions)
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err = io.Copy(d, s); err != nil {
		return err
	}

	return nil
}
