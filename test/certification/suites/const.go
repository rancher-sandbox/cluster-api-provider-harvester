//go:build e2e
// +build e2e

package suites

import _ "embed"

const (
	ShortTestLabel = "short"
	FullTestLabel  = "full"
)

// Provider manifests in cluster-api-operator format (operator.cluster.x-k8s.io/v1alpha2).
// They are envsubst templates: ${CAPI_VERSION}, ${CAPHV_VERSION} and ${CAPHV_COMPONENTS_URL}
// are resolved from the e2e config variables (exported to the environment by LoadE2EConfig)
// when applied through turtles' ApplyFromTemplate.

//go:embed data/providers/core.yaml
var CoreProviderCAPI []byte

//go:embed data/providers/rke2.yaml
var RKE2Providers []byte

//go:embed data/providers/harvester.yaml
var InfrastructureProviderHarvester []byte

// CAPIProviderHarvesterTurtles is the CAPHV provider in Rancher Turtles CAPIProvider
// format, used by the tier A (Rancher + Turtles) certification.
//
//go:embed data/providers/harvester-capiprovider.yaml
var CAPIProviderHarvesterTurtles []byte

// CAPIProviderRKE2Turtles declares the RKE2 providers in Rancher Turtles CAPIProvider
// format (tier A): a fresh Rancher does not install them by itself.
//
//go:embed data/providers/rke2-capiprovider.yaml
var CAPIProviderRKE2Turtles []byte

// HarvesterRKE2Topology is the full ClusterClass + Cluster template used by the on-demand
// e2e tier (real Harvester provisioning). Kept here for that suite; unused by the
// version-pairing certification.
//
//go:embed data/cluster-templates/harvester-rke2-topology.yaml
var HarvesterRKE2Topology []byte
