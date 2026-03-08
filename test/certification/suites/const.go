//go:build e2e
// +build e2e

package suites

import _ "embed"

const (
	ShortTestLabel = "short"
	FullTestLabel  = "full"
)

//go:embed data/providers/harvester.yaml
var CAPIProviderHarvester []byte

//go:embed data/cluster-templates/harvester-rke2-topology.yaml
var HarvesterRKE2Topology []byte
