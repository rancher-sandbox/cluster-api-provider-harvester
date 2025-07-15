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

// Package controller contains the HarvesterCluster controller logic.
package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"time"

	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend/allocator"
	"github.com/go-logr/logr"
	lbv1beta1 "github.com/harvester/harvester-load-balancer/pkg/apis/loadbalancer.harvesterhci.io/v1beta1"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
	requeueTimeShort             = 30 * time.Second
	requeueTimeMedium            = 5 * time.Minute
	requeueTimeLong              = 3 * time.Minute
	dhcpLbIP                     = "0.0.0.0"
	failureThreshold             = 3
	cloudProviderTargetNamespace = "kube-system"
)

// HarvesterClusterReconciler reconciles a HarvesterCluster object.
type HarvesterClusterReconciler struct {
	client.Client

	Scheme *runtime.Scheme
}

// ClusterScope is a struct that contains the necessary data needed for a HarvesterCluster controller.
type ClusterScope struct {
	Cluster          *clusterv1.Cluster
	HarvesterCluster *infrav1.HarvesterCluster
	Logger           logr.Logger
	Ctx              context.Context
	HarvesterClient  lbclient.Interface
	ReconcileClient  client.Client
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status;machinesets;machines;machines/status;machinepools;machinepools/status,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;update;patch;delete

// Reconcile reads that state of the cluster for a HarvesterCluster object and makes changes based on the state read.
func (r *HarvesterClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling HarvesterCluster", "cluster-name", req.Name, "cluster-namespace", req.Namespace)

	var cluster infrav1.HarvesterCluster

	err := r.Get(ctx, req.NamespacedName, &cluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("cluster not found", "cluster-name", req.Name, "cluster-namespace", req.Namespace)

			return ctrl.Result{}, nil
		}

		logger.Error(err, "Error happened when getting harvestercluster",
			"cluster-name", req.Name,
			"cluster-namespace", req.Namespace)

		return ctrl.Result{}, err
	}

	patchHelper, err := patch.NewHelper(&cluster, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		patchErr := patchHelper.Patch(ctx, &cluster)
		if patchErr != nil {
			clusterString := cluster.Namespace + "/" + cluster.Name
			logger.Error(patchErr, "unable to patch", "cluster", clusterString)
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

	hvRESTConfig, err = r.reconcileHarvesterConfig(ctx, &cluster)
	if err != nil {
		return ctrl.Result{RequeueAfter: requeueTimeLong}, err
	}

	// get a Harvester Client
	hvClient, err := lbclient.NewForConfig(hvRESTConfig)
	if err != nil {
		logger.Error(err, "unable to create kubernetes client from restConfig")

		return ctrl.Result{RequeueAfter: requeueTimeLong}, err
	}

	scope := &ClusterScope{
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
	secretIdField = ".spec.identitySecret.name" //nolint:gosec
)

// SetupWithManager sets up the controller with the Manager.
func (r *HarvesterClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(ctx, &infrav1.HarvesterCluster{}, secretIdField, func(obj client.Object) []string {
		cluster, ok := obj.(*infrav1.HarvesterCluster)
		if !ok {
			return nil
		}

		if (cluster.Spec.IdentitySecret == infrav1.SecretKey{}) || cluster.Spec.IdentitySecret.Name == "" {
			return nil
		}

		clusterSecretName := cluster.Spec.IdentitySecret.Name

		return []string{clusterSecretName}
	})
	if err != nil {
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

// ReconcileNormal is the reconciliation function when not deleting the HarvesterCluster instance.
func (r *HarvesterClusterReconciler) ReconcileNormal(scope *ClusterScope) (res ctrl.Result, err error) {
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
		// Give the LB a name that is RFC-1035 compliant
		lbName := locutil.GenerateRFC1035Name([]string{scope.HarvesterCluster.Namespace, scope.HarvesterCluster.Name, "lb"})
		lbNamespacedName := scope.HarvesterCluster.Spec.TargetNamespace + "/" + lbName
		// Create a placeholder LoadBalancer svc to avoid blocking the CAPI Controller
		existingPlaceholderLB, err1 := scope.HarvesterClient.CoreV1().Services(scope.HarvesterCluster.Spec.TargetNamespace).Get(
			scope.Ctx,
			lbName,
			v1.GetOptions{})
		if err1 != nil {
			if !apierrors.IsNotFound(err1) {
				logger.Error(err1, "could not get the placeholder LoadBalancer")

				return ctrl.Result{RequeueAfter: requeueTimeMedium}, err1
			}

			lbIP := dhcpLbIP

			if scope.HarvesterCluster.Spec.LoadBalancerConfig.IPAMType == infrav1.POOL {
				lbIP, err = getIPFromIPPool(scope, lbNamespacedName)
				if err != nil {
					logger.Error(err, "could not get IP from IP Pool")

					return ctrl.Result{RequeueAfter: requeueTimeShort}, err
				}
			}

			err = createPlaceholderSVC(lbName, scope, lbIP)
			if err != nil {
				if !apierrors.IsAlreadyExists(err) {
					scope.Logger.Error(err, "could not create the placeholder LoadBalancer")

					return ctrl.Result{Requeue: true}, err
				}

				scope.Logger.Info("placeholder LoadBalancer already exists, skipping ...")
			}

			scope.Logger.Info("placeholder LoadBalancer created successfully", "name", lbName, "namespace", scope.HarvesterCluster.Spec.TargetNamespace)

			res = ctrl.Result{RequeueAfter: requeueTimeShort}

			return res, err
		}

		// Requeue if the placeholder LoadBalancer IP is empty
		if len(existingPlaceholderLB.Status.LoadBalancer.Ingress) == 0 || existingPlaceholderLB.Status.LoadBalancer.Ingress[0].IP == "" {
			logger.Info("placeholder LoadBalancer IP is empty, waiting for IP to be set ...")

			if scope.HarvesterCluster.Spec.LoadBalancerConfig.IPAMType == infrav1.POOL {
				newLbIP, err := getIPFromIPPool(scope, lbNamespacedName)
				if err != nil {
					return ctrl.Result{RequeueAfter: requeueTimeShort}, err
				}

				existingPlaceholderLB.Spec.LoadBalancerIP = newLbIP

				_, err = scope.HarvesterClient.CoreV1().Services(scope.HarvesterCluster.Spec.TargetNamespace).Update(context.TODO(),
					existingPlaceholderLB, v1.UpdateOptions{})
				if err != nil {
					err = errors.Wrap(err, "could not update the placeholder LoadBalancer")

					return ctrl.Result{RequeueAfter: requeueTimeShort}, err
				}

				logger.Info("placeholder LoadBalancer IP updated successfully using IP Pools", "IP", newLbIP)

				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{RequeueAfter: requeueTimeShort}, nil
		}

		// res = ctrl.Result{RequeueAfter: 5 * time.Minute}
		scope.HarvesterCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
			Host: existingPlaceholderLB.Status.LoadBalancer.Ingress[0].IP,
			Port: apiServerLBPort,
		}
		scope.HarvesterCluster.Status.Ready = true

		res = ctrl.Result{RequeueAfter: 1 * time.Minute}

		return res, err
	}

	// Reconcile Cloud Provider Config
	err = r.reconcileCloudProviderConfig(scope)
	if err != nil {
		return ctrl.Result{RequeueAfter: requeueTimeLong}, err
	}

	// Set InfrastructureReady condition to in progress
	conditions.Set(scope.HarvesterCluster, &clusterv1.Condition{
		Type:    infrav1.InfrastructureReadyCondition,
		Status:  apiv1.ConditionFalse,
		Reason:  infrav1.InfrastructureProvisioningInProgressReason,
		Message: "Infrastructure provisioning in progress",
	})

	// The following is executed only if there are ownedCPHarvesterMachines
	if !conditions.IsTrue(scope.HarvesterCluster, infrav1.LoadBalancerReadyCondition) {
		err := createLoadBalancerIfNotExists(scope)
		if err != nil {
			logger.V(1).Info("could not create the LoadBalancer, requeuing ...")

			conditions.Set(scope.HarvesterCluster, &clusterv1.Condition{
				Type:    infrav1.InfrastructureReadyCondition,
				Status:  apiv1.ConditionFalse,
				Reason:  infrav1.InfrastructureProvisioningFailedReason,
				Message: fmt.Sprintf("Failed to create LoadBalancer: %v", err),
			})

			return ctrl.Result{RequeueAfter: 1 * time.Minute}, err //nolint:nlreturn
		}

		lbIP, err := getLoadBalancerIP(scope.HarvesterCluster, scope.HarvesterClient)
		if err != nil {
			logger.Info("LoadBalancer IP is not yet available, requeuing ...")

			conditions.Set(scope.HarvesterCluster, &clusterv1.Condition{
				Type:    infrav1.InfrastructureReadyCondition,
				Status:  apiv1.ConditionFalse,
				Reason:  infrav1.InfrastructureProvisioningInProgressReason,
				Message: "Waiting for LoadBalancer IP to be available",
			})

			return ctrl.Result{RequeueAfter: requeueTimeShort}, err //nolint:nlreturn
		}

		scope.HarvesterCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
			Host: lbIP,
			Port: apiServerLBPort,
		}

		scope.HarvesterCluster.Status.Ready = true

		// Set LoadBalancerReady condition
		conditions.Set(scope.HarvesterCluster, &clusterv1.Condition{
			Type:    infrav1.LoadBalancerReadyCondition,
			Status:  apiv1.ConditionTrue,
			Reason:  "LoadBalancerReady",
			Message: "LoadBalancer is ready with assigned IP",
		})

		// Set InfrastructureReady condition
		conditions.Set(scope.HarvesterCluster, &clusterv1.Condition{
			Type:    infrav1.InfrastructureReadyCondition,
			Status:  apiv1.ConditionTrue,
			Reason:  infrav1.InfrastructureReadyReason,
			Message: "All infrastructure components are ready",
		})

		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// If LoadBalancer is already ready, set InfrastructureReady as well
	if conditions.IsTrue(scope.HarvesterCluster, infrav1.LoadBalancerReadyCondition) {
		conditions.Set(scope.HarvesterCluster, &clusterv1.Condition{
			Type:    infrav1.InfrastructureReadyCondition,
			Status:  apiv1.ConditionTrue,
			Reason:  infrav1.InfrastructureReadyReason,
			Message: "All infrastructure components are ready",
		})
	}

	return res, err
}

func createPlaceholderSVC(lbName string, scope *ClusterScope, lbIP string) error {
	placeholderSVC := &apiv1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      lbName,
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
			LoadBalancerIP: lbIP,
		},
	}

	_, err := scope.HarvesterClient.CoreV1().Services(scope.HarvesterCluster.Spec.TargetNamespace).Create(
		scope.Ctx,
		placeholderSVC,
		v1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func checkValidIpPoolDefinition(ipPool infrav1.IpPool) bool {
	if ipPool == (infrav1.IpPool{}) {
		return false
	}

	if ipPool.Subnet == "" || ipPool.Gateway == "" || ipPool.VMNetwork == "" {
		return false
	}

	return true
}

func getIPFromIPPool(scope *ClusterScope, lbNamespacedName string) (string, error) {
	poolRef := scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPoolRef

	if poolRef == "" && !checkValidIpPoolDefinition(scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPool) {
		return "", errors.Errorf("IP Pool reference is empty, while IPAMType is set to %s", infrav1.POOL)
	}

	ipPool := &lbv1beta1.IPPool{}

	var err error

	if poolRef == "" && checkValidIpPoolDefinition(scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPool) {
		ipPool, err = createIPPoolIfNotExists(
			scope.HarvesterCluster,
			scope.HarvesterClient,
			scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPool.VMNetwork,
			scope.HarvesterCluster.Spec.TargetNamespace)
		if err != nil {
			return "", err
		}

		scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPoolRef = ipPool.Name
	}

	if poolRef != "" {
		ipPool, err = scope.HarvesterClient.LoadbalancerV1beta1().IPPools().Get(
			context.TODO(),
			poolRef,
			v1.GetOptions{})
		if err != nil {
			return "", errors.Wrapf(err, "could not get referenced IP Pool %s", poolRef)
		}
	}

	if ipPool.Status.Available == 0 {
		return "", errors.Errorf("IP Pool %s does not have available addresses", poolRef)
	}

	return allocateIPFromPool(ipPool, lbNamespacedName, scope)
}

func allocateIPFromPool(refPool *lbv1beta1.IPPool, lbNamespacedName string, scope *ClusterScope) (string, error) {
	rangeSlice := make([]allocator.Range, 0)

	var total int64

	ranges := refPool.Spec.Ranges

	for i := range ranges {
		element, err := locutil.MakeRange(&ranges[i])
		if err != nil {
			return "", err
		}

		rangeSlice = append(rangeSlice, *element)

		total += locutil.CountIP(element)
	}

	rangeSet := allocator.RangeSet(rangeSlice)

	a := allocator.NewIPAllocator(&rangeSet, locutil.NewStore(refPool), 0)

	var ipObj *current.IPConfig

	var err error

	// apply the IP allocated before in priority
	if refPool.Status.AllocatedHistory != nil {
		for k, v := range refPool.Status.AllocatedHistory {
			if lbNamespacedName == v {
				ipObj, err = a.Get(lbNamespacedName, "", net.ParseIP(k))
				if err != nil {
					return "", err
				}
			}
		}
	}

	if ipObj == nil {
		ipObj, err = a.Get(lbNamespacedName, "", nil)
		if err != nil {
			return "", err
		}
	}

	// Update Pool in Harvester with the new allocated IP
	_, err = scope.HarvesterClient.LoadbalancerV1beta1().IPPools().Update(context.TODO(), refPool, v1.UpdateOptions{})
	if err != nil {
		return "", err
	}

	return ipObj.Address.IP.String(), nil
}

func getLoadBalancerIP(harvesterCluster *infrav1.HarvesterCluster, hvClient lbclient.Interface) (string, error) {
	createdLB, err := hvClient.LoadbalancerV1beta1().LoadBalancers(harvesterCluster.Spec.TargetNamespace).Get(
		context.TODO(),
		locutil.GenerateRFC1035Name([]string{harvesterCluster.Namespace, harvesterCluster.Name, "lb"}),
		v1.GetOptions{})
	if err != nil {
		return "", err
	}

	if createdLB.Status.Address == "" {
		return "", errors.New("the LB Address was empty after its creation")
	}

	return createdLB.Status.Address, nil
}

// ReconcileDelete is the part of the Reconcialiation that deletes a HarvesterCluster and everything which depends on it.
func (r *HarvesterClusterReconciler) ReconcileDelete(scope *ClusterScope) (ctrl.Result, error) {
	logger := log.FromContext(scope.Ctx)
	logger.Info("Deleting Harvester Cluster ...", "cluster-name", scope.HarvesterCluster.Name, "cluster-namespace", scope.HarvesterCluster.Namespace)

	err := scope.HarvesterClient.LoadbalancerV1beta1().LoadBalancers(scope.HarvesterCluster.Spec.TargetNamespace).Delete(
		context.TODO(),
		locutil.GenerateRFC1035Name([]string{scope.HarvesterCluster.Namespace, scope.HarvesterCluster.Name, "lb"}),
		v1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to delete Load Balancer in Harvester")

			return ctrl.Result{RequeueAfter: requeueTimeLong}, err
		}

		logger.Info("no Load Balancer to be deleted, skipping ...")
	}

	logger.V(5).Info("Load Balancer deleted successfully")

	if conditions.IsTrue(scope.HarvesterCluster, infrav1.CustomIPPoolCreatedCondition) {
		err := scope.HarvesterClient.LoadbalancerV1beta1().IPPools().Delete(
			context.TODO(),
			scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPoolRef,
			v1.DeleteOptions{},
		)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "unable to delete IP Pool in Harvester")

				return ctrl.Result{RequeueAfter: requeueTimeLong}, err
			}

			logger.Info("no IP Pool to be deleted, skipping ...")
		}

		logger.Info("Custom IP Pool deleted")
		conditions.Delete(scope.HarvesterCluster, infrav1.CustomIPPoolCreatedCondition)
	}

	err = scope.HarvesterClient.CoreV1().Services(scope.HarvesterCluster.Spec.TargetNamespace).Delete(
		context.TODO(),
		locutil.GenerateRFC1035Name([]string{scope.HarvesterCluster.Namespace, scope.HarvesterCluster.Name, "lb"}),
		v1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to delete Load Balancer Service in Harvester")

			return ctrl.Result{RequeueAfter: requeueTimeLong}, err
		}

		logger.Info("no Load Balancer Service to be deleted, skipping ...")
	}

	logger.V(5).Info("Load Balancer Service deleted successfully") //nolint:mnd

	err = scope.HarvesterClient.LoadbalancerV1beta1().IPPools().Delete(
		context.TODO(),
		locutil.GenerateRFC1035Name([]string{scope.HarvesterCluster.Namespace, scope.HarvesterCluster.Name, "ippool"}),
		v1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to delete generated IP Pool in Harvester")

			return ctrl.Result{RequeueAfter: requeueTimeLong}, err
		}

		logger.Info("no IP Pool to be deleted, skipping ...")
	}

	logger.V(5).Info("IP Pool deleted successfully") //nolint:mnd
	logger.Info("Removing finalizer from HarvesterCluster ...",
		"cluster-name", scope.HarvesterCluster.Name,
		"cluster-namespace", scope.HarvesterCluster.Namespace)
	controllerutil.RemoveFinalizer(scope.HarvesterCluster, infrav1.ClusterFinalizer)

	return ctrl.Result{}, nil
}

