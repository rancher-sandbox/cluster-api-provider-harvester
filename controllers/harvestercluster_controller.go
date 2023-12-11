/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	lbv1beta1 "github.com/harvester/harvester-load-balancer/pkg/apis/loadbalancer.harvesterhci.io/v1beta1"
	lbclient "github.com/harvester/harvester-load-balancer/pkg/generated/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	infrav1 "github.com/rancher-sandbox/cluster-api-provider-harvester/api/v1alpha1"
	locutil "github.com/rancher-sandbox/cluster-api-provider-harvester/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	harvesterNamespace        = "harvester-system"
	harvesterDeploymentName   = "harvester"
	availableConditionType    = "Available"
	apiServerListener         = "Listener of API Server"
	apiServerLBPort           = 6443
	apiServerBackendPort      = 6443
	apiServerProtocol         = "TCP"
	cpIPPoolDescriptionPrefix = "IP Pool for the control plane's LB of cluster"
	// TODO: Re-do the following when the HarvesterMachine API is done.
	cpVMLabelKey         = "harvestercluster/machinetype"
	cpVMLabelValuePrefix = "controlplane"
)

// HarvesterClusterReconciler reconciles a HarvesterCluster object
type HarvesterClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status;machinesets;machines;machines/status;machinepools;machinepools/status,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *HarvesterClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling HarvesterCluster", "cluster-name", req.NamespacedName.Name, "cluster-namespace", req.NamespacedName.Namespace)

	var cluster infrav1.HarvesterCluster
	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {

		if apierrors.IsNotFound(err) {
			logger.Error(err, "cluster not found", "cluster-name", req.NamespacedName.Name, "cluster-namespace", req.NamespacedName.Namespace)
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Error happened when getting harvestercluster", "cluster-name", req.NamespacedName.Name, "cluster-namespace", req.NamespacedName.Namespace)
		return ctrl.Result{}, err
	}
	patchHelper, err := patch.NewHelper(&cluster, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		if err := patchHelper.Patch(ctx, &cluster); err != nil {
			clusterString := cluster.Namespace + "/" + cluster.Name
			logger.Error(err, "unable to patch", "cluster", clusterString)
		}
	}()

	clusterOwner, err := capiutil.GetOwnerCluster(ctx, r.Client, cluster.ObjectMeta)
	if err != nil {
		logger.Error(err, "Error Getting ClusterOwner")
		return ctrl.Result{}, err
	}

	if clusterOwner == nil {
		logger.Info("ClusterOwner not yet set, waiting for CAPI to set it")
		return ctrl.Result{}, nil
	}

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(&cluster, infrav1.ClusterFinalizer) {
		controllerutil.AddFinalizer(&cluster, infrav1.ClusterFinalizer)
		return ctrl.Result{}, nil
	}

	// Handling DeletionTimestamp to decide if it is a Deletion or a Normal reconcile
	if !cluster.DeletionTimestamp.IsZero() {
		return r.ReconcileDelete(ctx, &cluster)
	}

	return r.ReconcileNormal(ctx, &cluster)

}

const (
	secretIdField = ".spec.identitySecret.name"
)

// SetupWithManager sets up the controller with the Manager.
func (r *HarvesterClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &infrav1.HarvesterCluster{}, secretIdField, func(obj client.Object) []string {

		cluster := obj.(*infrav1.HarvesterCluster)

		if (cluster.Spec.IdentitySecret == infrav1.SecretKey{}) || cluster.Spec.IdentitySecret.Name == "" {
			return nil
		}

		clusterSecretName := cluster.Spec.IdentitySecret.Name
		return []string{clusterSecretName}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.HarvesterCluster{}).
		Watches(
			&apiv1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *HarvesterClusterReconciler) findObjectsForSecret(ctx context.Context, secret client.Object) []reconcile.Request {
	attachedClusters := &infrav1.HarvesterClusterList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(secretIdField, secret.GetName()),
	}

	err := r.List(context.TODO(), attachedClusters, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedClusters.Items))

	for i, item := range attachedClusters.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: item.GetNamespace(),
				Name:      item.GetName(),
			},
		}
	}
	return requests
}

