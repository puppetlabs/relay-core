package workdir

import (
	"errors"
	"os"
	"path/filepath"
)

const defaultMode = 0755

type CleanupFunc func() error

// WorkDir is a response type that contains the Path to a directory created by this package.
type WorkDir struct {
	// Path is the absolute path to the directory requested.
	Path string
	// Cleanup is a function that will cleanup any directory and files under
	// Path.
	Cleanup CleanupFunc
}

type dirType int

const (
	// DirTypeConfig is a directory used to store configuration. This is commonly
	// used to store application configs like yaml or json files used during
	// the bootstrapping phase of an application startup.
	DirTypeConfig dirType = iota
	// DirTypeCache is a directory used to store any temporary cache that is generated
	// by the application. This directory type can be used to store tokens when logging in,
	// or serialized cache such as large responses that you don't want to have to request again
	// for some amount of time. Anything in here should be considered temporary and can be removed
	// at any time.
	DirTypeCache
	// DirTypeData is a directory to store long term data. This data can be database files or assets
	// that need to later be extracted out into another location. Things in this directory should be
	// considered important and backed up.
	DirTypeData
)

// dirTypeEnvDefault is a type that represents the environment variable name
// and its default if it's not set.
type dirTypeEnvDefault struct {
	// envName is the name of the environment variable we should check for first.
	envName string
	// defaultLoc is the default location for the directory type. `default` is a
	// keyword in the Go language, so we use defaultLoc here to prevent syntax
	// collisions.
	defaultLoc string
}

var dirTypeEnv = map[dirType]dirTypeEnvDefault{
	DirTypeConfig: dirTypeEnvDefault{
		envName:    "XDG_CONFIG_HOME",
		defaultLoc: filepath.Join(os.Getenv("HOME"), ".config"),
	},
	DirTypeCache: dirTypeEnvDefault{
		envName:    "XDG_CACHE_HOME",
		defaultLoc: filepath.Join(os.Getenv("HOME"), ".cache"),
	},
	DirTypeData: dirTypeEnvDefault{
		envName:    "XDG_DATA_HOME",
		defaultLoc: filepath.Join(os.Getenv("HOME"), ".local", "share"),
	},
}

// New returns a new WorkDir or an error. An error is returned if p is empty.
// A standard cleanup function is made available so the caller can decide if they want to
// remove the directory created after they are done. Options allow additional control over
// the directory attributes.
func New(p string, opts Options) (*WorkDir, error) {
	if p == "" {
		return nil, errors.New("path cannot be empty")
	}

	mode := os.FileMode(defaultMode)
	if opts.Mode != 0 {
		mode = opts.Mode
	}

	if err := os.MkdirAll(p, mode); err != nil {
		return nil, err
	}

	wd := &WorkDir{
		Path: p,
		Cleanup: func() error {
			return os.RemoveAll(p)
		},
	}

	return wd, nil
}

// Namespace holds the directory parts that will be joined together to form
// a namespaced path segment in the final workdir.
type Namespace struct {
	parts []string
}

// New returns a new WorkDir under the context of dt (directory type) and allows for setting
// a namespace. Below is an example of its use:
// wd, _ := NewNamespace([]string{"foo", "bar"}).New(DirTypeConfig, Options{})
// fmt.Println(wd.Path)
//
// Out: /home/kyle/.config/foo/bar
func (n *Namespace) New(dt dirType, opts Options) (*WorkDir, error) {
	def := dirTypeEnv[dt]

	p := filepath.Join(def.defaultLoc, filepath.Join(n.parts...))

	if os.Getenv(def.envName) != "" {
		p = filepath.Join(os.Getenv(def.envName), filepath.Join(n.parts...))
	}

	return New(p, opts)
}

// NewNamespace returns a new Namespace with the provided parts slice set
func NewNamespace(parts []string) *Namespace {
	return &Namespace{parts: parts}
}
