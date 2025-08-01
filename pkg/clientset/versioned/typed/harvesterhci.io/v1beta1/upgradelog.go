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

	v1beta1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"

	scheme "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/scheme"
)

// UpgradeLogsGetter has a method to return a UpgradeLogInterface.
// A group's client should implement this interface.
type UpgradeLogsGetter interface {
	UpgradeLogs(namespace string) UpgradeLogInterface
}

// UpgradeLogInterface has methods to work with UpgradeLog resources.
type UpgradeLogInterface interface {
	Create(ctx context.Context, upgradeLog *v1beta1.UpgradeLog, opts v1.CreateOptions) (*v1beta1.UpgradeLog, error)
	Update(ctx context.Context, upgradeLog *v1beta1.UpgradeLog, opts v1.UpdateOptions) (*v1beta1.UpgradeLog, error)
	UpdateStatus(ctx context.Context, upgradeLog *v1beta1.UpgradeLog, opts v1.UpdateOptions) (*v1beta1.UpgradeLog, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta1.UpgradeLog, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1beta1.UpgradeLogList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.UpgradeLog, err error)
	UpgradeLogExpansion
}

// upgradeLogs implements UpgradeLogInterface
type upgradeLogs struct {
	client rest.Interface
	ns     string
}

// newUpgradeLogs returns a UpgradeLogs
func newUpgradeLogs(c *HarvesterhciV1beta1Client, namespace string) *upgradeLogs {
	return &upgradeLogs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the upgradeLog, and returns the corresponding upgradeLog object, and an error if there is any.
func (c *upgradeLogs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.UpgradeLog, err error) {
	result = &v1beta1.UpgradeLog{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("upgradelogs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of UpgradeLogs that match those selectors.
func (c *upgradeLogs) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.UpgradeLogList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1beta1.UpgradeLogList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("upgradelogs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested upgradeLogs.
func (c *upgradeLogs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("upgradelogs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a upgradeLog and creates it.  Returns the server's representation of the upgradeLog, and an error, if there is any.
func (c *upgradeLogs) Create(ctx context.Context, upgradeLog *v1beta1.UpgradeLog, opts v1.CreateOptions) (result *v1beta1.UpgradeLog, err error) {
	result = &v1beta1.UpgradeLog{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("upgradelogs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(upgradeLog).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a upgradeLog and updates it. Returns the server's representation of the upgradeLog, and an error, if there is any.
func (c *upgradeLogs) Update(ctx context.Context, upgradeLog *v1beta1.UpgradeLog, opts v1.UpdateOptions) (result *v1beta1.UpgradeLog, err error) {
	result = &v1beta1.UpgradeLog{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("upgradelogs").
		Name(upgradeLog.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(upgradeLog).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *upgradeLogs) UpdateStatus(ctx context.Context, upgradeLog *v1beta1.UpgradeLog, opts v1.UpdateOptions) (result *v1beta1.UpgradeLog, err error) {
	result = &v1beta1.UpgradeLog{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("upgradelogs").
		Name(upgradeLog.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(upgradeLog).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the upgradeLog and deletes it. Returns an error if one occurs.
func (c *upgradeLogs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("upgradelogs").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *upgradeLogs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("upgradelogs").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched upgradeLog.
func (c *upgradeLogs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.UpgradeLog, err error) {
	result = &v1beta1.UpgradeLog{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("upgradelogs").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
