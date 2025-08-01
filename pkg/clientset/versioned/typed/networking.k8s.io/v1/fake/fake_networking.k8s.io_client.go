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
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"

	v1 "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/typed/networking.k8s.io/v1"
)

type FakeNetworkingV1 struct {
	*testing.Fake
}

func (c *FakeNetworkingV1) Ingresses(namespace string) v1.IngressInterface {
	return &FakeIngresses{c, namespace}
}

func (c *FakeNetworkingV1) IngressClasses() v1.IngressClassInterface {
	return &FakeIngressClasses{c}
}

func (c *FakeNetworkingV1) NetworkPolicies(namespace string) v1.NetworkPolicyInterface {
	return &FakeNetworkPolicies{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeNetworkingV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
