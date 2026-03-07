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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	harvesterv1beta1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
	"github.com/pkg/errors"
	kubevirtv1 "kubevirt.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"

	infrav1 "github.com/rancher-sandbox/cluster-api-provider-harvester/api/v1alpha1"
	harvclient "github.com/rancher-sandbox/cluster-api-provider-harvester/pkg/clientset/versioned"
	locutil "github.com/rancher-sandbox/cluster-api-provider-harvester/util"
)

// HarvesterMachineReconciler reconciles a HarvesterMachine object.
type HarvesterMachineReconciler struct {
	client.Client

	Scheme *runtime.Scheme
}

// Scope stores context data for the reconciler.
type Scope struct {
	Ctx                    context.Context
	Cluster                *clusterv1.Cluster
	Machine                *clusterv1.Machine
	HarvesterCluster       *infrav1.HarvesterCluster
	HarvesterMachine       *infrav1.HarvesterMachine
	HarvesterClient        *harvclient.Clientset
	ReconcilerClient       client.Client
	Logger                 *logr.Logger
	EffectiveNetworkConfig *infrav1.NetworkConfig
}

const (
	vmAnnotationPVC        = "harvesterhci.io/volumeClaimTemplates"
	vmAnnotationNetworkIps = "networks.harvesterhci.io/ips"
	hvAnnotationDiskNames  = "harvesterhci.io/diskNames"
	hvAnnotationSSH        = "harvesterhci.io/sshNames"
	hvAnnotationImageID    = "harvesterhci.io/imageId"
	listImagesSelector     = "spec.displayName"
	requeueDelay           = 10 * time.Second
)

var (
	machineInitializationProvisioned = infrav1.Initialization{
		Provisioned: true,
	}

	machineInitializationNotProvisioned = infrav1.Initialization{
		Provisioned: false,
	}
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvestermachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvestermachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvestermachines/finalizers,verbs=update
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters,verbs=get;list
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=clusters;machines,verbs=get;list;watch

// Reconcile reconciles the HarvesterMachine object.
func (r *HarvesterMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, rerr error) {
	logger := log.FromContext(ctx)
	ctx = ctrl.LoggerInto(ctx, logger)

	logger.Info("Reconciling HarvesterMachine ...")

	hvMachine := &infrav1.HarvesterMachine{}

	err := r.Get(ctx, req.NamespacedName, hvMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("harvestermachine not found")

			return ctrl.Result{}, nil
		}

		logger.Error(err, "Error happened when getting harvestermachine")

		return ctrl.Result{}, err
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(hvMachine, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always attempt to Patch the HarvesterMachine object and status after each reconciliation.
	defer func() {
		err := patchHelper.Patch(ctx,
			hvMachine,
		// conditions.WithOwnedConditions( []clusterv1.ConditionType{ clusterv1.ReadyCondition}),
		)
		if err != nil {
			logger.Error(err, "failed to patch HarvesterMachine")

			if rerr == nil {
				rerr = err
			}
		}
	}()

	ownerMachine, err := util.GetOwnerMachine(ctx, r.Client, hvMachine.ObjectMeta)
	if err != nil {
		logger.Error(err, "unable to get owner machine")

		return ctrl.Result{}, err
	}

	if ownerMachine == nil {
		logger.Info("Waiting for Machine Controller to set OwnerRef on HarvesterMachine")

		return ctrl.Result{RequeueAfter: requeueTimeShort}, nil
	}

	ownerCluster, err := util.GetClusterFromMetadata(ctx, r.Client, ownerMachine.ObjectMeta)
	if err != nil {
		logger.Info("HarvesterMachine owner Machine is missing cluster label or cluster does not exist")

		return ctrl.Result{}, err
	}

	if ownerCluster == nil {
		logger.Info("Please associate this machine with a cluster using the label " + clusterv1.ClusterNameLabel + ": <name of cluster>")

		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("machine", ownerMachine.Namespace+"/"+ownerMachine.Name, "cluster", ownerCluster.Namespace+"/"+ownerCluster.Name)
	ctx = ctrl.LoggerInto(ctx, logger)

	hvCluster := &infrav1.HarvesterCluster{}

	hvClusterKey := types.NamespacedName{
		Namespace: ownerCluster.Spec.InfrastructureRef.Namespace,
		Name:      ownerCluster.Spec.InfrastructureRef.Name,
	}

	err = r.Get(ctx, hvClusterKey, hvCluster)
	if err != nil {
		logger.Error(err, "unable to find corresponding harvestercluster to harvestermachine")

		return ctrl.Result{}, err
	}

	hvSecret, err := locutil.GetSecretForHarvesterConfig(ctx, hvCluster, r.Client)
	if err != nil {
		logger.Error(err, "unable to get Datasource secret")

		return ctrl.Result{}, err
	}

	hvClient, err := locutil.GetHarvesterClientFromSecret(hvSecret)
	if err != nil {
		logger.Error(err, "unable to create Harvester client from Datasource secret "+hvSecret.Name)
	}

	hvScope := Scope{
		Ctx:              ctx,
		Cluster:          ownerCluster,
		Machine:          ownerMachine,
		HarvesterCluster: hvCluster,
		HarvesterMachine: hvMachine,
		HarvesterClient:  hvClient,
		ReconcilerClient: r.Client,
		Logger:           &logger,
	}

	if !hvMachine.DeletionTimestamp.IsZero() {
		return r.ReconcileDelete(hvScope) //nolint:contextcheck
	}

	return r.ReconcileNormal(&hvScope) //nolint:contextcheck
}

// SetupWithManager sets up the controller with the Manager.
func (r *HarvesterMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	clusterToHarvesterMachine, err := util.ClusterToTypedObjectsMapper(mgr.GetClient(), &infrav1.HarvesterMachineList{}, mgr.GetScheme())
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.HarvesterMachine{}).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("HarvesterMachine"))),
			builder.WithPredicates(predicates.ResourceNotPaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))),
		).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterToHarvesterMachine),
			builder.WithPredicates(predicates.ClusterUnpaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))),
		).
		Complete(r)
}