// isHarvesterAvailable is a function that parses all conditions for the Available type and Status == True.
// The function return a bool true if the AvailableCondition has a status true, and false in all other cases
func isHarvesterAvailable(conditions []appsv1.DeploymentCondition) bool {
	for _, condition := range conditions {
		if condition.Type == availableConditionType && condition.Status == apiv1.ConditionTrue {
			return true
		}
	}
	return false
}

// ReconcileNormal is the reconciliation function when not deleting the HarvesterCluster instance
func (r *HarvesterClusterReconciler) ReconcileNormal(ctx context.Context, cluster *infrav1.HarvesterCluster) (res ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	var hvRESTConfig *rest.Config

	if hvRESTConfig, err = r.reconcileHarvesterConfig(ctx, cluster); err != nil {
		return ctrl.Result{RequeueAfter: 3 * time.Minute}, err
	}
	// Initializing return values
	res = ctrl.Result{}
	ownedCPHarvesterMachines, err := r.getOwnedCPHarversterMachines(ctx, cluster)
	if err != nil {
		logger.Error(err, "could not get ownerCPMachines")
		return
	}

	if len(ownedCPHarvesterMachines) == 0 {
		logger.Info("no ControlPlane Machines exist yet for cluster, skipping Load Balancer creation and Requeing ...")
		res = ctrl.Result{RequeueAfter: 1 * time.Minute}
		cluster.Spec.ControlPlaneEndpoint = v1beta1.APIEndpoint{
			Host: "127.0.0.1",
			Port: 6443,
		}
		cluster.Status = infrav1.HarvesterClusterStatus{
			Ready: true,
		}
		return
	}

	lbIP, err := createLoadBalancerIfNotExists(cluster, hvRESTConfig, ownedCPHarvesterMachines)
	if err != nil {
		logger.Error(err, "could not create the LoadBalancer")
		return
	}

	cluster.Spec.ControlPlaneEndpoint = v1beta1.APIEndpoint{
		Host: lbIP,
		Port: apiServerLBPort,
	}

	statusReady := infrav1.HarvesterClusterStatus{
		Ready: true,
	}

	if cluster.Status != statusReady {
		cluster.Status = statusReady
	}

	return
}

func (r *HarvesterClusterReconciler) reconcileHarvesterConfig(ctx context.Context, cluster *infrav1.HarvesterCluster) (*rest.Config, error) {
	logger := log.FromContext(ctx)

	secret, err := locutil.GetSecretForHarvesterConfig(ctx, cluster, r.Client)
	if (err != nil || secret == &apiv1.Secret{}) {

		cluster.Status = infrav1.HarvesterClusterStatus{
			FailureReason:  "IdentitySecretUnavailable",
			FailureMessage: "unable to find the IdentitySecret for Harvester",
			Ready:          false,
		}

		if err := r.Status().Update(ctx, cluster); err != nil {
			return &rest.Config{}, errors.Wrapf(err, "failed to update status")
		}

		return &rest.Config{}, errors.Wrapf(err, "unable to find the IdentitySecret for Harvester %s", ctx)
	}

	kubeconfig := secret.Data[locutil.ConfigSecretDataKey]

	var config *clientcmdapi.Config
	if config, err = clientcmd.Load(kubeconfig); err != nil {
		cluster.Status = infrav1.HarvesterClusterStatus{
			FailureReason:  "MalformedIdentitySecret",
			FailureMessage: "unable to Load a valid Harvester config from the referenced Secret",
			Ready:          false,
		}

		if err := r.Status().Update(ctx, cluster); err != nil {
			return &rest.Config{}, errors.Wrapf(err, "failed to update status")
		}

		return &rest.Config{}, errors.Wrapf(err, "unable to Load a valid Harvester config from the referenced Secret %s", ctx)
	}

	logger.Info("Setting Server value from Kubeconfig if needed..")
	configCluster := config.Clusters[config.CurrentContext]

	configServer := configCluster.Server

	if cluster.Spec.Server == "" || cluster.Spec.Server != configServer {
		cluster.Spec.Server = configServer
	}
	logger.Info("Value for Server is now set to " + cluster.Spec.Server)

	hvRESTConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		logger.Error(err, "unable to create kubernetes client config for Harvester")
		return &rest.Config{}, err
	}
	hvClient, err := kubeclient.NewForConfig(hvRESTConfig)
	if err != nil {
		logger.Error(err, "unable to create kubernetes client from restConfig")
		return &rest.Config{}, err
	}

	harvesterDeployment, err := hvClient.AppsV1().Deployments(harvesterNamespace).Get(ctx, harvesterDeploymentName, v1.GetOptions{})

	if err != nil {
		logger.Error(err, "Harvester deployment not found on target Kubernetes cluster")
		return &rest.Config{}, err
	}

	if !isHarvesterAvailable(harvesterDeployment.Status.Conditions) {
		err = fmt.Errorf("harvester cluster unavailable")
		logger.Error(err, "harvester cluster is unavailable")
		return &rest.Config{}, err
	}

	return hvRESTConfig, nil
}

