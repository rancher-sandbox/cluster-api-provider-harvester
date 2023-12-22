/*
Copyright 2023 Rancher Labs, Inc.

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
	lbv1 "github.com/harvester/harvester-load-balancer/pkg/generated/clientset/versioned/typed/loadbalancer.harvesterhci.io/v1beta1"
	fakelbv1 "github.com/harvester/harvester-load-balancer/pkg/generated/clientset/versioned/typed/loadbalancer.harvesterhci.io/v1beta1/fake"
	clientset "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned"
	catalogv1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/catalog.cattle.io/v1"
	fakecatalogv1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/catalog.cattle.io/v1/fake"
	clusterv1alpha4 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/cluster.x-k8s.io/v1alpha4"
	fakeclusterv1alpha4 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/cluster.x-k8s.io/v1alpha4/fake"
	harvesterhciv1beta1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/harvesterhci.io/v1beta1"
	fakeharvesterhciv1beta1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/harvesterhci.io/v1beta1/fake"
	k8scnicncfiov1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/k8s.cni.cncf.io/v1"
	fakek8scnicncfiov1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/k8s.cni.cncf.io/v1/fake"
	kubevirtv1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/kubevirt.io/v1"
	fakekubevirtv1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/kubevirt.io/v1/fake"
	longhornv1beta2 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/longhorn.io/v1beta2"
	fakelonghornv1beta2 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/longhorn.io/v1beta2/fake"
	managementv3 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/management.cattle.io/v3"
	fakemanagementv3 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/management.cattle.io/v3/fake"
	monitoringv1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/monitoring.coreos.com/v1"
	fakemonitoringv1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/monitoring.coreos.com/v1/fake"
	networkingv1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/networking.k8s.io/v1"
	fakenetworkingv1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/networking.k8s.io/v1/fake"
	snapshotv1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/snapshot.storage.k8s.io/v1"
	fakesnapshotv1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/snapshot.storage.k8s.io/v1/fake"
	storagev1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/storage.k8s.io/v1"
	fakestoragev1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/storage.k8s.io/v1/fake"
	upgradev1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/upgrade.cattle.io/v1"
	fakeupgradev1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/upgrade.cattle.io/v1/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	"k8s.io/client-go/testing"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	cs := &Clientset{tracker: o}
	cs.discovery = &fakediscovery.FakeDiscovery{Fake: &cs.Fake}
	cs.AddReactor("*", "*", testing.ObjectReaction(o))
	cs.AddWatchReactor("*", func(action testing.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := o.Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		return true, watch, nil
	})

	return cs
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
	tracker   testing.ObjectTracker
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

func (c *Clientset) Tracker() testing.ObjectTracker {
	return c.tracker
}

var (
	_ clientset.Interface = &Clientset{}
	_ testing.FakeClient  = &Clientset{}
)

// CatalogV1 retrieves the CatalogV1Client
func (c *Clientset) CatalogV1() catalogv1.CatalogV1Interface {
	return &fakecatalogv1.FakeCatalogV1{Fake: &c.Fake}
}

// ClusterV1alpha4 retrieves the ClusterV1alpha4Client
func (c *Clientset) ClusterV1alpha4() clusterv1alpha4.ClusterV1alpha4Interface {
	return &fakeclusterv1alpha4.FakeClusterV1alpha4{Fake: &c.Fake}
}

// HarvesterhciV1beta1 retrieves the HarvesterhciV1beta1Client
func (c *Clientset) HarvesterhciV1beta1() harvesterhciv1beta1.HarvesterhciV1beta1Interface {
	return &fakeharvesterhciv1beta1.FakeHarvesterhciV1beta1{Fake: &c.Fake}
}

// K8sCniCncfIoV1 retrieves the K8sCniCncfIoV1Client
func (c *Clientset) K8sCniCncfIoV1() k8scnicncfiov1.K8sCniCncfIoV1Interface {
	return &fakek8scnicncfiov1.FakeK8sCniCncfIoV1{Fake: &c.Fake}
}

// KubevirtV1 retrieves the KubevirtV1Client
func (c *Clientset) KubevirtV1() kubevirtv1.KubevirtV1Interface {
	return &fakekubevirtv1.FakeKubevirtV1{Fake: &c.Fake}
}

// LonghornV1beta2 retrieves the LonghornV1beta2Client
func (c *Clientset) LonghornV1beta2() longhornv1beta2.LonghornV1beta2Interface {
	return &fakelonghornv1beta2.FakeLonghornV1beta2{Fake: &c.Fake}
}

// ManagementV3 retrieves the ManagementV3Client
func (c *Clientset) ManagementV3() managementv3.ManagementV3Interface {
	return &fakemanagementv3.FakeManagementV3{Fake: &c.Fake}
}

// MonitoringV1 retrieves the MonitoringV1Client
func (c *Clientset) MonitoringV1() monitoringv1.MonitoringV1Interface {
	return &fakemonitoringv1.FakeMonitoringV1{Fake: &c.Fake}
}

// NetworkingV1 retrieves the NetworkingV1Client
func (c *Clientset) NetworkingV1() networkingv1.NetworkingV1Interface {
	return &fakenetworkingv1.FakeNetworkingV1{Fake: &c.Fake}
}

// SnapshotV1 retrieves the SnapshotV1Client
func (c *Clientset) SnapshotV1() snapshotv1.SnapshotV1Interface {
	return &fakesnapshotv1.FakeSnapshotV1{Fake: &c.Fake}
}

// StorageV1 retrieves the StorageV1Client
func (c *Clientset) StorageV1() storagev1.StorageV1Interface {
	return &fakestoragev1.FakeStorageV1{Fake: &c.Fake}
}

// UpgradeV1 retrieves the UpgradeV1Client
func (c *Clientset) UpgradeV1() upgradev1.UpgradeV1Interface {
	return &fakeupgradev1.FakeUpgradeV1{Fake: &c.Fake}
}

// CoreV1 retrieves the CoreV1Client
func (c *Clientset) CoreV1() corev1.CoreV1Interface {
	return &fakecorev1.FakeCoreV1{Fake: &c.Fake}
}

// LoadbalancerV1beta1 retrieves the LoadbalancerV1beta1Client
func (c *Clientset) LoadbalancerV1beta1() lbv1.LoadbalancerV1beta1Interface {
	return &fakelbv1.FakeLoadbalancerV1beta1{Fake: &c.Fake}
}