// ReconcileNormal reconciles the HarvesterMachine object.
func (r *HarvesterMachineReconciler) ReconcileNormal(hvScope *Scope) (res reconcile.Result, rerr error) {
	logger := log.FromContext(hvScope.Ctx)

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(hvScope.Cluster, hvScope.HarvesterMachine) {
		logger.Info("Reconciliation is paused for this object")

		hvScope.HarvesterMachine.Status.Ready = false
		hvScope.HarvesterMachine.Status.Initialization = machineInitializationNotProvisioned

		return ctrl.Result{}, nil
	}

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(hvScope.HarvesterMachine, infrav1.MachineFinalizer) && hvScope.HarvesterMachine.DeletionTimestamp.IsZero() {
		controllerutil.AddFinalizer(hvScope.HarvesterMachine, infrav1.MachineFinalizer)

		hvScope.HarvesterMachine.Status.Ready = false
		hvScope.HarvesterMachine.Status.Initialization = machineInitializationNotProvisioned

		return ctrl.Result{}, nil
	}

	// Return early if the ownerCluster has infrastructureReady = false
	if !hvScope.Cluster.Status.InfrastructureReady {
		logger.Info("Waiting for Infrastructure to be ready ... ")

		hvScope.HarvesterMachine.Status.Ready = false
		hvScope.HarvesterMachine.Status.Initialization = machineInitializationNotProvisioned

		conditions.Set(hvScope.HarvesterMachine, &clusterv1.Condition{
			Type:    infrav1.InfrastructureReadyCondition,
			Status:  v1.ConditionFalse,
			Reason:  infrav1.InfrastructureProvisioningInProgressReason,
			Message: "Waiting for cluster infrastructure to be ready",
		})

		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Set InfrastructureReady condition when cluster infrastructure is ready
	conditions.Set(hvScope.HarvesterMachine, &clusterv1.Condition{
		Type:    infrav1.InfrastructureReadyCondition,
		Status:  v1.ConditionTrue,
		Reason:  infrav1.InfrastructureReadyReason,
		Message: "Cluster infrastructure is ready",
	})

	// Return early if no userdata secret is referenced in ownerMachine
	if hvScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		logger.Info("Waiting for Machine's Userdata to be set ... ")

		hvScope.HarvesterMachine.Status.Ready = false
		hvScope.HarvesterMachine.Status.Initialization = machineInitializationNotProvisioned

		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Resolve effective network config: cluster-level pool allocation or machine-level static config
	if hvScope.HarvesterCluster.Spec.VMNetworkConfig != nil && hvScope.HarvesterMachine.Spec.NetworkConfig == nil {
		// Allocate IP from cluster-level VM IP pool
		allocErr := r.allocateVMIP(hvScope)
		if allocErr != nil {
			logger.Error(allocErr, "failed to allocate VM IP from pool")

			conditions.Set(hvScope.HarvesterMachine, &clusterv1.Condition{
				Type:    infrav1.VMIPAllocatedCondition,
				Status:  v1.ConditionFalse,
				Reason:  infrav1.VMIPAllocationFailedReason,
				Message: fmt.Sprintf("Failed to allocate VM IP: %v", allocErr),
			})

			return ctrl.Result{RequeueAfter: requeueTimeShort}, allocErr
		}

		vmNetCfg := hvScope.HarvesterCluster.Spec.VMNetworkConfig
		hvScope.EffectiveNetworkConfig = &infrav1.NetworkConfig{
			Address:    hvScope.HarvesterMachine.Status.AllocatedIPAddress,
			Gateway:    vmNetCfg.Gateway,
			DNSServers: vmNetCfg.DNSServers,
			DNSSearch:  vmNetCfg.DNSSearch,
		}

		conditions.Set(hvScope.HarvesterMachine, &clusterv1.Condition{
			Type:    infrav1.VMIPAllocatedCondition,
			Status:  v1.ConditionTrue,
			Reason:  infrav1.VMIPAllocatedReason,
			Message: fmt.Sprintf("Allocated IP %s from pool", hvScope.HarvesterMachine.Status.AllocatedIPAddress),
		})
	} else if hvScope.HarvesterMachine.Spec.NetworkConfig != nil {
		// Use machine-level static network config
		hvScope.EffectiveNetworkConfig = hvScope.HarvesterMachine.Spec.NetworkConfig
	}

	vmExists := false

	// check if Harvester has a machine with the same name and namespace
	existingVM, err := hvScope.HarvesterClient.KubevirtV1().VirtualMachines(hvScope.HarvesterCluster.Spec.TargetNamespace).Get(
		context.TODO(), hvScope.HarvesterMachine.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "unable to check existence of VM from Harvester")

		hvScope.HarvesterMachine.Status.Ready = false
		hvScope.HarvesterMachine.Status.Initialization = machineInitializationNotProvisioned

		return ctrl.Result{}, err
	}

	if (existingVM != nil) && (existingVM.Name == hvScope.HarvesterMachine.Name) {
		vmExists = true

		// Set VMProvisioningReady condition for existing VM
		conditions.Set(hvScope.HarvesterMachine, &clusterv1.Condition{
			Type:    infrav1.VMProvisioningReadyCondition,
			Status:  v1.ConditionTrue,
			Reason:  infrav1.VMProvisioningReadyReason,
			Message: "VM already exists and is provisioned",
		})

		if isVMRunning(existingVM) {
			ipAddresses, err := getIPAddressesFromVMI(existingVM, hvScope.HarvesterClient)
			if err != nil {
				hvScope.HarvesterMachine.Status.Ready = false
				hvScope.HarvesterMachine.Status.Initialization = machineInitializationNotProvisioned

				if apierrors.IsNotFound(err) {
					logger.Info("VM is not running yet, waiting for it to be ready")

					return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
				}

				logger.V(1).Info("unable to get IP addresses from VMI in Harvester, requeuing ...")

				return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
			}

			hvScope.HarvesterMachine.Status.Addresses = ipAddresses

			if len(ipAddresses) == 0 && hvScope.HarvesterMachine.Status.Ready {
				hvScope.HarvesterMachine.Status.Ready = false
				hvScope.HarvesterMachine.Status.Initialization = machineInitializationNotProvisioned

				return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
			}

			if len(ipAddresses) > 0 && !hvScope.HarvesterMachine.Status.Ready {
				logger.Info("VM is running, IP addresses found", "addresses", ipAddresses)

				hvScope.HarvesterMachine.Status.Ready = true
				hvScope.HarvesterMachine.Status.Initialization = machineInitializationProvisioned
				conditions.MarkTrue(hvScope.HarvesterMachine, infrav1.VMProvisioningReadyCondition)

				return ctrl.Result{Requeue: true}, nil
			}

			if hvScope.HarvesterMachine.Spec.ProviderID == "" {
				providerID, err := getProviderIDFromWorkloadCluster(hvScope)
				if err != nil {
					providerID = "harvester://" + string(existingVM.UID)
				}

				if providerID != "" {
					hvScope.HarvesterMachine.Spec.ProviderID = providerID
					hvScope.HarvesterMachine.Status.Ready = true
					hvScope.HarvesterMachine.Status.Initialization = machineInitializationProvisioned
				} else {
					logger.Info("Waiting for ProviderID to be set on Node resource in Workload Cluster ...")

					return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
				}
			} else {
				conditions.MarkTrue(hvScope.HarvesterMachine, infrav1.MachineCreatedCondition)
				hvScope.HarvesterMachine.Status.Ready = true
			}
		} else {
			hvScope.HarvesterMachine.Status.Ready = false

			return ctrl.Result{RequeueAfter: requeueTimeShort}, nil
		}
	}

	if !conditions.IsTrue(hvScope.HarvesterMachine, infrav1.MachineCreatedCondition) {
		logger.Info("No existing VM found in Harvester, creating a new one ...")

		hvScope.HarvesterMachine.Status.Ready = false

		// Set VMProvisioningReady condition to in progress
		conditions.Set(hvScope.HarvesterMachine, &clusterv1.Condition{
			Type:    infrav1.VMProvisioningReadyCondition,
			Status:  v1.ConditionFalse,
			Reason:  infrav1.VMProvisioningInProgressReason,
			Message: "VM provisioning in progress",
		})

		_, err = createVMFromHarvesterMachine(hvScope)
		if err != nil {
			logger.Error(err, "unable to create VM from HarvesterMachine information")

			conditions.Set(hvScope.HarvesterMachine, &clusterv1.Condition{
				Type:    infrav1.VMProvisioningReadyCondition,
				Status:  v1.ConditionFalse,
				Reason:  infrav1.VMProvisioningFailedReason,
				Message: fmt.Sprintf("Failed to create VM: %v", err),
			})

			return ctrl.Result{}, err
		}

		conditions.MarkTrue(hvScope.HarvesterMachine, infrav1.MachineCreatedCondition)
		hvScope.HarvesterMachine.Status.Ready = false

		// Patch the HarvesterCluster resource with the InitMachineCreatedCondition if it is not already set.
		if !conditions.IsTrue(hvScope.HarvesterCluster, infrav1.InitMachineCreatedCondition) {
			hvClusterCopy := hvScope.HarvesterCluster.DeepCopy()
			conditions.MarkTrue(hvClusterCopy, infrav1.InitMachineCreatedCondition)
			hvClusterCopy.Status.Ready = hvScope.HarvesterCluster.Status.Ready

			err := r.Client.Status().Patch(hvScope.Ctx, hvClusterCopy, client.MergeFrom(hvScope.HarvesterCluster))
			if err != nil {
				logger.Error(err, "failed to update HarvesterCluster Conditions with InitMachineCreatedCondition")
			}
		}
	} else {
		if !vmExists {
			hvScope.HarvesterMachine.Status.Ready = false
			conditions.MarkFalse(hvScope.HarvesterMachine,
				infrav1.MachineCreatedCondition, infrav1.MachineNotFoundReason, clusterv1.ConditionSeverityError, "VM not found in Harvester")

			return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
		}
	}

	hvScope.HarvesterMachine.Status.Ready = true

	// Initialize workload cluster node: set providerID and remove uninitialized taint.
	// This bypasses the cloud-provider bootstrap chicken-and-egg problem.
	r.initializeWorkloadNode(hvScope)

	return ctrl.Result{}, nil
}

