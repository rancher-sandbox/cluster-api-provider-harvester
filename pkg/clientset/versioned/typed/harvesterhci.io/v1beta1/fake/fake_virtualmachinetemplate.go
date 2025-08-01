/*
Copyright 2025 SUSE.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by main. DO NOT EDIT.

package fake

import (
	"context"

	v1beta1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVirtualMachineTemplates implements VirtualMachineTemplateInterface
type FakeVirtualMachineTemplates struct {
	Fake *FakeHarvesterhciV1beta1
	ns   string
}

var virtualmachinetemplatesResource = schema.GroupVersionResource{Group: "harvesterhci.io", Version: "v1beta1", Resource: "virtualmachinetemplates"}

var virtualmachinetemplatesKind = schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "VirtualMachineTemplate"}

// Get takes name of the virtualMachineTemplate, and returns the corresponding virtualMachineTemplate object, and an error if there is any.
func (c *FakeVirtualMachineTemplates) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.VirtualMachineTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(virtualmachinetemplatesResource, c.ns, name), &v1beta1.VirtualMachineTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.VirtualMachineTemplate), err
}

// List takes label and field selectors, and returns the list of VirtualMachineTemplates that match those selectors.
func (c *FakeVirtualMachineTemplates) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.VirtualMachineTemplateList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(virtualmachinetemplatesResource, virtualmachinetemplatesKind, c.ns, opts), &v1beta1.VirtualMachineTemplateList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.VirtualMachineTemplateList{ListMeta: obj.(*v1beta1.VirtualMachineTemplateList).ListMeta}
	for _, item := range obj.(*v1beta1.VirtualMachineTemplateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested virtualMachineTemplates.
func (c *FakeVirtualMachineTemplates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(virtualmachinetemplatesResource, c.ns, opts))

}

// Create takes the representation of a virtualMachineTemplate and creates it.  Returns the server's representation of the virtualMachineTemplate, and an error, if there is any.
func (c *FakeVirtualMachineTemplates) Create(ctx context.Context, virtualMachineTemplate *v1beta1.VirtualMachineTemplate, opts v1.CreateOptions) (result *v1beta1.VirtualMachineTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(virtualmachinetemplatesResource, c.ns, virtualMachineTemplate), &v1beta1.VirtualMachineTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.VirtualMachineTemplate), err
}

// Update takes the representation of a virtualMachineTemplate and updates it. Returns the server's representation of the virtualMachineTemplate, and an error, if there is any.
func (c *FakeVirtualMachineTemplates) Update(ctx context.Context, virtualMachineTemplate *v1beta1.VirtualMachineTemplate, opts v1.UpdateOptions) (result *v1beta1.VirtualMachineTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(virtualmachinetemplatesResource, c.ns, virtualMachineTemplate), &v1beta1.VirtualMachineTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.VirtualMachineTemplate), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeVirtualMachineTemplates) UpdateStatus(ctx context.Context, virtualMachineTemplate *v1beta1.VirtualMachineTemplate, opts v1.UpdateOptions) (*v1beta1.VirtualMachineTemplate, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(virtualmachinetemplatesResource, "status", c.ns, virtualMachineTemplate), &v1beta1.VirtualMachineTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.VirtualMachineTemplate), err
}

// Delete takes name of the virtualMachineTemplate and deletes it. Returns an error if one occurs.
func (c *FakeVirtualMachineTemplates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(virtualmachinetemplatesResource, c.ns, name, opts), &v1beta1.VirtualMachineTemplate{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVirtualMachineTemplates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(virtualmachinetemplatesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.VirtualMachineTemplateList{})
	return err
}

// Patch applies the patch and returns the patched virtualMachineTemplate.
func (c *FakeVirtualMachineTemplates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.VirtualMachineTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(virtualmachinetemplatesResource, c.ns, name, pt, data, subresources...), &v1beta1.VirtualMachineTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.VirtualMachineTemplate), err
}
