package taskutil

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
)

func WriteToFile(file, content string) error {
	f, err := createFilePath(file)
	if err != nil {
		return err
	}

	defer f.Close()

	if err := write(f, content); err != nil {
		return err
	}

	f.Sync()

	return nil
}

func WriteDataToFile(file string, data []byte) error {
	f, err := createFilePath(file)
	if err != nil {
		return err
	}

	defer f.Close()

	if err := writeData(f, data); err != nil {
		return err
	}

	f.Sync()

	return nil
}

func write(w io.Writer, content string) error {
	b, err := base64.StdEncoding.DecodeString(content)
	if err == nil {
		return writeData(w, b)
	}

	return writeData(w, []byte(content))
}

func writeData(w io.Writer, data []byte) error {
	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}

func createFilePath(file string) (*os.File, error) {
	dir := filepath.Dir(file)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	f, err := os.Create(file)
	if err != nil {
		return nil, err
	}

	return f, nil
}