// initializeWorkloadNode sets the providerID and removes the cloud-provider
// uninitialized taint on the workload cluster node corresponding to this machine.
// Errors are logged as warnings but do not block reconciliation.
//
//nolint:funcorder
func (r *HarvesterMachineReconciler) initializeWorkloadNode(hvScope *Scope) {
	if hvScope.HarvesterMachine.Spec.ProviderID == "" {
		return
	}

	workloadConfig, err := getWorkloadClusterConfig(hvScope)
	if err != nil {
		// Workload cluster not ready yet, will retry on next reconcile
		return
	}

	locutil.InitializeWorkloadNode(
		hvScope.Ctx,
		*hvScope.Logger,
		workloadConfig,
		hvScope.HarvesterMachine.Name,
		hvScope.HarvesterMachine.Spec.ProviderID,
	)
}

func getProviderIDFromWorkloadCluster(hvScope *Scope) (string, error) {
	var workloadConfig *rest.Config

	workloadConfig, err := getWorkloadClusterConfig(hvScope)
	if err != nil {
		return "", errors.Wrap(err, "unable to get workload cluster config from Management Cluster")
	}

	// Get Kubernetes client for workload cluster.
	workloadClient, err := client.New(workloadConfig, client.Options{})
	if err != nil {
		return "", errors.Wrap(err, "unable to get workload cluster client from Kubeconfig")
	}

	// Get ProviderID from the Node object in the workload cluster
	node := &v1.Node{}

	err = workloadClient.Get(hvScope.Ctx, types.NamespacedName{Name: hvScope.HarvesterMachine.Name}, node)
	if err != nil {
		return "", err
	}

	return node.Spec.ProviderID, nil
}

