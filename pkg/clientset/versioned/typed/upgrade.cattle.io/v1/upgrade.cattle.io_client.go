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

package v1

import (
	"net/http"

	v1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"

	rest "k8s.io/client-go/rest"

	"github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned/scheme"
)

type UpgradeV1Interface interface {
	RESTClient() rest.Interface
	PlansGetter
}

// UpgradeV1Client is used to interact with features provided by the upgrade.cattle.io group.
type UpgradeV1Client struct {
	restClient rest.Interface
}

func (c *UpgradeV1Client) Plans(namespace string) PlanInterface {
	return newPlans(c, namespace)
}

// NewForConfig creates a new UpgradeV1Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*UpgradeV1Client, error) {
	config := *c
	err := setConfigDefaults(&config)
	if err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new UpgradeV1Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*UpgradeV1Client, error) {
	config := *c
	err := setConfigDefaults(&config)
	if err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &UpgradeV1Client{client}, nil
}

// NewForConfigOrDie creates a new UpgradeV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *UpgradeV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new UpgradeV1Client for the given RESTClient.
func New(c rest.Interface) *UpgradeV1Client {
	return &UpgradeV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *UpgradeV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
