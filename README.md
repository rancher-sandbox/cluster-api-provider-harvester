# cluster-api-provider-harvester

This project has begun as a [Hack Week 23](https://hackweek.opensuse.org/23/projects/cluster-api-provider-for-harvester) project. It is still in a very early phase. Please do not use in Production.

## What is Cluster API Provider Harvester (CAPHV)

The [Cluster API](https://cluster-api.sigs.k8s.io/) brings declarative, Kubernetes-style APIs to cluster creation, configuration and management.

Cluster API Provider Harvester is __Cluster API Infrastructure Provider__ for provisioning Kubernetes Clusters on [Harvester](https://harvesterhci.io/).

At this stage, the Provider has been tested on a single environment, with Harvester v1.2.0 using two Control Plane/Bootstrap providers: [Kubeadm](https://github.com/kubernetes-sigs/cluster-api/tree/main/controlplane/kubeadm) and [RKE2](https://github.com/rancher-sandbox/cluster-api-provider-rke2).

The [templates](https://github.com/rancher-sandbox/cluster-api-provider-harvester/tree/main/templates) folder contains examples of such configurations.

## Getting Started
Cluster API Provider Harvester is compliant with the `clusterctl` contract, which means that `clusterctl` simplifies its deployment to the CAPI Management Cluster. In this Getting Started guide, we will be using the Harvester Provider with the `RKE2` provider (also called `CAPRKE2`).

### Management Cluster

In order to use this provider, you need to have a management cluster available to you and have your current KUBECONFIG context set to talk to that cluster. If you do not have a cluster available to you, you can create a `kind` cluster. These are the steps needed to achieve that:
1. Ensure kind is installed (https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
2. Create a special `kind` configuration file if you intend to use the Docker infrastructure provider:

```bash
cat > kind-cluster-with-extramounts.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: capi-test
nodes:
- role: control-plane
  extraMounts:
    - hostPath: /var/run/docker.sock
      containerPath: /var/run/docker.sock
EOF
```

3. Run the following command to create a local kind cluster:

```bash
kind create cluster --config kind-cluster-with-extramounts.yaml
```

4. Check your newly created `kind` cluster :

```bash
kubectl cluster-info
```
and get a similar result to this:

```
Kubernetes control plane is running at https://127.0.0.1:40819
CoreDNS is running at https://127.0.0.1:40819/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```

### Setting up clusterctl
Before the Harvester provider can be installed with `clusterctl`, it is necessary to explain to `clusterctl` where to find it, which repository, which type of provider it is, etc. This can be done by creating or modifying the file `$HOME/.cluster-api/clusterctl.yaml`, and adding to it the following content:

```yaml
providers:
  - name: "harvester"
    url: "https://github.com/rancher-sandbox/cluster-api-provider-harvester/releases/latest/infrastructure-components.yaml"
    type: "InfrastructureProvider"
```

Now, the Harvester and RKE2 providers can be installed with the `clusterctl` command. In this particular case, our manifests will be using the `ResourceSet` feature gate for Cluster API, we will need to set the environment variable `EXP_CLUSTER_RESOURCE_SET` to `true` before running the `clusterctl init` command.

```bash
export EXP_CLUSTER_RESOURCE_SET=true

$ clusterctl init --infrastructure harvester --control-plane rke2 --bootstrap rke2
Fetching providers
Installing cert-manager Version="v1.14.5"
Waiting for cert-manager to be available...
Installing Provider="cluster-api" Version="v1.7.2" TargetNamespace="capi-system"
Installing Provider="bootstrap-rke2" Version="v0.3.0" TargetNamespace="rke2-bootstrap-system"
Installing Provider="control-plane-rke2" Version="v0.3.0" TargetNamespace="rke2-control-plane-system"
Installing Provider="infrastructure-harvester" Version="v0.1.2" TargetNamespace="caphv-system"

Your management cluster has been initialized successfully!

You can now create your first workload cluster by running the following:

  clusterctl generate cluster [name] --kubernetes-version [version] | kubectl apply -f -

```

### Create a workload cluster
Now, you can test out the provider by generating some YAML and applying it to the above `kind` cluster. Such YAML templates can be found in `./templates` directory. We will be interested here in the `RKE2` examples under `./templates`. Please be aware that the file [cluster-template-rke2-dhcp.yaml](./templates/cluster-template-rke2-dhcp.yaml) is a template with placeholders: it cannot be applied directly to the cluster. You need to generate a valid YAML file first. In order to do that, you need to set the following environment variables:

```bash
export CLUSTER_NAME=test-rk # Name of the cluster that will be created.
export HARVESTER_ENDPOINT=x.x.x.x # Harvester Clusters IP Adr.
export NAMESPACE=example-rk # Namespace where the cluster will be created.
export KUBERNETES_VERSION=v1.26.6 # Kubernetes Version
export SSH_KEYPAIR=<public-key-name> # should exist in Harvester prior to applying manifest. Should have the format <TARGET_HARVESTER_NAMESPACE>/<NAME>
export VM_IMAGE_NAME=default/jammy-server-cloudimg-amd64.img # Should have the format <TARGET_HARVESTER_NAMESPACE>/<NAME> for an image that exists on Harvester
export CONTROL_PLANE_MACHINE_COUNT=3
export WORKER_MACHINE_COUNT=2
export VM_DISK_SIZE=40Gi # Put here the desired disk size
export RANCHER_TURTLES_LABEL='' # This is used if you are using Rancher CAPI Extension (Turtles) to import the cluster automatically.
export VM_NETWORK=default/untagged # change here according to your Harvester available VM Networks. Should have the format <TARGET_HARVESTER_NAMESPACE>/<NAME>
export HARVESTER_KUBECONFIG_B64=XXXYYY #Full Harvester's kubeconfig encoded in Base64. You can use: cat kubeconfig.yaml | base64
export CLOUD_CONFIG_KUBECONFIG_B64=ZZZZAAA # Kubeconfig generated for the Cloud Provider: https://docs.harvesterhci.io/v1.3/rancher/cloud-provider#deploying-to-the-rke2-custom-cluster-experimental 
export IP_POOL_NAME=default # for the non-DHCP template, specify the IP pool for the Harvester load balancer. The IP pool must exist in Harvester prior to applying manifest
export TARGET_HARVESTER_NAMESPACE=default # the namespace on the Harvester cluster where the VMs, load balancers etc. should be created
```

NOTE: The `CLOUD_CONFIG_KUBECONFIG_B64` variable content should be the result of the script available [here](https://docs.harvesterhci.io/v1.3/rancher/cloud-provider#deploying-to-the-rke2-custom-cluster-experimental) -- meaning, the generated kubeconfig -- encoded in BASE64.

Now, we can generate the YAML using the following command:

```bash
clusterctl generate yaml --from https://github.com/rancher-sandbox/cluster-api-provider-harvester/blob/main/templates/cluster-template-rke2.yaml > harvester-rke2-clusterctl.yaml
```

After examining the resulting YAML file, you can apply it to the management cluster:
```bash
kubectl apply -f harvester-rke2-clusterctl.yaml
```

You should see the following output:
```bash
namespace/example-rk created
cluster.cluster.x-k8s.io/test-rk created
harvestercluster.infrastructure.cluster.x-k8s.io/test-rk-hv created
secret/hv-identity-secret created
rke2controlplane.controlplane.cluster.x-k8s.io/test-rk-control-plane created
rke2configtemplate.bootstrap.cluster.x-k8s.io/test-rk-worker created
machinedeployment.cluster.x-k8s.io/test-rk-workers created
harvestermachinetemplate.infrastructure.cluster.x-k8s.io/test-rk-wk-machine created
harvestermachinetemplate.infrastructure.cluster.x-k8s.io/test-rk-cp-machine created
clusterresourceset.addons.cluster.x-k8s.io/crs-harvester-ccm created
clusterresourceset.addons.cluster.x-k8s.io/crs-harvester-csi created
clusterresourceset.addons.cluster.x-k8s.io/crs-calico-chart-config created
configmap/cloud-controller-manager-addon created
configmap/harvester-csi-driver-addon created
configmap/calico-helm-config created
```

### Checking the workload cluster:
After a while you should be able to check functionality of the workload cluster using `clusterctl`:

```bash
clusterctl describe cluster -n example-rk test-rk
```

and once the cluster is provisioned, it should look similar to the following:

```
NAME                                                     READY  SEVERITY  REASON  SINCE  MESSAGE
Cluster/test-rk                                          True                     7h35m
├─ClusterInfrastructure - HarvesterCluster/test-rk-hv
├─ControlPlane - RKE2ControlPlane/test-rk-control-plane  True                     7h35m
│ └─3 Machines...                                        True                     7h45m  See test-rk-control-plane-dmrg5, test-rk-control-plane-jkdrb, ...
└─Workers
  └─MachineDeployment/test-rk-workers                    True                     7h46m
    └─2 Machines...                                      True                     7h46m  See test-rk-workers-jwjdg-sz7qk, test-rk-workers-jwjdg-vxgbx
```