// getWorkloadClusterConfig returns a rest.Config for the workload cluster from a secret in the management cluster.
func getWorkloadClusterConfig(hvScope *Scope) (*rest.Config, error) {
	// Get the workload cluster kubeconfig secret
	workloadClusterKubeconfigSecret := &v1.Secret{}

	err := hvScope.ReconcilerClient.Get(hvScope.Ctx, types.NamespacedName{
		Namespace: hvScope.Cluster.Namespace,
		Name:      hvScope.Cluster.Name + "-kubeconfig",
	}, workloadClusterKubeconfigSecret)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get workload cluster kubeconfig secret")
	}

	// Get the kubeconfig data from the secret
	kubeconfigData, ok := workloadClusterKubeconfigSecret.Data["value"]
	if !ok {
		return nil, fmt.Errorf("no kubeconfig data found in secret %s", workloadClusterKubeconfigSecret.Name)
	}

	// Create a rest.Config from the kubeconfig data
	workloadConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigData)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get workload cluster config from kubeconfig data")
	}

	return workloadConfig, nil
}

func runStrategyPtr(s kubevirtv1.VirtualMachineRunStrategy) *kubevirtv1.VirtualMachineRunStrategy {
	return &s
}

// isVMRunning checks whether a VM is intended to be running.
// KubeVirt VMs can use either spec.running (bool pointer) or spec.runStrategy.
// Harvester uses runStrategy instead of running, which makes spec.running nil.
// We use KubeVirt's built-in RunStrategy() method that handles both cases.
func isVMRunning(vm *kubevirtv1.VirtualMachine) bool {
	strategy, err := vm.RunStrategy()
	if err != nil {
		// Both running and runStrategy set (mutually exclusive) — fallback to status
		return vm.Status.Ready
	}

	return strategy == kubevirtv1.RunStrategyAlways ||
		strategy == kubevirtv1.RunStrategyRerunOnFailure ||
		strategy == kubevirtv1.RunStrategyOnce
}

func getIPAddressesFromVMI(existingVM *kubevirtv1.VirtualMachine, hvClient *harvclient.Clientset) ([]clusterv1.MachineAddress, error) {
	ipAddresses := []clusterv1.MachineAddress{}

	vmInstance, err := hvClient.KubevirtV1().VirtualMachineInstances(existingVM.Namespace).Get(context.TODO(), existingVM.Name, metav1.GetOptions{})
	if err != nil {
		// if apierrors.IsNotFound(err) {
		// 	return ipAddresses, fmt.Errorf("no VM instance found for VM %s", existingVM.Name)
		// }
		return ipAddresses, err
	}

	for _, nic := range vmInstance.Status.Interfaces {
		if nic.IP == "" {
			continue
		}

		ipAddresses = append(ipAddresses, clusterv1.MachineAddress{
			Type:    clusterv1.MachineExternalIP,
			Address: nic.IP,
		})
	}

	return ipAddresses, nil
}

// diskInfo holds the PVC name and volume index for each disk attached to a VM.
type diskInfo struct {
	pvcName string
	index   int
}

func createVMFromHarvesterMachine(hvScope *Scope) (*kubevirtv1.VirtualMachine, error) {
	var err error

	vmLabels := map[string]string{
		"harvesterhci.io/creator": "harvester",
	}

	if _, ok := hvScope.HarvesterMachine.Labels[clusterv1.MachineControlPlaneLabel]; ok {
		vmLabels[cpVMLabelKey] = cpVMLabelValuePrefix + "-" + hvScope.Cluster.Name
	}

	vmiLabels := vmLabels

	vmName := hvScope.HarvesterMachine.Name

	vmiLabels["harvesterhci.io/vmName"] = vmName
	vmiLabels["harvesterhci.io/vmNamePrefix"] = vmName

	targetNS := hvScope.HarvesterCluster.Spec.TargetNamespace

	// Build PVC list for all volumes
	pvcs := make([]*v1.PersistentVolumeClaim, 0, len(hvScope.HarvesterMachine.Spec.Volumes))
	disks := make([]diskInfo, 0, len(hvScope.HarvesterMachine.Spec.Volumes))

	for i, vol := range hvScope.HarvesterMachine.Spec.Volumes {
		diskRandomID := locutil.RandomID()
		pvcName := fmt.Sprintf("%s-disk-%d-%s", vmName, i, diskRandomID)

		pvc, err := buildPVCForVolume(&vol, pvcName, targetNS, hvScope)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to build PVC for volume %d", i)
		}

		pvcs = append(pvcs, pvc)
		disks = append(disks, diskInfo{pvcName: pvcName, index: i})
	}

	pvcAnnotation, err := json.Marshal(pvcs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal PVC annotations")
	}

	vmTemplate, err := buildVMTemplate(hvScope, disks, vmiLabels)
	if err != nil {
		return &kubevirtv1.VirtualMachine{}, errors.Wrap(err, "unable to build VM definition")
	}

	if vmTemplate.ObjectMeta.Labels == nil {
		vmTemplate.ObjectMeta.Labels = make(map[string]string)
	}

	ubuntuVM := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: targetNS,
			Annotations: map[string]string{
				vmAnnotationPVC:        string(pvcAnnotation),
				vmAnnotationNetworkIps: "[]",
			},
			Labels: vmLabels,
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			RunStrategy: runStrategyPtr(kubevirtv1.RunStrategyAlways),
			Template:    vmTemplate,
		},
	}

	hvCreatedMachine, err := hvScope.HarvesterClient.KubevirtV1().VirtualMachines(targetNS).Create(
		context.TODO(),
		ubuntuVM,
		metav1.CreateOptions{})
	if err != nil {
		return hvCreatedMachine, err
	}

	return hvCreatedMachine, nil
}