func createLoadBalancerIfNotExists(cluster *infrav1.HarvesterCluster, hvRESTConfig *rest.Config, ownedCPMachines []infrav1.HarvesterMachine) (string, error) {

	hvClientCustom, err := lbclient.NewForConfig(hvRESTConfig)
	if err != nil {
		return "", err
	}

	targetVMNamespace := ownedCPMachines[0].Spec.TargetNamespace

	lbToCreate := &lbv1beta1.LoadBalancer{
		ObjectMeta: v1.ObjectMeta{
			Name:      cluster.Namespace + "-" + cluster.Name + "-lb",
			Namespace: targetVMNamespace,
		},
		Spec: lbv1beta1.LoadBalancerSpec{
			Description:  "Load Balancer for cluster" + cluster.Name,
			WorkloadType: "vm",
			IPAM:         lbv1beta1.IPAM(cluster.Spec.LoadBalancerConfig.IPAMType),
			Listeners: []lbv1beta1.Listener{
				{
					Name:        apiServerListener,
					Port:        apiServerLBPort,
					Protocol:    apiServerProtocol,
					BackendPort: apiServerBackendPort,
				},
			},
			HealthCheck: &lbv1beta1.HealthCheck{
				Port:             apiServerBackendPort,
				SuccessThreshold: 1,
				FailureThreshold: 3,
				PeriodSeconds:    30,
				TimeoutSeconds:   60,
			},
			BackendServerSelector: map[string][]string{
				cpVMLabelKey: {cpVMLabelValuePrefix + "-" + cluster.Name},
			},
		},
	}

	// TODO: replace constante with actual value
	const machineNetwork = "default/untagged"

	if cluster.Spec.LoadBalancerConfig.IPAMType == infrav1.POOL {
		if cluster.Spec.LoadBalancerConfig.IpPoolRef != "" {
			lbToCreate.Spec.IPPool = cluster.Spec.LoadBalancerConfig.IpPoolRef
		}

		if cluster.Spec.LoadBalancerConfig.IpPool != (infrav1.IpPool{}) {
			createdIPPool, err := createIPPool(
				cluster,
				hvClientCustom,
				machineNetwork,
				targetVMNamespace)
			if err != nil {
				return "", err
			}
			lbToCreate.Spec.IPPool = createdIPPool
		}
	}

	// Harvester Call to Harvester
	createdLB, err := hvClientCustom.LoadbalancerV1beta1().LoadBalancers(cluster.Namespace).Create(context.TODO(), lbToCreate, v1.CreateOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "error during creation of LB")
	}

	if createdLB.Status.Address == "" {
		return "", fmt.Errorf("the LB Address was empty after its creation")
	}

	return createdLB.Status.Address, nil
}

