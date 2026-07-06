//go:build e2e
// +build e2e

package suites

import (
	"context"

	"sigs.k8s.io/cluster-api/test/framework/bootstrap"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
)

// KindBootstrapCluster creates the kind management cluster WITHOUT mounting the host
// docker socket: only the CAPD provider needs that socket and the certification tiers
// do not use it. It also keeps the suites runnable on podman hosts, where bind-mounting
// a missing /var/run/docker.sock is a hard error instead of an implicit directory
// creation.
func KindBootstrapCluster(ctx context.Context, config *clusterctl.E2EConfig, clusterName, kubernetesVersion string) bootstrap.ClusterProvider {
	return bootstrap.CreateKindBootstrapClusterAndLoadImages(ctx, bootstrap.CreateKindBootstrapClusterAndLoadImagesInput{
		Name:               clusterName,
		KubernetesVersion:  kubernetesVersion,
		RequiresDockerSock: false,
		Images:             config.Images,
	})
}