// buildPVCForVolume creates a PersistentVolumeClaim for a single volume.
// For "image" volumes, the PVC references a Harvester VM image (StorageClass = longhorn-<imageName>).
// For "storageClass" volumes, the PVC uses the specified StorageClass directly (blank data disk).
func buildPVCForVolume(
	vol *infrav1.Volume,
	pvcName string,
	pvcNamespace string,
	hvScope *Scope,
) (*v1.PersistentVolumeClaim, error) {
	block := v1.PersistentVolumeBlock

	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pvcName,
			Namespace:   pvcNamespace,
			Annotations: map[string]string{},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteMany,
			},
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					"storage": *vol.VolumeSize,
				},
			},
			VolumeMode: &block,
		},
	}

	switch vol.VolumeType {
	case "image":
		vmImage, err := getImageByName(vol.ImageName, pvcNamespace, hvScope)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to find VM image %s", vol.ImageName)
		}

		scName := "longhorn-" + vmImage.Name
		pvc.Spec.StorageClassName = &scName
		pvc.Annotations[hvAnnotationImageID] = vmImage.Namespace + "/" + vmImage.Name

	case "storageClass":
		scName := vol.StorageClass
		pvc.Spec.StorageClassName = &scName
	}

	return pvc, nil
}

// getImageByName resolves a Harvester VirtualMachineImage by its display name.
// imageName can be "namespace/name" or just "name" (defaults to targetNamespace).
func getImageByName(imageName, defaultNamespace string, hvScope *Scope) (*harvesterv1beta1.VirtualMachineImage, error) {
	vmImageNamespacedName, err := locutil.GetNamespacedName(imageName, defaultNamespace)
	if err != nil {
		return nil, fmt.Errorf("imageName %q is malformed, expecting <NAMESPACE>/<NAME> format: %w", imageName, err)
	}

	foundImages, err := hvScope.HarvesterClient.HarvesterhciV1beta1().VirtualMachineImages(vmImageNamespacedName.Namespace).List(
		context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, image := range foundImages.Items {
		if image.Spec.DisplayName == vmImageNamespacedName.Name {
			return &image, nil
		}
	}

	return nil, fmt.Errorf("VM image %s not found in namespace %s", vmImageNamespacedName.Name, vmImageNamespacedName.Namespace)
}

// buildVMTemplate creates a *kubevirtv1.VirtualMachineInstanceTemplateSpec from the CLI Flags and some computed values.
func buildVMTemplate(hvScope *Scope,
	disks []diskInfo, vmiLabels map[string]string,
) (vmTemplate *kubevirtv1.VirtualMachineInstanceTemplateSpec, err error) {
	var sshKey *harvesterv1beta1.KeyPair

	keyName := hvScope.HarvesterMachine.Spec.SSHKeyPair

	keyPairFullName, err := locutil.GetNamespacedName(keyName, hvScope.HarvesterCluster.Spec.TargetNamespace)
	if err != nil {
		return nil, errors.New("SSHKeyPair is HarvesterMachine is Malformed, expecting <NAMESPACE>/<NAME> format or simple DNS name without slash")
	}

	sshKey, err = hvScope.HarvesterClient.HarvesterhciV1beta1().KeyPairs(keyPairFullName.Namespace).Get(
		context.TODO(), keyPairFullName.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = fmt.Errorf(
				`no keypair could be found in namespace %s, keypair was only referenced by name %s, 
try to specify the namespace using the format <NAMESPACE>/<NAME>`,
				keyPairFullName.Namespace, keyName)

			return nil, err
		}

		err = fmt.Errorf("error during getting keypair from Harvester: %w", err)

		return nil, err
	}

	hvScope.Logger.V(3).Info("SSH Key Name " + keyName + " given does exist!") //nolint:mnd

	// building cloud-init user data
	cloudInitBase := `package_update: true
packages:
  - qemu-guest-agent
  - iptables
runcmd:
  - - systemctl
    - enable
    - --now
    - qemu-guest-agent.service`
	cloudInitSSHSection := "\nssh_authorized_keys:\n  - " + sshKey.Spec.PublicKey + "\n"

	cloudInitUserData, err1 := getCloudInitData(hvScope)
	if err1 != nil {
		err = fmt.Errorf("error during getting cloud init user data from Harvester: %w", err1)

		return nil, err
	}

	finalCloudInit, err := locutil.MergeCloudInitData(cloudInitBase, cloudInitSSHSection, cloudInitUserData)
	if err != nil {
		err = fmt.Errorf("error during merging cloud init user data from Harvester: %w", err)

		return nil, err
	}

	// Build cloud-init secret data with lowercase keys (required by KubeVirt)
	secretData := map[string][]byte{
		"userdata": finalCloudInit,
	}

	// Build network-config v1 for all NICs (static for eth0 if EffectiveNetworkConfig is set, DHCP otherwise)
	if len(hvScope.HarvesterMachine.Spec.Networks) > 0 {
		secretData["networkdata"] = []byte(buildNetworkData(hvScope))
	}

	// create cloud-init secret for reference in Harvester.
	cloudInitSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hvScope.HarvesterMachine.Name + "-cloud-init",
			Namespace: hvScope.HarvesterCluster.Spec.TargetNamespace,
		},
		Data: secretData,
	}

	hvScope.Logger.V(5).Info("cloud-init final value is " + string(finalCloudInit)) //nolint:mnd

	// check if secret already exists
	_, err = hvScope.HarvesterClient.CoreV1().Secrets(hvScope.HarvesterCluster.Spec.TargetNamespace).Get(
		context.TODO(), hvScope.HarvesterMachine.Name+"-cloud-init", metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			hvScope.Logger.V(3).Info("unable to get cloud-init secret, error was different than NotFound")
		} else {
			_, err = hvScope.HarvesterClient.CoreV1().Secrets(hvScope.HarvesterCluster.Spec.TargetNamespace).Create(
				context.TODO(), cloudInitSecret, metav1.CreateOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "unable to create cloud-init secret")
			}
		}
	} else {
		_, err = hvScope.HarvesterClient.CoreV1().Secrets(hvScope.HarvesterCluster.Spec.TargetNamespace).Update(
			context.TODO(), cloudInitSecret, metav1.UpdateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "unable to update cloud-init secret")
		}
	}

	// Build volumes and disks lists dynamically from all configured volumes
	volumes := make([]kubevirtv1.Volume, 0, len(disks)+1)
	kvDisks := make([]kubevirtv1.Disk, 0, len(disks)+1)

	for _, d := range disks {
		diskName := fmt.Sprintf("disk-%d", d.index)

		volumes = append(volumes, kubevirtv1.Volume{
			Name: diskName,
			VolumeSource: kubevirtv1.VolumeSource{
				PersistentVolumeClaim: &kubevirtv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: v1.PersistentVolumeClaimVolumeSource{
						ClaimName: d.pvcName,
					},
				},
			},
		})

		disk := kubevirtv1.Disk{
			Name: diskName,
			DiskDevice: kubevirtv1.DiskDevice{
				Disk: &kubevirtv1.DiskTarget{
					Bus: "virtio",
				},
			},
		}

		// Apply boot order if specified in the volume spec
		if d.index < len(hvScope.HarvesterMachine.Spec.Volumes) {
			bootOrder := hvScope.HarvesterMachine.Spec.Volumes[d.index].BootOrder
			if bootOrder > 0 {
				bo := uint(bootOrder)
				disk.BootOrder = &bo
			}
		}

		kvDisks = append(kvDisks, disk)
	}

	// Append cloud-init disk last
	volumes = append(volumes, kubevirtv1.Volume{
		Name: "cloudinitdisk",
		VolumeSource: kubevirtv1.VolumeSource{
			CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
				UserDataSecretRef: &v1.LocalObjectReference{
					Name: hvScope.HarvesterMachine.Name + "-cloud-init",
				},
				NetworkDataSecretRef: &v1.LocalObjectReference{
					Name: hvScope.HarvesterMachine.Name + "-cloud-init",
				},
			},
		},
	})
	kvDisks = append(kvDisks, kubevirtv1.Disk{
		Name: "cloudinitdisk",
		DiskDevice: kubevirtv1.DiskDevice{
			Disk: &kubevirtv1.DiskTarget{
				Bus: "virtio",
			},
		},
	})

	// Build network interfaces
	interfaces := buildNetworkInterfaces(hvScope.HarvesterMachine)

	// Build affinity with user-specified NodeAffinity and WorkloadAffinity
	affinity := buildAffinity(hvScope)

	// Build disk names annotation as JSON array
	diskNames := make([]string, 0, len(disks))

	for _, d := range disks {
		diskNames = append(diskNames, d.pvcName)
	}

	diskNamesJSON, err := json.Marshal(diskNames)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal disk names to JSON")
	}

	vmTemplate = &kubevirtv1.VirtualMachineInstanceTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				hvAnnotationDiskNames: string(diskNamesJSON),
				hvAnnotationSSH:       "[\"" + sshKey.GetName() + "\"]",
				cpVMLabelKey:          cpVMLabelValuePrefix + "-" + hvScope.Cluster.Name,
			},
			Labels: vmiLabels,
		},
		Spec: kubevirtv1.VirtualMachineInstanceSpec{
			Hostname: hvScope.HarvesterMachine.Name,
			Networks: getKubevirtNetworksFromHarvesterMachine(hvScope.HarvesterMachine),
			Volumes:  volumes,
			Domain: kubevirtv1.DomainSpec{
				CPU: &kubevirtv1.CPU{
					Cores:   hvScope.HarvesterMachine.Spec.CPU,
					Sockets: 1,
					Threads: 1,
				},
				Devices: kubevirtv1.Devices{
					Inputs: []kubevirtv1.Input{
						{
							Bus:  "usb",
							Type: "tablet",
							Name: "tablet",
						},
					},
					Interfaces: interfaces,
					Disks:      kvDisks,
				},
				Memory: &kubevirtv1.Memory{
					Guest: quantityPtr(resource.MustParse(hvScope.HarvesterMachine.Spec.Memory)),
				},
				Resources: kubevirtv1.ResourceRequirements{
					Requests: v1.ResourceList{
						"memory": resource.MustParse(hvScope.HarvesterMachine.Spec.Memory),
					},
					Limits: v1.ResourceList{
						"cpu":    *resource.NewQuantity(int64(hvScope.HarvesterMachine.Spec.CPU), resource.DecimalSI),
						"memory": resource.MustParse(hvScope.HarvesterMachine.Spec.Memory),
					},
				},
			},
			Affinity: affinity,
		},
	}

	return vmTemplate, nil
}

