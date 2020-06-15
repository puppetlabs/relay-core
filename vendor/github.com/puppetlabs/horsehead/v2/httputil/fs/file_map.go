package fs

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type fileMapFile struct {
	http.File

	fm fileMap
}

func (fmf fileMapFile) Readdir(count int) ([]os.FileInfo, error) {
	fis, err := fmf.File.Readdir(count)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(fis); {
		fi := fis[i]

		if _, found := fmf.fm[fi.Name()]; !found {
			copy(fis[i:], fis[i+1:])
			fis[len(fis)-1] = nil
			fis = fis[:len(fis)-1]
		} else {
			i++
		}
	}

	if count > 0 && len(fis) == 0 {
		return nil, io.EOF
	}

	return fis, nil
}

type fileMap map[string]fileMap

type fileMapFS struct {
	root string
	fm   fileMap
}

func (fmfs fileMapFS) Open(name string) (http.File, error) {
	name = filepath.FromSlash(path.Clean("/" + name))
	fl := strings.Split(name, string(filepath.Separator))

	iter := fmfs.fm
	for _, fc := range fl {
		if fc == "" {
			continue
		}

		var found bool
		iter, found = iter[fc]
		if !found {
			return nil, os.ErrNotExist
		}
	}

	fp, err := os.Open(filepath.Join(fmfs.root, name))
	if err != nil {
		return nil, err
	}

	return &fileMapFile{File: fp, fm: iter}, nil
}

func FileMap(root string, files []string) http.FileSystem {
	if root == "" {
		root = "."
	}

	fm := make(fileMap)

	for _, f := range files {
		fl := strings.Split(filepath.FromSlash(path.Clean(f)), string(filepath.Separator))

		for i, iter := 0, fm; i < len(fl); i++ {
			fc := fl[i]
			if fc == "" {
				continue
			}

			if _, found := iter[fc]; !found {
				iter[fc] = make(fileMap)
			}

			iter = iter[fc]
		}
	}

	return &fileMapFS{
		root: root,
		fm:   fm,
	}
}

func FileMapWithoutModTimes(root string, files []string) http.FileSystem {
	return FileSystemWithoutModTimes(FileMap(root, files))
}
