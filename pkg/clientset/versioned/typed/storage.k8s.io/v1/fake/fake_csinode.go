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

	storagev1 "k8s.io/api/storage/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeCSINodes implements CSINodeInterface
type FakeCSINodes struct {
	Fake *FakeStorageV1
}

var csinodesResource = schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "csinodes"}

var csinodesKind = schema.GroupVersionKind{Group: "storage.k8s.io", Version: "v1", Kind: "CSINode"}

// Get takes name of the cSINode, and returns the corresponding cSINode object, and an error if there is any.
func (c *FakeCSINodes) Get(ctx context.Context, name string, options v1.GetOptions) (result *storagev1.CSINode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(csinodesResource, name), &storagev1.CSINode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*storagev1.CSINode), err
}

// List takes label and field selectors, and returns the list of CSINodes that match those selectors.
func (c *FakeCSINodes) List(ctx context.Context, opts v1.ListOptions) (result *storagev1.CSINodeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(csinodesResource, csinodesKind, opts), &storagev1.CSINodeList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &storagev1.CSINodeList{ListMeta: obj.(*storagev1.CSINodeList).ListMeta}
	for _, item := range obj.(*storagev1.CSINodeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested cSINodes.
func (c *FakeCSINodes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(csinodesResource, opts))
}

// Create takes the representation of a cSINode and creates it.  Returns the server's representation of the cSINode, and an error, if there is any.
func (c *FakeCSINodes) Create(ctx context.Context, cSINode *storagev1.CSINode, opts v1.CreateOptions) (result *storagev1.CSINode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(csinodesResource, cSINode), &storagev1.CSINode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*storagev1.CSINode), err
}

// Update takes the representation of a cSINode and updates it. Returns the server's representation of the cSINode, and an error, if there is any.
func (c *FakeCSINodes) Update(ctx context.Context, cSINode *storagev1.CSINode, opts v1.UpdateOptions) (result *storagev1.CSINode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(csinodesResource, cSINode), &storagev1.CSINode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*storagev1.CSINode), err
}

// Delete takes name of the cSINode and deletes it. Returns an error if one occurs.
func (c *FakeCSINodes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(csinodesResource, name, opts), &storagev1.CSINode{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeCSINodes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(csinodesResource, listOpts)

	_, err := c.Fake.Invokes(action, &storagev1.CSINodeList{})
	return err
}

// Patch applies the patch and returns the patched cSINode.
func (c *FakeCSINodes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *storagev1.CSINode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(csinodesResource, name, pt, data, subresources...), &storagev1.CSINode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*storagev1.CSINode), err
}