// buildNetworkInterfaces creates kubevirt Interface specs for each network.
func buildNetworkInterfaces(machine *infrav1.HarvesterMachine) []kubevirtv1.Interface {
	interfaces := make([]kubevirtv1.Interface, 0, len(machine.Spec.Networks))

	for i := range machine.Spec.Networks {
		interfaces = append(interfaces, kubevirtv1.Interface{
			Name:                   "nic-" + strconv.Itoa(i+1),
			Model:                  "virtio",
			InterfaceBindingMethod: kubevirtv1.DefaultBridgeNetworkInterface().InterfaceBindingMethod,
		})
	}

	return interfaces
}

// buildNetworkData generates cloud-init network-config v1 YAML for all NICs.
// If EffectiveNetworkConfig is set, eth0 gets a static config; additional NICs get DHCP.
// If EffectiveNetworkConfig is nil, all NICs get DHCP.
func buildNetworkData(hvScope *Scope) string {
	var b strings.Builder

	b.WriteString("version: 1\nconfig:\n")

	for i := range hvScope.HarvesterMachine.Spec.Networks {
		ethName := "eth" + strconv.Itoa(i)

		fmt.Fprintf(&b, "  - type: physical\n    name: %s\n    subnets:\n", ethName)

		if i == 0 && hvScope.EffectiveNetworkConfig != nil {
			netCfg := hvScope.EffectiveNetworkConfig
			subnetMask := "255.255.0.0" // default

			if hvScope.HarvesterCluster.Spec.VMNetworkConfig != nil {
				subnetMask = hvScope.HarvesterCluster.Spec.VMNetworkConfig.SubnetMask
			}

			fmt.Fprintf(&b, "      - type: static\n        address: %s\n        netmask: %s\n        gateway: %s\n",
				netCfg.Address, subnetMask, netCfg.Gateway)
		} else {
			b.WriteString("      - type: dhcp\n")
		}
	}

	// Collect DNS servers from EffectiveNetworkConfig or VMNetworkConfig
	var dnsServers []string

	var dnsSearch []string

	if hvScope.EffectiveNetworkConfig != nil {
		dnsServers = hvScope.EffectiveNetworkConfig.DNSServers
		dnsSearch = hvScope.EffectiveNetworkConfig.DNSSearch
	} else if hvScope.HarvesterCluster.Spec.VMNetworkConfig != nil {
		dnsServers = hvScope.HarvesterCluster.Spec.VMNetworkConfig.DNSServers
		dnsSearch = hvScope.HarvesterCluster.Spec.VMNetworkConfig.DNSSearch
	}

	if len(dnsServers) > 0 {
		b.WriteString("  - type: nameserver\n    address:\n")

		for _, dns := range dnsServers {
			fmt.Fprintf(&b, "      - %s\n", dns)
		}

		if len(dnsSearch) > 0 {
			b.WriteString("    search:\n")

			for _, search := range dnsSearch {
				fmt.Fprintf(&b, "      - %s\n", search)
			}
		}
	}

	return b.String()
}

