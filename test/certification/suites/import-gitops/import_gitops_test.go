//go:build e2e
// +build e2e

package import_gitops

import (
	. "github.com/onsi/ginkgo/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/rancher/turtles/test/e2e"
	"github.com/rancher/turtles/test/e2e/specs"

	"github.com/jniedergang/cluster-api-provider-harvester/test/certification/suites"
)

var _ = Describe("[Harvester] [RKE2] Create and import CAPI cluster via Turtles", Label(suites.FullTestLabel), func() {
	BeforeEach(func() {
		komega.SetClient(bootstrapClusterProxy.GetClient())
		komega.SetContext(ctx)
	})

	specs.CreateUsingGitOpsSpec(ctx, func() specs.CreateUsingGitOpsSpecInput {
		return specs.CreateUsingGitOpsSpecInput{
			E2EConfig:             e2e.LoadE2EConfig(),
			BootstrapClusterProxy: bootstrapClusterProxy,
			ClusterTemplate:       suites.HarvesterRKE2Topology,
			ClusterName:           "cluster-harvester-rke2",

			ControlPlaneMachineCount: ptr.To(1),
			WorkerMachineCount:       ptr.To(0),

			LabelNamespace:      true,
			TestClusterReimport: false,
			RancherServerURL:    hostName,

			CAPIClusterCreateWaitName: "wait-harvester-create-cluster",
			DeleteClusterWaitName:     "wait-delete-cluster",

			CapiClusterOwnerLabel:          e2e.CapiClusterOwnerLabel,
			CapiClusterOwnerNamespaceLabel: e2e.CapiClusterOwnerNamespaceLabel,
			OwnedLabelName:                 e2e.OwnedLabelName,
		}
	})
})
