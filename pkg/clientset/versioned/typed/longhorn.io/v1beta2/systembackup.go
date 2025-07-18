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

package v1beta2

import (
	"context"
	"time"

	v1beta2 "github.com/longhorn/longhorn-manager/k8s/pkg/apis/longhorn/v1beta2"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"

	scheme "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/scheme"
)

// SystemBackupsGetter has a method to return a SystemBackupInterface.
// A group's client should implement this interface.
type SystemBackupsGetter interface {
	SystemBackups(namespace string) SystemBackupInterface
}

// SystemBackupInterface has methods to work with SystemBackup resources.
type SystemBackupInterface interface {
	Create(ctx context.Context, systemBackup *v1beta2.SystemBackup, opts v1.CreateOptions) (*v1beta2.SystemBackup, error)
	Update(ctx context.Context, systemBackup *v1beta2.SystemBackup, opts v1.UpdateOptions) (*v1beta2.SystemBackup, error)
	UpdateStatus(ctx context.Context, systemBackup *v1beta2.SystemBackup, opts v1.UpdateOptions) (*v1beta2.SystemBackup, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta2.SystemBackup, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1beta2.SystemBackupList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.SystemBackup, err error)
	SystemBackupExpansion
}

// systemBackups implements SystemBackupInterface
type systemBackups struct {
	client rest.Interface
	ns     string
}

// newSystemBackups returns a SystemBackups
func newSystemBackups(c *LonghornV1beta2Client, namespace string) *systemBackups {
	return &systemBackups{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the systemBackup, and returns the corresponding systemBackup object, and an error if there is any.
func (c *systemBackups) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta2.SystemBackup, err error) {
	result = &v1beta2.SystemBackup{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("systembackups").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SystemBackups that match those selectors.
func (c *systemBackups) List(ctx context.Context, opts v1.ListOptions) (result *v1beta2.SystemBackupList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1beta2.SystemBackupList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("systembackups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested systemBackups.
func (c *systemBackups) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("systembackups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a systemBackup and creates it.  Returns the server's representation of the systemBackup, and an error, if there is any.
func (c *systemBackups) Create(ctx context.Context, systemBackup *v1beta2.SystemBackup, opts v1.CreateOptions) (result *v1beta2.SystemBackup, err error) {
	result = &v1beta2.SystemBackup{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("systembackups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(systemBackup).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a systemBackup and updates it. Returns the server's representation of the systemBackup, and an error, if there is any.
func (c *systemBackups) Update(ctx context.Context, systemBackup *v1beta2.SystemBackup, opts v1.UpdateOptions) (result *v1beta2.SystemBackup, err error) {
	result = &v1beta2.SystemBackup{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("systembackups").
		Name(systemBackup.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(systemBackup).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *systemBackups) UpdateStatus(ctx context.Context, systemBackup *v1beta2.SystemBackup, opts v1.UpdateOptions) (result *v1beta2.SystemBackup, err error) {
	result = &v1beta2.SystemBackup{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("systembackups").
		Name(systemBackup.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(systemBackup).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the systemBackup and deletes it. Returns an error if one occurs.
func (c *systemBackups) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("systembackups").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *systemBackups) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("systembackups").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched systemBackup.
func (c *systemBackups) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.SystemBackup, err error) {
	result = &v1beta2.SystemBackup{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("systembackups").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
