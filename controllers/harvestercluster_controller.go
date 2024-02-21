/*
Copyright 2024.

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

	"github.com/go-logr/logr"
	lbv1beta1 "github.com/harvester/harvester-load-balancer/pkg/apis/loadbalancer.harvesterhci.io/v1beta1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"

	infrav1 "github.com/rancher-sandbox/cluster-api-provider-harvester/api/v1alpha1"
	lbclient "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned"
	locutil "github.com/rancher-sandbox/cluster-api-provider-harvester/util"
)

const (
	harvesterNamespace           = "harvester-system"
	harvesterDeploymentName      = "harvester"
	availableConditionType       = "Available"
	lbHealthCheckPeriodSections  = 30
	lbHealthCheckTimeoutSections = 60
	apiServerListener            = "api-server"
	apiServerLBPort              = 6443
	apiServerBackendPort         = 6443
	apiServerProtocol            = "TCP"
	cpIPPoolDescriptionPrefix    = "IP Pool for the control plane's LB of cluster"
	cpVMLabelKey                 = "harvestercluster/machinetype"
	cpVMLabelValuePrefix         = "controlplane"
	requeueTimeThirtySeconds     = 30 * time.Second
	requeueTimeFiveMinutes       = 5 * time.Minute
)

// HarvesterClusterReconciler reconciles a HarvesterCluster object.
type HarvesterClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type ClusterScope struct {
	Cluster          *clusterv1.Cluster
	HarvesterCluster *infrav1.HarvesterCluster
	Logger           logr.Logger
	Ctx              context.Context
	HarvesterClient  *lbclient.Clientset
	ReconcileClient  client.Client
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

		logger.Error(err, "Error happened when getting harvestercluster",
			"cluster-name", req.NamespacedName.Name,
			"cluster-namespace", req.NamespacedName.Namespace)

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

	var hvRESTConfig *rest.Config

	if hvRESTConfig, err = r.reconcileHarvesterConfig(ctx, &cluster); err != nil {
		return ctrl.Result{RequeueAfter: 3 * time.Minute}, err
	}

	// get a Harvester Client
	hvClient, err := lbclient.NewForConfig(hvRESTConfig)
	if err != nil {
		logger.Error(err, "unable to create kubernetes client from restConfig")

		return ctrl.Result{RequeueAfter: 3 * time.Minute}, err
	}

	scope := ClusterScope{
		Cluster:          clusterOwner,
		HarvesterCluster: &cluster,
		Logger:           logger,
		Ctx:              ctx,
		HarvesterClient:  hvClient,
		ReconcileClient:  r.Client,
	}

	// Handling DeletionTimestamp to decide if it is a Deletion or a Normal reconcile
	if !cluster.DeletionTimestamp.IsZero() {
		return r.ReconcileDelete(scope) //nolint:contextcheck
	}

	return r.ReconcileNormal(scope) //nolint:contextcheck
}

const (
	secretIdField = ".spec.identitySecret.name"
)

// SetupWithManager sets up the controller with the Manager.
func (r *HarvesterClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &infrav1.HarvesterCluster{}, secretIdField, func(obj client.Object) []string {
		cluster, ok := obj.(*infrav1.HarvesterCluster)
		if !ok {
			return nil
		}

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

	err := r.List(ctx, attachedClusters, listOps)
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
// The function return a bool true if the AvailableCondition has a status true, and false in all other cases.
func isHarvesterAvailable(conditions []appsv1.DeploymentCondition) bool {
	for _, condition := range conditions {
		if condition.Type == availableConditionType && condition.Status == apiv1.ConditionTrue {
			return true
		}
	}

	return false
}

// ReconcileNormal is the reconciliation function when not deleting the HarvesterCluster instance.
func (r *HarvesterClusterReconciler) ReconcileNormal(scope ClusterScope) (res ctrl.Result, err error) {
	logger := log.FromContext(scope.Ctx)

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(scope.HarvesterCluster, infrav1.ClusterFinalizer) {
		controllerutil.AddFinalizer(scope.HarvesterCluster, infrav1.ClusterFinalizer)

		return ctrl.Result{}, nil
	}

	// Check if TargetNamespace exists, if not create it
	_, err = scope.HarvesterClient.CoreV1().Namespaces().Get(context.TODO(), scope.HarvesterCluster.Spec.TargetNamespace, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = scope.HarvesterClient.CoreV1().Namespaces().Create(context.TODO(), &apiv1.Namespace{
				ObjectMeta: v1.ObjectMeta{
					Name: scope.HarvesterCluster.Spec.TargetNamespace,
				},
			}, v1.CreateOptions{})
			if err != nil {
				logger.Error(err, "unable to create TargetNamespace")
			}
		} else {
			logger.Error(err, "unable to get TargetNamespace in Harvester, problem with the HarvesterClient")
		}
	}

	// Initializing return values
	res = ctrl.Result{}

	ownedCPHarvesterMachines, err := r.getOwnedCPHarversterMachines(scope)
	if err != nil {
		logger.Error(err, "could not get ownerCPMachines")

		return res, err
	}

	if len(ownedCPHarvesterMachines) == 0 {
		logger.Info("no ControlPlane Machines exist yet for cluster, skipping Load Balancer creation and Requeing ...")

		// Create a placeholder LoadBalancer svc to avoid blocking the CAPI Controller
		existingPlaceholderLB, err1 := scope.HarvesterClient.CoreV1().Services(scope.HarvesterCluster.Spec.TargetNamespace).Get(
			scope.Ctx,
			scope.HarvesterCluster.Namespace+"-"+scope.HarvesterCluster.Name+"-lb",
			v1.GetOptions{})

		if err1 != nil {
			if !apierrors.IsNotFound(err1) {
				logger.Error(err1, "could not get the placeholder LoadBalancer")

				return ctrl.Result{RequeueAfter: requeueTimeFiveMinutes}, err1
			} else {
				placeholderSVC := &apiv1.Service{
					ObjectMeta: v1.ObjectMeta{
						Name:      scope.HarvesterCluster.Namespace + "-" + scope.HarvesterCluster.Name + "-lb",
						Namespace: scope.HarvesterCluster.Spec.TargetNamespace,
						Labels: map[string]string{
							"loadbalancer.harvesterhci.io/servicelb": "true",
						},
					},
					Spec: apiv1.ServiceSpec{
						AllocateLoadBalancerNodePorts: func() *bool { b := true; return &b }(), //nolint:nlreturn
						Ports: []apiv1.ServicePort{
							{
								Name:       apiServerListener,
								Port:       apiServerLBPort,
								Protocol:   apiServerProtocol,
								TargetPort: intstr.FromInt(apiServerBackendPort),
							},
						},
						Type: apiv1.ServiceTypeLoadBalancer,
						IPFamilies: []apiv1.IPFamily{
							apiv1.IPv4Protocol,
						},
						IPFamilyPolicy: func() *apiv1.IPFamilyPolicyType {
							policy := apiv1.IPFamilyPolicySingleStack

							return &policy
						}(),
						LoadBalancerIP: "0.0.0.0",
					},
				}

				_, err = scope.HarvesterClient.CoreV1().Services(scope.HarvesterCluster.Spec.TargetNamespace).Create(
					scope.Ctx,
					placeholderSVC,
					v1.CreateOptions{})
				if err != nil {
					if !apierrors.IsAlreadyExists(err) {
						logger.Error(err, "could not create the placeholder LoadBalancer")

						return ctrl.Result{RequeueAfter: requeueTimeFiveMinutes}, err
					} else {
						logger.Info("placeholder LoadBalancer already exists, skipping ...")
					}
				}
				res = ctrl.Result{RequeueAfter: requeueTimeThirtySeconds}

				return res, err
			}
		}

		// Requeue if the placeholder LoadBalancer IP is empty
		if len(existingPlaceholderLB.Status.LoadBalancer.Ingress) == 0 || existingPlaceholderLB.Status.LoadBalancer.Ingress[0].IP == "" {
			logger.Info("placeholder LoadBalancer IP is empty, waiting for IP to be set ...")

			return ctrl.Result{RequeueAfter: requeueTimeThirtySeconds}, nil
		}

		// res = ctrl.Result{RequeueAfter: 5 * time.Minute}
		scope.HarvesterCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
			Host: existingPlaceholderLB.Status.LoadBalancer.Ingress[0].IP,
			Port: apiServerLBPort,
		}
		scope.HarvesterCluster.Status = infrav1.HarvesterClusterStatus{
			Ready: true,
		}

		return res, err
	}

	// The following is executed only if there are ownedCPHarvesterMachines
	if !conditions.IsTrue(scope.HarvesterCluster, infrav1.LoadBalancerReadyCondition) {
		err := createLoadBalancerIfNotExists(scope, ownedCPHarvesterMachines)
		if err != nil {
			logger.V(1).Info("could not create the LoadBalancer, requeuing ...")

			return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
		}

		lbIP, err := getLoadBalancerIP(scope.Cluster, scope.HarvesterCluster, scope.HarvesterClient)
		if err != nil {
			logger.Error(err, "could not get the LoadBalancer IP")

			return ctrl.Result{RequeueAfter: requeueTimeThirtySeconds}, err
		}

		scope.HarvesterCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
			Host: lbIP,
			Port: apiServerLBPort,
		}

		scope.HarvesterCluster.Status.Ready = true
		scope.HarvesterCluster.Status.Conditions = append(scope.HarvesterCluster.Status.Conditions, clusterv1.Condition{
			Type:   infrav1.LoadBalancerReadyCondition,
			Status: apiv1.ConditionTrue,
		})

		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	return res, err
}

func getLoadBalancerIP(cluster *clusterv1.Cluster, harvesterCluster *infrav1.HarvesterCluster, hvClient *lbclient.Clientset) (string, error) {
	createdLB, err := hvClient.LoadbalancerV1beta1().LoadBalancers(harvesterCluster.Spec.TargetNamespace).Get(
		context.TODO(),
		harvesterCluster.Namespace+"-"+harvesterCluster.Name+"-lb",
		v1.GetOptions{})
	if err != nil {
		return "", err
	}

	if createdLB.Status.Address == "" {
		return "", fmt.Errorf("the LB Address was empty after its creation")
	}

	return createdLB.Status.Address, nil
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

	configCluster := config.Clusters[config.CurrentContext]

	configServer := configCluster.Server

	if cluster.Spec.Server == "" || cluster.Spec.Server != configServer {
		cluster.Spec.Server = configServer
		logger.Info("Value for Server is now set to " + cluster.Spec.Server)
	}

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
		logger.Error(err, "harvester cluster is unavailable")

		return &rest.Config{}, err
	}

	return hvRESTConfig, nil
}

func createLoadBalancerIfNotExists(scope ClusterScope, ownedCPMachines []infrav1.HarvesterMachine) (err error) {
	additionalListeners := getListenersFromAPI(scope.HarvesterCluster)

	lbToCreate := &lbv1beta1.LoadBalancer{
		ObjectMeta: v1.ObjectMeta{
			Name:      scope.HarvesterCluster.Namespace + "-" + scope.HarvesterCluster.Name + "-lb",
			Namespace: scope.HarvesterCluster.Spec.TargetNamespace,
		},
		Spec: lbv1beta1.LoadBalancerSpec{
			Description:  "Load Balancer for cluster " + scope.HarvesterCluster.Name,
			WorkloadType: "vm",
			IPAM:         lbv1beta1.IPAM(scope.HarvesterCluster.Spec.LoadBalancerConfig.IPAMType),
			Listeners: append(additionalListeners, lbv1beta1.Listener{
				Name:        apiServerListener,
				Port:        apiServerLBPort,
				Protocol:    apiServerProtocol,
				BackendPort: apiServerBackendPort,
			}),
			HealthCheck: &lbv1beta1.HealthCheck{
				Port:             apiServerBackendPort,
				SuccessThreshold: 1,
				FailureThreshold: 3,
				PeriodSeconds:    lbHealthCheckPeriodSections,
				TimeoutSeconds:   lbHealthCheckTimeoutSections,
			},
			BackendServerSelector: map[string][]string{
				cpVMLabelKey: {cpVMLabelValuePrefix + "-" + scope.Cluster.Name},
			},
		},
	}

	machineNetwork := types.NamespacedName{}
	if scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPool.VMNetwork != "" {
		err, machineNetwork = locutil.GetNamespacedName(
			scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPool.VMNetwork,
			scope.HarvesterCluster.Spec.TargetNamespace)
		if err != nil {
			return errors.Wrap(err, "VMNetwork reference is not valid")
		}
	}

	if scope.HarvesterCluster.Spec.LoadBalancerConfig.IPAMType == infrav1.POOL {
		if scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPoolRef != "" {
			// IPPools are not namespaced, thus we don't need to add the namespace to the name
			lbToCreate.Spec.IPPool = scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPoolRef
		} else {
			if scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPool != (infrav1.IpPool{}) {
				createdIPPool, err := createIPPool(
					scope.HarvesterCluster,
					scope.HarvesterClient,
					machineNetwork.Namespace+"/"+machineNetwork.Name,
					scope.HarvesterCluster.Spec.TargetNamespace)
				if err != nil {
					return err
				}

				lbToCreate.Spec.IPPool = createdIPPool
			} else {
				return fmt.Errorf("IP Pool is not defined")
			}
		}
	}

	// Harvester Call to Harvester
	_, err = scope.HarvesterClient.LoadbalancerV1beta1().LoadBalancers(scope.HarvesterCluster.Spec.TargetNamespace).Create(
		context.TODO(),
		lbToCreate,
		v1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrapf(err, "error during creation of LB")
		}
	}

	return nil
}

// getListenersFromAPI is a function that gets the listeners from the HarvesterCluster Resource and returns them as a slice of lbv1beta1.Listener.
func getListenersFromAPI(cluster *infrav1.HarvesterCluster) []lbv1beta1.Listener {
	additionalListeners := make([]lbv1beta1.Listener, len(cluster.Spec.LoadBalancerConfig.Listeners))
	for i, listener := range cluster.Spec.LoadBalancerConfig.Listeners {
		additionalListeners[i] = lbv1beta1.Listener{
			Name:        listener.Name,
			Port:        listener.Port,
			Protocol:    listener.Protocol,
			BackendPort: listener.BackendPort,
		}
	}

	return additionalListeners
}

// createIPPool is a function that creates an IP Pool in Harvester.
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

// getOwnedCPHarversterMachines is a function that gets the HarvesterMachines that are owned by the HarvesterCluster and are controlplane machines.
func (r *HarvesterClusterReconciler) getOwnedCPHarversterMachines(scope ClusterScope) ([]infrav1.HarvesterMachine, error) {
	// Get all the harvestermachines for the cluster.
	// Filter the harvestermachines that are controlplane machines.
	// Filter the harvestermachines that are owned by the cluster.
	// Return the harvestermachines that are owned by the cluster and are controlplane machines.
	ownedCPHarvesterMachines := &infrav1.HarvesterMachineList{}

	err := r.Client.List(scope.Ctx,
		ownedCPHarvesterMachines,
		client.MatchingLabels{clusterv1.ClusterNameLabel: scope.Cluster.Name},
		client.HasLabels{clusterv1.MachineControlPlaneLabel})
	if err != nil {
		if apierrors.IsNotFound(err) {
			scope.Logger.Info("no ControlPlane Machines exist yet for cluster")

			return []infrav1.HarvesterMachine{}, nil
		}

		return []infrav1.HarvesterMachine{}, errors.Wrap(err, "unable to list owned ControlPlane Machines")
	}

	return ownedCPHarvesterMachines.Items, nil
}

// ReconcileDelete is the part of the Reconcialiation that deletes a HarvesterCluster and everything which depends on it.
func (r *HarvesterClusterReconciler) ReconcileDelete(scope ClusterScope) (ctrl.Result, error) {
	logger := log.FromContext(scope.Ctx)
	logger.Info("Deleting Harvester Cluster ...", "cluster-name", scope.HarvesterCluster.Name, "cluster-namespace", scope.HarvesterCluster.Namespace)

	err := scope.HarvesterClient.LoadbalancerV1beta1().LoadBalancers(scope.HarvesterCluster.Spec.TargetNamespace).Delete(
		context.TODO(),
		scope.HarvesterCluster.Namespace+"-"+scope.HarvesterCluster.Name+"-lb",
		v1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to delete Load Balancer in Harvester")

			return ctrl.Result{RequeueAfter: 3 * time.Minute}, err
		}

		logger.Info("no Load Balancer to be deleted, skipping ...")
	}

	logger.V(5).Info("Load Balancer deleted successfully")

	err = scope.HarvesterClient.CoreV1().Services(scope.HarvesterCluster.Spec.TargetNamespace).Delete(
		context.TODO(),
		scope.HarvesterCluster.Namespace+"-"+scope.HarvesterCluster.Name+"-lb",
		v1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to delete Load Balancer Service in Harvester")

			return ctrl.Result{RequeueAfter: 3 * time.Minute}, err
		}

		logger.Info("no Load Balancer Service to be deleted, skipping ...")
	}

	logger.V(5).Info("Load Balancer Service deleted successfully")

	err = scope.HarvesterClient.LoadbalancerV1beta1().IPPools().Delete(
		context.TODO(),
		scope.HarvesterCluster.Namespace+"-"+scope.HarvesterCluster.Name+"-ip-pool",
		v1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to delete generated IP Pool in Harvester")

			return ctrl.Result{RequeueAfter: 3 * time.Minute}, err
		}

		logger.Info("no IP Pool to be deleted, skipping ...")
	}

	logger.V(5).Info("IP Pool deleted successfully")
	logger.Info("Removing finalizer from HarvesterCluster ...",
		"cluster-name", scope.HarvesterCluster.Name,
		"cluster-namespace", scope.HarvesterCluster.Namespace)
	controllerutil.RemoveFinalizer(scope.HarvesterCluster, infrav1.ClusterFinalizer)

	return ctrl.Result{}, nil
}
