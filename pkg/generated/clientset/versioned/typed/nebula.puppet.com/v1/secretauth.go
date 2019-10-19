/*
Copyright (c) Puppet, Inc.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	scheme "github.com/puppetlabs/nebula-tasks/pkg/generated/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SecretAuthsGetter has a method to return a SecretAuthInterface.
// A group's client should implement this interface.
type SecretAuthsGetter interface {
	SecretAuths(namespace string) SecretAuthInterface
}

// SecretAuthInterface has methods to work with SecretAuth resources.
type SecretAuthInterface interface {
	Create(*v1.SecretAuth) (*v1.SecretAuth, error)
	Update(*v1.SecretAuth) (*v1.SecretAuth, error)
	UpdateStatus(*v1.SecretAuth) (*v1.SecretAuth, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.SecretAuth, error)
	List(opts metav1.ListOptions) (*v1.SecretAuthList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.SecretAuth, err error)
	SecretAuthExpansion
}

// secretAuths implements SecretAuthInterface
type secretAuths struct {
	client rest.Interface
	ns     string
}

// newSecretAuths returns a SecretAuths
func newSecretAuths(c *NebulaV1Client, namespace string) *secretAuths {
	return &secretAuths{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the secretAuth, and returns the corresponding secretAuth object, and an error if there is any.
func (c *secretAuths) Get(name string, options metav1.GetOptions) (result *v1.SecretAuth, err error) {
	result = &v1.SecretAuth{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("secretauths").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SecretAuths that match those selectors.
func (c *secretAuths) List(opts metav1.ListOptions) (result *v1.SecretAuthList, err error) {
	result = &v1.SecretAuthList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("secretauths").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested secretAuths.
func (c *secretAuths) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("secretauths").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a secretAuth and creates it.  Returns the server's representation of the secretAuth, and an error, if there is any.
func (c *secretAuths) Create(secretAuth *v1.SecretAuth) (result *v1.SecretAuth, err error) {
	result = &v1.SecretAuth{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("secretauths").
		Body(secretAuth).
		Do().
		Into(result)
	return
}

// Update takes the representation of a secretAuth and updates it. Returns the server's representation of the secretAuth, and an error, if there is any.
func (c *secretAuths) Update(secretAuth *v1.SecretAuth) (result *v1.SecretAuth, err error) {
	result = &v1.SecretAuth{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("secretauths").
		Name(secretAuth.Name).
		Body(secretAuth).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *secretAuths) UpdateStatus(secretAuth *v1.SecretAuth) (result *v1.SecretAuth, err error) {
	result = &v1.SecretAuth{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("secretauths").
		Name(secretAuth.Name).
		SubResource("status").
		Body(secretAuth).
		Do().
		Into(result)
	return
}

// Delete takes name of the secretAuth and deletes it. Returns an error if one occurs.
func (c *secretAuths) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("secretauths").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *secretAuths) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("secretauths").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched secretAuth.
func (c *secretAuths) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.SecretAuth, err error) {
	result = &v1.SecretAuth{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("secretauths").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
