/*
Copyright (c) Puppet, Inc.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// SecretAuthLister helps list SecretAuths.
type SecretAuthLister interface {
	// List lists all SecretAuths in the indexer.
	List(selector labels.Selector) (ret []*v1.SecretAuth, err error)
	// SecretAuths returns an object that can list and get SecretAuths.
	SecretAuths(namespace string) SecretAuthNamespaceLister
	SecretAuthListerExpansion
}

// secretAuthLister implements the SecretAuthLister interface.
type secretAuthLister struct {
	indexer cache.Indexer
}

// NewSecretAuthLister returns a new SecretAuthLister.
func NewSecretAuthLister(indexer cache.Indexer) SecretAuthLister {
	return &secretAuthLister{indexer: indexer}
}

// List lists all SecretAuths in the indexer.
func (s *secretAuthLister) List(selector labels.Selector) (ret []*v1.SecretAuth, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.SecretAuth))
	})
	return ret, err
}

// SecretAuths returns an object that can list and get SecretAuths.
func (s *secretAuthLister) SecretAuths(namespace string) SecretAuthNamespaceLister {
	return secretAuthNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// SecretAuthNamespaceLister helps list and get SecretAuths.
type SecretAuthNamespaceLister interface {
	// List lists all SecretAuths in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.SecretAuth, err error)
	// Get retrieves the SecretAuth from the indexer for a given namespace and name.
	Get(name string) (*v1.SecretAuth, error)
	SecretAuthNamespaceListerExpansion
}

// secretAuthNamespaceLister implements the SecretAuthNamespaceLister
// interface.
type secretAuthNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all SecretAuths in the indexer for a given namespace.
func (s secretAuthNamespaceLister) List(selector labels.Selector) (ret []*v1.SecretAuth, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.SecretAuth))
	})
	return ret, err
}

// Get retrieves the SecretAuth from the indexer for a given namespace and name.
func (s secretAuthNamespaceLister) Get(name string) (*v1.SecretAuth, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("secretauth"), name)
	}
	return obj.(*v1.SecretAuth), nil
}