// buildAffinity merges user-specified NodeAffinity and WorkloadAffinity with default PodAntiAffinity.
func buildAffinity(hvScope *Scope) *v1.Affinity {
	affinity := &v1.Affinity{
		PodAntiAffinity: &v1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
				{
					Weight: int32(1),
					PodAffinityTerm: v1.PodAffinityTerm{
						TopologyKey: "kubernetes.io/hostname",
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"harvesterhci.io/vmNamePrefix": hvScope.HarvesterMachine.Name,
							},
						},
					},
				},
			},
		},
	}

	// Merge user-specified NodeAffinity
	if hvScope.HarvesterMachine.Spec.NodeAffinity != nil {
		affinity.NodeAffinity = hvScope.HarvesterMachine.Spec.NodeAffinity
	}

	// Merge user-specified WorkloadAffinity (PodAffinity)
	if hvScope.HarvesterMachine.Spec.WorkloadAffinity != nil {
		affinity.PodAffinity = hvScope.HarvesterMachine.Spec.WorkloadAffinity
	}

	return affinity
}

func getKubevirtNetworksFromHarvesterMachine(harvesterMachine *infrav1.HarvesterMachine) []kubevirtv1.Network {
	networks := make([]kubevirtv1.Network, 0, len(harvesterMachine.Spec.Networks))
	for i, network := range harvesterMachine.Spec.Networks {
		networks = append(networks, kubevirtv1.Network{
			Name: "nic-" + strconv.Itoa(i+1),
			NetworkSource: kubevirtv1.NetworkSource{
				Multus: &kubevirtv1.MultusNetwork{
					NetworkName: network,
				},
			},
		})
	}

	return networks
}

func getCloudInitData(hvScope *Scope) (string, error) {
	dataSecretNamespacedName := types.NamespacedName{
		Namespace: hvScope.Machine.Namespace,
		Name:      *hvScope.Machine.Spec.Bootstrap.DataSecretName,
	}

	dataSecret := &v1.Secret{}

	err := hvScope.ReconcilerClient.Get(hvScope.Ctx, dataSecretNamespacedName, dataSecret)
	if err != nil {
		return "", err
	}

	userData, ok := dataSecret.Data["value"]
	if !ok {
		return "", fmt.Errorf("no userData key found in secret %s", dataSecretNamespacedName)
	}

	return string(userData), nil
}

// allocateVMIP allocates an IP from the cluster-level VM IP pool for this machine.
// It is idempotent: if an IP is already allocated, it returns early.
//
//nolint:funcorder
func (r *HarvesterMachineReconciler) allocateVMIP(hvScope *Scope) error {
	machine := hvScope.HarvesterMachine
	logger := hvScope.Logger

	// Return early if already allocated (idempotent)
	if machine.Status.AllocatedIPAddress != "" {
		logger.V(3).Info("VM IP already allocated", "ip", machine.Status.AllocatedIPAddress)

		return nil
	}

	vmNetCfg := hvScope.HarvesterCluster.Spec.VMNetworkConfig
	if vmNetCfg == nil {
		return errors.New("VMNetworkConfig is nil on HarvesterCluster")
	}

	poolRef := vmNetCfg.IPPoolRef
	if poolRef == "" {
		return errors.New("VMNetworkConfig.IPPoolRef is empty, ensure reconcileVMIPPool has run")
	}

	// Get the pool from Harvester
	pool, err := hvScope.HarvesterClient.LoadbalancerV1beta1().IPPools().Get(
		context.TODO(), poolRef, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get VM IP pool %s", poolRef)
	}

	// Check for existing allocation in pool
	machineID := machine.Namespace + "/" + machine.Name

	if pool.Status.Allocated != nil {
		for ip, id := range pool.Status.Allocated {
			if id == machineID {
				machine.Status.AllocatedIPAddress = ip
				logger.Info("Found existing IP allocation in pool", "ip", ip)

				return nil
			}
		}
	}

	// Allocate new IP
	allocatedIP, err := locutil.AllocateVMIPFromPool(pool, machineID)
	if err != nil {
		return errors.Wrap(err, "failed to allocate IP from VM IP pool")
	}

	// Update pool in Harvester
	_, err = hvScope.HarvesterClient.LoadbalancerV1beta1().IPPools().Update(
		context.TODO(), pool, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update VM IP pool after allocation")
	}

	machine.Status.AllocatedIPAddress = allocatedIP
	logger.Info("Allocated VM IP from pool", "ip", allocatedIP, "machine", machine.Name)

	return nil
}

