package taskutil

import (
	"encoding/base64"
	"io"
	"os"
)

func WriteToFile(path, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	if err := WriteBase64String(f, content); err != nil {
		return err
	}

	f.Sync()

	return nil
}

func WriteBase64String(w io.Writer, content string) error {
	b, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return err
	}

	if _, err := w.Write(b); err != nil {
		return err
	}

	return nil
}
