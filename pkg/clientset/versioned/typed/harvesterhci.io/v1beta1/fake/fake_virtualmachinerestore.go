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

// FakeVirtualMachineRestores implements VirtualMachineRestoreInterface
type FakeVirtualMachineRestores struct {
	Fake *FakeHarvesterhciV1beta1
	ns   string
}

var virtualmachinerestoresResource = schema.GroupVersionResource{Group: "harvesterhci.io", Version: "v1beta1", Resource: "virtualmachinerestores"}

var virtualmachinerestoresKind = schema.GroupVersionKind{Group: "harvesterhci.io", Version: "v1beta1", Kind: "VirtualMachineRestore"}

// Get takes name of the virtualMachineRestore, and returns the corresponding virtualMachineRestore object, and an error if there is any.
func (c *FakeVirtualMachineRestores) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.VirtualMachineRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(virtualmachinerestoresResource, c.ns, name), &v1beta1.VirtualMachineRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.VirtualMachineRestore), err
}

// List takes label and field selectors, and returns the list of VirtualMachineRestores that match those selectors.
func (c *FakeVirtualMachineRestores) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.VirtualMachineRestoreList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(virtualmachinerestoresResource, virtualmachinerestoresKind, c.ns, opts), &v1beta1.VirtualMachineRestoreList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.VirtualMachineRestoreList{ListMeta: obj.(*v1beta1.VirtualMachineRestoreList).ListMeta}
	for _, item := range obj.(*v1beta1.VirtualMachineRestoreList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested virtualMachineRestores.
func (c *FakeVirtualMachineRestores) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(virtualmachinerestoresResource, c.ns, opts))

}

// Create takes the representation of a virtualMachineRestore and creates it.  Returns the server's representation of the virtualMachineRestore, and an error, if there is any.
func (c *FakeVirtualMachineRestores) Create(ctx context.Context, virtualMachineRestore *v1beta1.VirtualMachineRestore, opts v1.CreateOptions) (result *v1beta1.VirtualMachineRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(virtualmachinerestoresResource, c.ns, virtualMachineRestore), &v1beta1.VirtualMachineRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.VirtualMachineRestore), err
}

// Update takes the representation of a virtualMachineRestore and updates it. Returns the server's representation of the virtualMachineRestore, and an error, if there is any.
func (c *FakeVirtualMachineRestores) Update(ctx context.Context, virtualMachineRestore *v1beta1.VirtualMachineRestore, opts v1.UpdateOptions) (result *v1beta1.VirtualMachineRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(virtualmachinerestoresResource, c.ns, virtualMachineRestore), &v1beta1.VirtualMachineRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.VirtualMachineRestore), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeVirtualMachineRestores) UpdateStatus(ctx context.Context, virtualMachineRestore *v1beta1.VirtualMachineRestore, opts v1.UpdateOptions) (*v1beta1.VirtualMachineRestore, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(virtualmachinerestoresResource, "status", c.ns, virtualMachineRestore), &v1beta1.VirtualMachineRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.VirtualMachineRestore), err
}

// Delete takes name of the virtualMachineRestore and deletes it. Returns an error if one occurs.
func (c *FakeVirtualMachineRestores) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(virtualmachinerestoresResource, c.ns, name, opts), &v1beta1.VirtualMachineRestore{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVirtualMachineRestores) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(virtualmachinerestoresResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.VirtualMachineRestoreList{})
	return err
}

// Patch applies the patch and returns the patched virtualMachineRestore.
func (c *FakeVirtualMachineRestores) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.VirtualMachineRestore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(virtualmachinerestoresResource, c.ns, name, pt, data, subresources...), &v1beta1.VirtualMachineRestore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.VirtualMachineRestore), err
}
