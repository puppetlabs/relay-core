/*
Copyright (c) Puppet, Inc.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	nebulapuppetcomv1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeSecretAuths implements SecretAuthInterface
type FakeSecretAuths struct {
	Fake *FakeNebulaV1
	ns   string
}

var secretauthsResource = schema.GroupVersionResource{Group: "nebula.puppet.com", Version: "v1", Resource: "secretauths"}

var secretauthsKind = schema.GroupVersionKind{Group: "nebula.puppet.com", Version: "v1", Kind: "SecretAuth"}

// Get takes name of the secretAuth, and returns the corresponding secretAuth object, and an error if there is any.
func (c *FakeSecretAuths) Get(name string, options v1.GetOptions) (result *nebulapuppetcomv1.SecretAuth, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(secretauthsResource, c.ns, name), &nebulapuppetcomv1.SecretAuth{})

	if obj == nil {
		return nil, err
	}
	return obj.(*nebulapuppetcomv1.SecretAuth), err
}

// List takes label and field selectors, and returns the list of SecretAuths that match those selectors.
func (c *FakeSecretAuths) List(opts v1.ListOptions) (result *nebulapuppetcomv1.SecretAuthList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(secretauthsResource, secretauthsKind, c.ns, opts), &nebulapuppetcomv1.SecretAuthList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &nebulapuppetcomv1.SecretAuthList{ListMeta: obj.(*nebulapuppetcomv1.SecretAuthList).ListMeta}
	for _, item := range obj.(*nebulapuppetcomv1.SecretAuthList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested secretAuths.
func (c *FakeSecretAuths) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(secretauthsResource, c.ns, opts))

}

// Create takes the representation of a secretAuth and creates it.  Returns the server's representation of the secretAuth, and an error, if there is any.
func (c *FakeSecretAuths) Create(secretAuth *nebulapuppetcomv1.SecretAuth) (result *nebulapuppetcomv1.SecretAuth, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(secretauthsResource, c.ns, secretAuth), &nebulapuppetcomv1.SecretAuth{})

	if obj == nil {
		return nil, err
	}
	return obj.(*nebulapuppetcomv1.SecretAuth), err
}

// Update takes the representation of a secretAuth and updates it. Returns the server's representation of the secretAuth, and an error, if there is any.
func (c *FakeSecretAuths) Update(secretAuth *nebulapuppetcomv1.SecretAuth) (result *nebulapuppetcomv1.SecretAuth, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(secretauthsResource, c.ns, secretAuth), &nebulapuppetcomv1.SecretAuth{})

	if obj == nil {
		return nil, err
	}
	return obj.(*nebulapuppetcomv1.SecretAuth), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeSecretAuths) UpdateStatus(secretAuth *nebulapuppetcomv1.SecretAuth) (*nebulapuppetcomv1.SecretAuth, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(secretauthsResource, "status", c.ns, secretAuth), &nebulapuppetcomv1.SecretAuth{})

	if obj == nil {
		return nil, err
	}
	return obj.(*nebulapuppetcomv1.SecretAuth), err
}

// Delete takes name of the secretAuth and deletes it. Returns an error if one occurs.
func (c *FakeSecretAuths) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(secretauthsResource, c.ns, name), &nebulapuppetcomv1.SecretAuth{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSecretAuths) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(secretauthsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &nebulapuppetcomv1.SecretAuthList{})
	return err
}

// Patch applies the patch and returns the patched secretAuth.
func (c *FakeSecretAuths) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *nebulapuppetcomv1.SecretAuth, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(secretauthsResource, c.ns, name, data, subresources...), &nebulapuppetcomv1.SecretAuth{})

	if obj == nil {
		return nil, err
	}
	return obj.(*nebulapuppetcomv1.SecretAuth), err
}
