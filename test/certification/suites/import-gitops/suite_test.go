//go:build e2e
// +build e2e

package import_gitops

import (
	"context"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/json"
	capiframework "sigs.k8s.io/cluster-api/test/framework"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/rancher/turtles/test/e2e"
	"github.com/rancher/turtles/test/testenv"
)

var (
	hostName string
	ctx      = context.Background()

	setupClusterResult    *testenv.SetupTestClusterResult
	bootstrapClusterProxy capiframework.ClusterProxy
)

func TestCAPHVCertification(t *testing.T) {
	RegisterFailHandler(Fail)

	ctrl.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	RunSpecs(t, "caphv-turtles-certification")
}

var _ = SynchronizedBeforeSuite(
	func() []byte {
		e2eConfig := e2e.LoadE2EConfig()
		e2eConfig.ManagementClusterName = e2eConfig.ManagementClusterName + "-caphv-cert"

		By("Setting up the management cluster proxy (USE_EXISTING_CLUSTER=true)")
		setupClusterResult = testenv.SetupTestCluster(ctx, testenv.SetupTestClusterInput{
			E2EConfig: e2eConfig,
			Scheme:    e2e.InitScheme(),
		})

		By("Management cluster proxy is ready")

		data, err := json.Marshal(e2e.Setup{
			ClusterName:     setupClusterResult.ClusterName,
			KubeconfigPath:  setupClusterResult.KubeconfigPath,
			RancherHostname: e2eConfig.MustGetVariable(e2e.RancherHostnameVar),
		})
		Expect(err).ToNot(HaveOccurred())

		return data
	},
	func(sharedData []byte) {
		setup := e2e.Setup{}
		Expect(json.Unmarshal(sharedData, &setup)).To(Succeed())

		hostName = setup.RancherHostname

		bootstrapClusterProxy = capiframework.NewClusterProxy(
			setup.ClusterName,
			setup.KubeconfigPath,
			e2e.InitScheme(),
			capiframework.WithMachineLogCollector(capiframework.DockerLogCollector{}),
		)
		Expect(bootstrapClusterProxy).ToNot(BeNil(), "cluster proxy should not be nil")
	},
)

var _ = SynchronizedAfterSuite(
	func() {},
	func() {
		config := e2e.LoadE2EConfig()
		skipCleanup, _ := strconv.ParseBool(config.GetVariableOrEmpty(e2e.SkipResourceCleanupVar))
		if skipCleanup {
			return
		}

		testenv.CleanupTestCluster(ctx, testenv.CleanupTestClusterInput{
			SetupTestClusterResult: *setupClusterResult,
		})
	},
)
