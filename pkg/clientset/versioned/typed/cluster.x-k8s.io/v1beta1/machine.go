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

package v1beta1

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	scheme "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/scheme"
)

// MachinesGetter has a method to return a MachineInterface.
// A group's client should implement this interface.
type MachinesGetter interface {
	Machines(namespace string) MachineInterface
}

// MachineInterface has methods to work with Machine resources.
type MachineInterface interface {
	Create(ctx context.Context, machine *clusterv1.Machine, opts v1.CreateOptions) (*clusterv1.Machine, error)
	Update(ctx context.Context, machine *clusterv1.Machine, opts v1.UpdateOptions) (*clusterv1.Machine, error)
	UpdateStatus(ctx context.Context, machine *clusterv1.Machine, opts v1.UpdateOptions) (*clusterv1.Machine, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*clusterv1.Machine, error)
	List(ctx context.Context, opts v1.ListOptions) (*clusterv1.MachineList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *clusterv1.Machine, err error)
	MachineExpansion
}

// machines implements MachineInterface
type machines struct {
	client rest.Interface
	ns     string
}

// newMachines returns a Machines
func newMachines(c *Clusterv1beta1Client, namespace string) *machines {
	return &machines{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the machine, and returns the corresponding machine object, and an error if there is any.
func (c *machines) Get(ctx context.Context, name string, options v1.GetOptions) (result *clusterv1.Machine, err error) {
	result = &clusterv1.Machine{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("machines").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Machines that match those selectors.
func (c *machines) List(ctx context.Context, opts v1.ListOptions) (result *clusterv1.MachineList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &clusterv1.MachineList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("machines").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested machines.
func (c *machines) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("machines").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a machine and creates it.  Returns the server's representation of the machine, and an error, if there is any.
func (c *machines) Create(ctx context.Context, machine *clusterv1.Machine, opts v1.CreateOptions) (result *clusterv1.Machine, err error) {
	result = &clusterv1.Machine{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("machines").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(machine).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a machine and updates it. Returns the server's representation of the machine, and an error, if there is any.
func (c *machines) Update(ctx context.Context, machine *clusterv1.Machine, opts v1.UpdateOptions) (result *clusterv1.Machine, err error) {
	result = &clusterv1.Machine{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("machines").
		Name(machine.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(machine).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *machines) UpdateStatus(ctx context.Context, machine *clusterv1.Machine, opts v1.UpdateOptions) (result *clusterv1.Machine, err error) {
	result = &clusterv1.Machine{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("machines").
		Name(machine.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(machine).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the machine and deletes it. Returns an error if one occurs.
func (c *machines) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("machines").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *machines) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("machines").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched machine.
func (c *machines) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *clusterv1.Machine, err error) {
	result = &clusterv1.Machine{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("machines").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