// releaseVMIP releases the allocated IP back to the pool during machine deletion.
// Errors are logged as warnings but do not block deletion.
//
//nolint:funcorder
func (r *HarvesterMachineReconciler) releaseVMIP(hvScope *Scope) {
	machine := hvScope.HarvesterMachine
	logger := hvScope.Logger

	if machine.Status.AllocatedIPAddress == "" {
		return
	}

	vmNetCfg := hvScope.HarvesterCluster.Spec.VMNetworkConfig
	if vmNetCfg == nil || vmNetCfg.IPPoolRef == "" {
		logger.Info("No VMNetworkConfig/IPPoolRef, skipping IP release")

		return
	}

	pool, err := hvScope.HarvesterClient.LoadbalancerV1beta1().IPPools().Get(
		context.TODO(), vmNetCfg.IPPoolRef, metav1.GetOptions{})
	if err != nil {
		logger.Info("Warning: failed to get VM IP pool for release, skipping", "error", err)

		return
	}

	store := locutil.NewStore(pool)
	ip := net.ParseIP(machine.Status.AllocatedIPAddress)

	if ip == nil {
		logger.Info("Warning: failed to parse allocated IP for release", "ip", machine.Status.AllocatedIPAddress)

		return
	}

	releaseErr := store.Release(ip)
	if releaseErr != nil {
		logger.Info("Warning: failed to release IP from pool", "error", releaseErr, "ip", machine.Status.AllocatedIPAddress)

		return
	}

	_, err = hvScope.HarvesterClient.LoadbalancerV1beta1().IPPools().Update(
		context.TODO(), pool, metav1.UpdateOptions{})
	if err != nil {
		logger.Info("Warning: failed to update pool after IP release", "error", err)

		return
	}

	logger.Info("Released VM IP back to pool", "ip", machine.Status.AllocatedIPAddress, "machine", machine.Name)
}

// removeEtcdMemberIfControlPlane removes the etcd member for a control-plane machine
// from the workload cluster before VM deletion. This prevents stale etcd members from
// blocking replacement nodes from joining the cluster.
// Errors are logged as warnings but do not block deletion.
//
//nolint:funcorder
func (r *HarvesterMachineReconciler) removeEtcdMemberIfControlPlane(hvScope *Scope) {
	logger := hvScope.Logger

	// Skip if not a control-plane machine
	if _, ok := hvScope.HarvesterMachine.Labels[clusterv1.MachineControlPlaneLabel]; !ok {
		return
	}

	workloadConfig, err := getWorkloadClusterConfig(hvScope)
	if err != nil {
		logger.Info("Warning: failed to get workload cluster config for etcd cleanup, skipping",
			"error", err)

		return
	}

	locutil.RemoveEtcdMember(hvScope.Ctx, *logger, workloadConfig, hvScope.HarvesterMachine.Name)
}

// ReconcileDelete deletes a HarvesterMachine with all its dependencies.
func (r *HarvesterMachineReconciler) ReconcileDelete(hvScope Scope) (res ctrl.Result, rerr error) {
	logger := log.FromContext(hvScope.Ctx)
	logger.Info("Deleting HarvesterMachine ...")

	// Release allocated IP back to pool before deletion
	r.releaseVMIP(&hvScope)

	// Remove etcd member from workload cluster before VM deletion (control-plane only)
	r.removeEtcdMemberIfControlPlane(&hvScope)

	err := hvScope.HarvesterClient.CoreV1().Secrets(hvScope.HarvesterCluster.Spec.TargetNamespace).Delete(
		hvScope.Ctx, hvScope.HarvesterMachine.Name+"-cloud-init", metav1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to delete cloud-init secret, error was different than NotFound")

			return ctrl.Result{Requeue: true}, err
		}

		logger.Info("cloud-init secret not found, doing nothing")
	}

	logger.V(5).Info("cloud-init secret deleted successfully: " + hvScope.HarvesterMachine.Name + "-cloud-init")

	targetNS := hvScope.HarvesterCluster.Spec.TargetNamespace
	machineName := hvScope.HarvesterMachine.Name

	vm, err := hvScope.HarvesterClient.KubevirtV1().VirtualMachines(targetNS).Get(
		hvScope.Ctx, machineName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to get VM, error was different than NotFound")

			return ctrl.Result{Requeue: true}, err
		}

		// VM is gone — clean up any orphaned PVCs by name prefix
		logger.Info("No VM found, cleaning up orphaned PVCs")
		r.deletePVCsByPrefix(hvScope.Ctx, &hvScope, targetNS, machineName+"-disk-")
	} else {
		logger.V(5).Info("found VM: " + vm.Namespace + "/" + vm.Name)

		if (vm != &kubevirtv1.VirtualMachine{}) {
			// Delete VM first
			err = hvScope.HarvesterClient.KubevirtV1().VirtualMachines(targetNS).Delete(
				hvScope.Ctx, machineName, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				logger.Error(err, "unable to delete VM, error was different than NotFound")

				return ctrl.Result{Requeue: true}, err
			}

			logger.Info("VM delete requested, requeuing to wait for termination before PVC cleanup")

			return ctrl.Result{RequeueAfter: requeueDelay}, nil
		}
	}

	if ok := controllerutil.RemoveFinalizer(hvScope.HarvesterMachine, infrav1.MachineFinalizer); !ok {
		return ctrl.Result{}, fmt.Errorf("unable to remove finalizer %s from HarvesterMachine %s/%s",
			infrav1.MachineFinalizer,
			hvScope.HarvesterMachine.Namespace,
			hvScope.HarvesterMachine.Name)
	}

	return ctrl.Result{}, nil
}

// deletePVCsByPrefix lists all PVCs in the namespace and deletes those whose name
// starts with the given prefix. This handles orphaned PVCs left behind when VM
// deletion completes before PVC cleanup.
func (r *HarvesterMachineReconciler) deletePVCsByPrefix(ctx context.Context, hvScope *Scope, namespace, prefix string) {
	logger := hvScope.Logger

	pvcList, err := hvScope.HarvesterClient.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Info("Warning: failed to list PVCs for cleanup", "error", err)

		return
	}

	for i := range pvcList.Items {
		pvc := &pvcList.Items[i]
		if !strings.HasPrefix(pvc.Name, prefix) {
			continue
		}

		err = hvScope.HarvesterClient.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			logger.Info("Warning: failed to delete orphaned PVC", "pvc", pvc.Name, "error", err)

			continue
		}

		logger.Info("Deleted orphaned PVC", "pvc", pvc.Name)
	}
}

// quantityPtr returns a pointer to a resource.Quantity.
func quantityPtr(q resource.Quantity) *resource.Quantity {
	return &q
}
