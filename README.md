# cluster-api-provider-harvester

This project has begun as a [Hack Week 23](https://hackweek.opensuse.org/23/projects/cluster-api-provider-for-harvester) project. It is still in a very early phase. Please do not use in Production.

## What is Cluster API Provider Harvester (CAPHV)

The [Cluster API](https://cluster-api.sigs.k8s.io/) brings declarative, Kubernetes-style APIs to cluster creation, configuration and management.

Cluster API Provider Harvester is __Cluster API Infrastructure Provider__ for provisioning Kubernetes Clusters on [Harvester](https://harvesterhci.io/).

At this stage, the Provider has been tested on a single environment, with Harvester v1.2.0 using two Control Plane/Bootstrap providers: [Kubeadm](https://github.com/kubernetes-sigs/cluster-api/tree/main/controlplane/kubeadm) and [RKE2](https://github.com/rancher-sandbox/cluster-api-provider-rke2).

The [samples](./samples/) folder contains examples of such configurations.