func (r *HarvesterClusterReconciler) reconcileCloudProviderConfig(scope *ClusterScope) error {
	// Skip if the Cloud Provider Config is already ready
	if conditions.IsTrue(scope.HarvesterCluster, infrav1.CloudProviderConfigReadyCondition) {
		return nil
	}

	// Check if user provided the necessary information to generate the cloud provider config
	updateCloudConfig := scope.HarvesterCluster.Spec.UpdateCloudProviderConfig
	if (updateCloudConfig != infrav1.UpdateCloudProviderConfig{}) {
		// Get the Cloud Provider Manifest from the referenced ConfigMap
		if updateCloudConfig.ManifestsConfigMapName == "" || updateCloudConfig.ManifestsConfigMapNamespace == "" {
			return errors.New("ManifestsConfigMapName and ManifestsConfigMapNamespace must be set")
		}

		referencedConfigMap := &apiv1.ConfigMap{}

		err := r.Get(context.TODO(), types.NamespacedName{
			Name:      updateCloudConfig.ManifestsConfigMapName,
			Namespace: updateCloudConfig.ManifestsConfigMapNamespace,
		}, referencedConfigMap)
		if err != nil {
			return errors.Wrapf(err, "unable to get the referenced config map %s/%s",
				updateCloudConfig.ManifestsConfigMapNamespace,
				updateCloudConfig.ManifestsConfigMapName)
		}

		cloudConfigManifest, err := locutil.GetDataKeyFromConfigMap(referencedConfigMap,
			updateCloudConfig.ManifestsConfigMapKey)
		if err != nil {
			return errors.Wrapf(err, "unable to get the data key %s from the referenced config map %s/%s",
				updateCloudConfig.ManifestsConfigMapKey,
				updateCloudConfig.ManifestsConfigMapNamespace,
				updateCloudConfig.ManifestsConfigMapName)
		}

		// Generate the B64 Kubeconfig fpr the cloud provider
		cloudProviderKubeconfigB64, err := locutil.GetCloudConfigB64(scope.HarvesterClient,
			scope.Cluster.Name, scope.HarvesterCluster.Spec.TargetNamespace, scope.HarvesterCluster.Spec.Server)
		if err != nil {
			return errors.Wrapf(err, "unable to generate the kubeconfig for the cloud provider")
		}

		cloudProviderKubeconfigBytes, err := base64.StdEncoding.DecodeString(cloudProviderKubeconfigB64)
		if err != nil {
			return errors.Wrapf(err, "unable to decode the kubeconfig for the cloud provider")
		}

		// Modify the cloudConfig Manifest to include the B64 Kubeconfig
		modifiedManifests, err := locutil.ModifyYAMlString(
			cloudConfigManifest,
			updateCloudConfig.CloudConfigCredentialsSecretName,
			cloudProviderTargetNamespace,
			updateCloudConfig.CloudConfigCredentialsSecretKey,
			cloudProviderKubeconfigBytes)
		if err != nil {
			return errors.Wrapf(err, "unable to modify the cloudConfig Manifest to include the B64 Kubeconfig")
		}

		// Update the ConfigMap with the modified cloudConfig Manifest
		referencedConfigMap.Data[updateCloudConfig.ManifestsConfigMapKey] = modifiedManifests

		err = r.Update(context.TODO(), referencedConfigMap)
		if err != nil {
			return errors.Wrapf(err, "unable to update the referenced config map %s/%s",
				updateCloudConfig.ManifestsConfigMapNamespace, updateCloudConfig.ManifestsConfigMapName)
		}
	}

	conditions.Set(scope.HarvesterCluster, &clusterv1.Condition{
		Type:    infrav1.CloudProviderConfigReadyCondition,
		Status:  apiv1.ConditionTrue,
		Reason:  infrav1.CloudProviderConfigGeneratedSuccessfullyReason,
		Message: "Cloud Provider Config was generated successfully",
	})

	return nil
}

