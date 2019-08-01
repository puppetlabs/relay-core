package storage

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]BlobStorageFactory)
)

type BlobStorageFactory func(url.URL) (BlobStorage, error)

func NewBlobStorage(u url.URL) (BlobStorage, error) {
	scheme := strings.ToLower(u.Scheme)
	factoriesMu.RLock()
	factory, ok := factories[scheme]
	factoriesMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("stroage: unknown scheme %q (forgotten import?)", scheme)
	}
	return factory(u)
}

func RegisterFactory(scheme string, factory BlobStorageFactory) {
	scheme = strings.ToLower(scheme)
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	if nil == factory {
		panic("storage: RegisterFactory passed a nil factory")
	}
	if _, dup := factories[scheme]; dup {
		panic("storage: RegisterFactory called twice for factory " + scheme)
	}
	factories[scheme] = factory
}

func SupportedSchemes() []string {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	var list []string
	for scheme := range factories {
		list = append(list, scheme)
	}
	return list
}