func createIPPool(cluster *infrav1.HarvesterCluster, lbClient *lbclient.Clientset, machineNetwork string, targetVMNamespace string) (string, error) {
	ipPoolToCreate := lbv1beta1.IPPool{
		ObjectMeta: v1.ObjectMeta{
			Name:      cluster.Namespace + "-" + cluster.Name + "-ip-pool",
			Namespace: targetVMNamespace,
		},
		Spec: lbv1beta1.IPPoolSpec{
			Description: cpIPPoolDescriptionPrefix + " " + cluster.Name,
			Ranges: []lbv1beta1.Range{
				{
					Subnet:  cluster.Spec.LoadBalancerConfig.IpPool.Subnet,
					Gateway: cluster.Spec.LoadBalancerConfig.IpPool.Gateway,
				},
			},
			Selector: lbv1beta1.Selector{
				Network: machineNetwork,
			},
		},
	}

	createdIPPool, err := lbClient.LoadbalancerV1beta1().IPPools().Create(context.TODO(), &ipPoolToCreate, v1.CreateOptions{})
	if err != nil {
		return "", err
	}

	if createdIPPool.Name == "" {
		return "", fmt.Errorf("IP Pool for HarvesterCluster %s could not be correctly created", cluster.Name)
	}

	return createdIPPool.Name, nil
}

// TODO: Implement
func (r *HarvesterClusterReconciler) getOwnedCPHarversterMachines(ctx context.Context, cluster *infrav1.HarvesterCluster) ([]infrav1.HarvesterMachine, error) {
	return []infrav1.HarvesterMachine{}, nil
}

// ReconcileDelete is the part of the Reconcialiation that deletes a HarvesterCluster and everything which depends on it
func (r *HarvesterClusterReconciler) ReconcileDelete(ctx context.Context, cluster *infrav1.HarvesterCluster) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Deleting Harvester Cluster ...", "cluster-name", cluster.Name, "cluster-namespace", cluster.Namespace)

	ownedCPHarvesterMachines, err := r.getOwnedCPHarversterMachines(ctx, cluster)
	if err != nil {
		return ctrl.Result{RequeueAfter: 10 * time.Minute}, errors.Wrapf(err, "unable to determine target namespace of load balancers and IP Pools because no ControlPlane machines were found")
	}

	if len(ownedCPHarvesterMachines) != 0 {
		var hvRESTConfig *rest.Config

		if hvRESTConfig, err = r.reconcileHarvesterConfig(ctx, cluster); err != nil {
			return ctrl.Result{RequeueAfter: 3 * time.Minute}, err
		}
		lbClient, err := lbclient.NewForConfig(hvRESTConfig)
		if err != nil {
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, err
		}
		targetVMNamespace := ownedCPHarvesterMachines[0].Spec.TargetNamespace

		err = lbClient.LoadbalancerV1beta1().LoadBalancers(targetVMNamespace).Delete(context.TODO(), cluster.Namespace+"-"+cluster.Name+"-lb", v1.DeleteOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "unable to delete Load Balancer in Harvester")
				return ctrl.Result{RequeueAfter: 3 * time.Minute}, err
			}
			logger.Info("no Load Balancer to be deleted, skipping ...")
		}

		err = lbClient.LoadbalancerV1beta1().IPPools().Delete(context.TODO(), cluster.Namespace+"-"+cluster.Name+"-ip-pool", v1.DeleteOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "unable to delete generated IP Pool in Harvester")
				return ctrl.Result{RequeueAfter: 3 * time.Minute}, err
			}
			logger.Info("no IP Pool to be deleted, skipping ...")
		}
	}

	logger.Info("Removing finalizer from HarvesterCluster ...", "cluster-name", cluster.Name, "cluster-namespace", cluster.Namespace)
	controllerutil.RemoveFinalizer(cluster, infrav1.ClusterFinalizer)

	return ctrl.Result{}, nil
}