func (r *HarvesterClusterReconciler) reconcileHarvesterConfig(ctx context.Context, cluster *infrav1.HarvesterCluster) (*rest.Config, error) {
	logger := log.FromContext(ctx)

	secret, err := locutil.GetSecretForHarvesterConfig(ctx, cluster, r.Client)
	if (err != nil || secret == &apiv1.Secret{}) {
		cluster.Status.FailureReason = "IdentitySecretUnavailable"
		cluster.Status.FailureMessage = "unable to find the IdentitySecret for Harvester"
		cluster.Status.Ready = false

		return &rest.Config{}, errors.Wrapf(err, "unable to find the IdentitySecret for Harvester %s", ctx)
	}

	kubeconfig := secret.Data[locutil.ConfigSecretDataKey]

	harvesterServer, err := getHarvesterServerFromKubeconfig(kubeconfig)
	if err != nil {
		cluster.Status.FailureReason = "MalformedIdentitySecret"
		cluster.Status.FailureMessage = err.Error()
		cluster.Status.Ready = false

		return &rest.Config{}, err
	}

	if cluster.Spec.Server == "" || cluster.Spec.Server != harvesterServer {
		cluster.Spec.Server = harvesterServer
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

func createLoadBalancerIfNotExists(scope *ClusterScope) (err error) {
	additionalListeners := getListenersFromAPI(scope.HarvesterCluster)

	lbToCreate := &lbv1beta1.LoadBalancer{
		ObjectMeta: v1.ObjectMeta{
			Name:      locutil.GenerateRFC1035Name([]string{scope.HarvesterCluster.Namespace, scope.HarvesterCluster.Name, "lb"}),
			Namespace: scope.HarvesterCluster.Spec.TargetNamespace,
		},
		Spec: lbv1beta1.LoadBalancerSpec{
			Description:  "Load Balancer for cluster " + scope.HarvesterCluster.Name,
			WorkloadType: "vm",
			IPPool:       scope.HarvesterCluster.Spec.LoadBalancerConfig.IpPoolRef,
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
				FailureThreshold: failureThreshold,
				PeriodSeconds:    lbHealthCheckPeriodSections,
				TimeoutSeconds:   lbHealthCheckTimeoutSections,
			},
			BackendServerSelector: map[string][]string{
				cpVMLabelKey: {cpVMLabelValuePrefix + "-" + scope.Cluster.Name},
			},
		},
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

// createIPPoolIfNotExists is a function that creates an IP Pool in Harvester.
func createIPPoolIfNotExists(cluster *infrav1.HarvesterCluster,
	lbClient lbclient.Interface,
	machineNetwork string,
	targetVMNamespace string,
) (*lbv1beta1.IPPool, error) {
	ipPoolToCreate := lbv1beta1.IPPool{
		ObjectMeta: v1.ObjectMeta{
			Name:      locutil.GenerateRFC1035Name([]string{cluster.Namespace, cluster.Name, "ippool"}),
			Namespace: targetVMNamespace,
		},
		Spec: lbv1beta1.IPPoolSpec{
			Description: cpIPPoolDescriptionPrefix + " " + cluster.Name,
			Ranges: []lbv1beta1.Range{
				{
					Subnet:     cluster.Spec.LoadBalancerConfig.IpPool.Subnet,
					Gateway:    cluster.Spec.LoadBalancerConfig.IpPool.Gateway,
					RangeStart: cluster.Spec.LoadBalancerConfig.IpPool.RangeStart,
					RangeEnd:   cluster.Spec.LoadBalancerConfig.IpPool.RangeEnd,
				},
			},
			Selector: lbv1beta1.Selector{
				Network: machineNetwork,
			},
		},
	}

	createdIPPool, err := lbClient.LoadbalancerV1beta1().IPPools().Create(context.TODO(), &ipPoolToCreate, v1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return lbClient.LoadbalancerV1beta1().IPPools().Get(context.TODO(), ipPoolToCreate.Name, v1.GetOptions{})
		}

		cluster.Status.Conditions = append(cluster.Status.Conditions, clusterv1.Condition{
			Type:    infrav1.CustomIPPoolCreatedCondition,
			Status:  apiv1.ConditionFalse,
			Reason:  infrav1.CustomPoolCreationInHarvesterFailedReason,
			Message: "Unable to create Custom Ip Pool in Harvester",
		})
		cluster.Status.Ready = false

		return &lbv1beta1.IPPool{}, err
	}

	if createdIPPool.Name == "" {
		return &lbv1beta1.IPPool{}, errors.Errorf("IP Pool for HarvesterCluster %s could not be correctly created", cluster.Name)
	}

	cluster.Status.Conditions = append(cluster.Status.Conditions, clusterv1.Condition{
		Type:    infrav1.CustomIPPoolCreatedCondition,
		Status:  apiv1.ConditionTrue,
		Reason:  infrav1.CustomIPPoolCreatedSuccessfullyReason,
		Message: "Custom Pool was created successfully",
	})
	cluster.Status.Ready = false

	return createdIPPool, nil
}

// getOwnedCPHarversterMachines is a function that gets the HarvesterMachines that are owned by the HarvesterCluster and are controlplane machines.
func (r *HarvesterClusterReconciler) getOwnedCPHarversterMachines(scope *ClusterScope) ([]infrav1.HarvesterMachine, error) {
	// Get all the harvestermachines for the cluster.
	// Filter the harvestermachines that are controlplane machines.
	// Filter the harvestermachines that are owned by the cluster.
	// Return the harvestermachines that are owned by the cluster and are controlplane machines.
	ownedCPHarvesterMachines := &infrav1.HarvesterMachineList{}

	err := r.List(scope.Ctx,
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

	ownedCPMachinesExistInHarvester := make([]infrav1.HarvesterMachine, 0)

	for _, machine := range ownedCPHarvesterMachines.Items {
		hvMachineName := machine.Name
		hvMachineNamespace := scope.HarvesterCluster.Spec.TargetNamespace

		_, err := scope.HarvesterClient.KubevirtV1().VirtualMachines(hvMachineNamespace).Get(context.TODO(), hvMachineName, v1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				scope.Logger.V(4).Info("Owned ControlPlane Machine does not exist in Harvester yet", "machine-name", //nolint:mnd
					hvMachineName, "machine-namespace", hvMachineNamespace)

				continue
			}
		}

		ownedCPMachinesExistInHarvester = append(ownedCPMachinesExistInHarvester, machine)
	}

	return ownedCPMachinesExistInHarvester, nil
}

func getHarvesterServerFromKubeconfig(kubeconfig []byte) (server string, err error) {
	var config *clientcmdapi.Config

	config, err = clientcmd.Load(kubeconfig)
	if err != nil {
		return "", errors.Wrapf(err, "unable to load a valid harvester config from the referenced secret")
	}

	if config.CurrentContext == "" {
		return "", errors.New("the provided kubeconfig is malformed: no current-context set")
	}

	configContext := config.Contexts[config.CurrentContext]
	if configContext == nil {
		return "", errors.Errorf("the provided kubeconfig is malformed, no context section corresponds to the current-context, with the name %s",
			config.CurrentContext)
	}

	configCluster := config.Clusters[configContext.Cluster]
	if configCluster == nil {
		return "", errors.Errorf("the provided kubeconfig is malformed, no cluster section corresponds to the cluster name %s in the context %s",
			configContext.Cluster, config.CurrentContext)
	}

	configServer := configCluster.Server
	if configServer == "" {
		return "", errors.Errorf("the provided Kubeconfig is malformed, no server found for cluster %s", configContext.Cluster)
	}

	return configServer, nil
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
