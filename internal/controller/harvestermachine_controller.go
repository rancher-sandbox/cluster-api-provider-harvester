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

// Package controller contains the HarvesterMachine controller logic.
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	Ctx              context.Context
	Cluster          *clusterv1.Cluster
	Machine          *clusterv1.Machine
	HarvesterCluster *infrav1.HarvesterCluster
	HarvesterMachine *infrav1.HarvesterMachine
	HarvesterClient  *harvclient.Clientset
	ReconcilerClient client.Client
	Logger           *logr.Logger
}

const (
	vmAnnotationPVC        = "harvesterhci.io/volumeClaimTemplates"
	vmAnnotationNetworkIps = "networks.harvesterhci.io/ips"
	hvAnnotationDiskNames  = "harvesterhci.io/diskNames"
	hvAnnotationSSH        = "harvesterhci.io/sshNames"
	hvAnnotationImageID    = "harvesterhci.io/imageId"
	listImagesSelector     = "spec.displayName"
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
		return r.ReconcileDelete(hvScope)
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

		return ctrl.Result{}, nil
	}

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(hvScope.HarvesterMachine, infrav1.MachineFinalizer) && hvScope.HarvesterMachine.DeletionTimestamp.IsZero() {
		controllerutil.AddFinalizer(hvScope.HarvesterMachine, infrav1.MachineFinalizer)

		hvScope.HarvesterMachine.Status.Ready = false

		return ctrl.Result{}, nil
	}

	// Return early if the ownerCluster has infrastructureReady = false
	if !hvScope.Cluster.Status.InfrastructureReady {
		logger.Info("Waiting for Infrastructure to be ready ... ")

		hvScope.HarvesterMachine.Status.Ready = false

		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Return early if no userdata secret is referenced in ownerMachine
	if hvScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		logger.Info("Waiting for Machine's Userdata to be set ... ")

		hvScope.HarvesterMachine.Status.Ready = false

		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	vmExists := false

	// check if Harvester has a machine with the same name and namespace
	existingVM, err := hvScope.HarvesterClient.KubevirtV1().VirtualMachines(hvScope.HarvesterCluster.Spec.TargetNamespace).Get(
		context.TODO(), hvScope.HarvesterMachine.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "unable to check existence of VM from Harvester")

		hvScope.HarvesterMachine.Status.Ready = false

		return ctrl.Result{}, err
	}

	if (existingVM != nil) && (existingVM.Name == hvScope.HarvesterMachine.Name) {
		vmExists = true

		if *existingVM.Spec.Running {
			ipAddresses, err := getIPAddressesFromVMI(existingVM, hvScope.HarvesterClient)
			if err != nil {
				hvScope.HarvesterMachine.Status.Ready = false

				if apierrors.IsNotFound(err) {
					logger.Info("VM is not running yet, waiting for it to be ready")

					return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
				}

				logger.V(1).Info("unable to get IP addresses from VMI in Harvester, requeuing ...")

				return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
			}

			hvScope.HarvesterMachine.Status.Addresses = ipAddresses
			hvScope.HarvesterMachine.Status.Ready = false

			if len(ipAddresses) == 0 && hvScope.HarvesterMachine.Status.Ready {
				hvScope.HarvesterMachine.Status.Ready = false

				return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
			}

			if hvScope.HarvesterMachine.Spec.ProviderID == "" {
				providerID, _ := getProviderIDFromWorkloadCluster(hvScope, existingVM)

				if providerID != "" {
					hvScope.HarvesterMachine.Spec.ProviderID = providerID
					hvScope.HarvesterMachine.Status.Ready = true
				} else {
					logger.Info("Waiting for ProviderID to be set on Node resource in Workload Cluster ...")

					hvScope.HarvesterMachine.Status.Ready = false

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

		_, err = createVMFromHarvesterMachine(hvScope)
		if err != nil {
			logger.Error(err, "unable to create VM from HarvesterMachine information")

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

	return ctrl.Result{}, nil
}

func getProviderIDFromWorkloadCluster(hvScope *Scope, existingVM *kubevirtv1.VirtualMachine) (string, error) {
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

	if node.Spec.ProviderID == "" {
		return "harvester://" + string(existingVM.UID), nil
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
		ipAddresses = append(ipAddresses, clusterv1.MachineAddress{
			Type:    clusterv1.MachineExternalIP,
			Address: nic.IP,
		})
	}

	return ipAddresses, nil
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
	diskRandomID := locutil.RandomID()
	pvcName := vmName + "-disk-0-" + diskRandomID

	hasVMIMageName := func(volume infrav1.Volume) bool { return volume.ImageName != "" }

	// Supposing that the imageName field in HarvesterMachine.Spec.Volumes has the format "<NAMESPACE>/<NAME>",
	// we use the following to get vmImageNS and vmImageName
	imageVolumes := locutil.Filter[infrav1.Volume](hvScope.HarvesterMachine.Spec.Volumes, hasVMIMageName)

	vmImage, err := getImageFromHarvesterMachine(imageVolumes, hvScope)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find VM image reference in HarvesterMachine")
	}

	pvcAnnotation, err := buildPVCAnnotationFromImageID(&imageVolumes[0], pvcName, hvScope.HarvesterCluster.Spec.TargetNamespace, vmImage)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to generate PVC annotation on VM")
	}

	vmTemplate, err := buildVMTemplate(hvScope, pvcName, vmiLabels)
	if err != nil {
		return &kubevirtv1.VirtualMachine{}, errors.Wrap(err, "unable to build VM definition")
	}

	if vmTemplate.ObjectMeta.Labels == nil {
		vmTemplate.ObjectMeta.Labels = make(map[string]string)
	}

	ubuntuVM := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: hvScope.HarvesterCluster.Spec.TargetNamespace,
			Annotations: map[string]string{
				vmAnnotationPVC:        pvcAnnotation,
				vmAnnotationNetworkIps: "[]",
			},
			Labels: vmLabels,
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			Running: locutil.NewTrue(),

			Template: vmTemplate,
		},
	}

	hvCreatedMachine, err := hvScope.HarvesterClient.KubevirtV1().VirtualMachines(hvScope.HarvesterCluster.Spec.TargetNamespace).Create(
		context.TODO(),
		ubuntuVM,
		metav1.CreateOptions{})
	if err != nil {
		return hvCreatedMachine, err
	}

	return hvCreatedMachine, nil
}

func buildPVCAnnotationFromImageID(
	imageVolume *infrav1.Volume,
	pvcName string,
	pvcNamespace string,
	vmImage *harvesterv1beta1.VirtualMachineImage,
) (string, error) {
	block := v1.PersistentVolumeBlock
	scName := "longhorn-" + vmImage.Name
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: pvcNamespace,
			Annotations: map[string]string{
				hvAnnotationImageID: vmImage.Namespace + "/" + vmImage.Name,
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteMany,
			},
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					"storage": *imageVolume.VolumeSize,
				},
			},
			VolumeMode:       &block,
			StorageClassName: &scName,
		},
	}

	pvcJsonString, err := json.Marshal([]*v1.PersistentVolumeClaim{pvc})
	if err != nil {
		return "", err
	}

	return string(pvcJsonString), nil
}

func getImageFromHarvesterMachine(imageVolumes []infrav1.Volume, hvScope *Scope) (image *harvesterv1beta1.VirtualMachineImage, err error) {
	vmImageNamespacedString := imageVolumes[0].ImageName

	vmImageNamespacedName, err := locutil.GetNamespacedName(vmImageNamespacedString, hvScope.HarvesterCluster.Spec.TargetNamespace)
	if err != nil {
		return &harvesterv1beta1.VirtualMachineImage{}, errors.New("ImageName is HarvesterMachine is Malformed, expecting <NAMESPACE>/<NAME> format")
	}

	foundImages, err := hvScope.HarvesterClient.HarvesterhciV1beta1().VirtualMachineImages(vmImageNamespacedName.Namespace).List(
		context.TODO(), metav1.ListOptions{})
	if err != nil {
		return &harvesterv1beta1.VirtualMachineImage{}, err
	}

	if len(foundImages.Items) == 0 {
		return &harvesterv1beta1.VirtualMachineImage{}, fmt.Errorf(
			"impossible to find any VM image referenced namespace %s", vmImageNamespacedName.Namespace)
	}

	for _, image := range foundImages.Items {
		if image.Spec.DisplayName == vmImageNamespacedName.Name {
			return &image, nil
		}
	}

	return &harvesterv1beta1.VirtualMachineImage{}, fmt.Errorf(
		"impossible to find VM image %s in namespace %s", vmImageNamespacedName.Name, vmImageNamespacedName.Namespace)
}

// buildVMTemplate creates a *kubevirtv1.VirtualMachineInstanceTemplateSpec from the CLI Flags and some computed values.
func buildVMTemplate(hvScope *Scope,
	pvcName string, vmiLabels map[string]string,
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

	// create cloud-init secret for reference in Harvester.
	cloudInitSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hvScope.HarvesterMachine.Name + "-cloud-init",
			Namespace: hvScope.HarvesterCluster.Spec.TargetNamespace,
		},
		Data: map[string][]byte{
			//"userData": []byte(cloudInitUserData + cloudInitBase + cloudInitSSHSection),
			"userData": finalCloudInit,
		},
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

	vmTemplate = &kubevirtv1.VirtualMachineInstanceTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				hvAnnotationDiskNames: "[\"" + pvcName + "\"]",
				hvAnnotationSSH:       "[\"" + sshKey.GetName() + "\"]",
				cpVMLabelKey:          cpVMLabelValuePrefix + "-" + hvScope.Cluster.Name,
			},
			Labels: vmiLabels,
		},
		Spec: kubevirtv1.VirtualMachineInstanceSpec{
			Hostname: hvScope.HarvesterMachine.Name,
			Networks: getKubevirtNetworksFromHarvesterMachine(hvScope.HarvesterMachine),

			// Networks: []kubevirtv1.Network{

			// 	{
			// 		Name: "nic-1",

			// 		NetworkSource: kubevirtv1.NetworkSource{
			// 			Multus: &kubevirtv1.MultusNetwork{
			// 				NetworkName: "vlan1",
			// 			},
			// 		},
			// 	},
			// },
			Volumes: []kubevirtv1.Volume{
				{
					Name: "disk-0",
					VolumeSource: kubevirtv1.VolumeSource{
						PersistentVolumeClaim: &kubevirtv1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: v1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					},
				},
				{
					Name: "cloudinitdisk",
					VolumeSource: kubevirtv1.VolumeSource{
						CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
							UserDataSecretRef: &v1.LocalObjectReference{
								Name: hvScope.HarvesterMachine.Name + "-cloud-init",
							},
						},
					},
				},
			},
			Domain: kubevirtv1.DomainSpec{
				CPU: &kubevirtv1.CPU{
					Cores:   hvScope.HarvesterMachine.Spec.CPU,
					Sockets: hvScope.HarvesterMachine.Spec.CPU,
					Threads: hvScope.HarvesterMachine.Spec.CPU,
				},
				Devices: kubevirtv1.Devices{
					Inputs: []kubevirtv1.Input{
						{
							Bus:  "usb",
							Type: "tablet",
							Name: "tablet",
						},
					},
					Interfaces: []kubevirtv1.Interface{
						{
							Name:                   "nic-1",
							Model:                  "virtio",
							InterfaceBindingMethod: kubevirtv1.DefaultBridgeNetworkInterface().InterfaceBindingMethod,
						},
					},
					Disks: []kubevirtv1.Disk{
						{
							Name: "disk-0",
							DiskDevice: kubevirtv1.DiskDevice{
								Disk: &kubevirtv1.DiskTarget{
									Bus: "virtio",
								},
							},
						},
						{
							Name: "cloudinitdisk",
							DiskDevice: kubevirtv1.DiskDevice{
								Disk: &kubevirtv1.DiskTarget{
									Bus: "virtio",
								},
							},
						},
					},
				},
				Resources: kubevirtv1.ResourceRequirements{
					Requests: v1.ResourceList{
						"memory": resource.MustParse(hvScope.HarvesterMachine.Spec.Memory),
					},
				},
			},
			Affinity: &v1.Affinity{
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
			},
		},
	}

	return vmTemplate, nil
}

func getKubevirtNetworksFromHarvesterMachine(harvesterMachine *infrav1.HarvesterMachine) []kubevirtv1.Network {
	networks := []kubevirtv1.Network{}
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

// ReconcileDelete deletes a HarvesterMachine with all its dependencies.
func (r *HarvesterMachineReconciler) ReconcileDelete(hvScope Scope) (res ctrl.Result, rerr error) {
	logger := log.FromContext(hvScope.Ctx)
	logger.Info("Deleting HarvesterMachine ...")

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

	vm, err := hvScope.HarvesterClient.KubevirtV1().VirtualMachines(hvScope.HarvesterCluster.Spec.TargetNamespace).Get(
		hvScope.Ctx, hvScope.HarvesterMachine.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to get VM, error was different than NotFound")

			return ctrl.Result{Requeue: true}, err
		}

		logger.Info("No VM found in Harvester that corresponds to HarvesterMachine...")
	} else {
		logger.V(5).Info("found VM: " + vm.Namespace + "/" + vm.Name)

		if (vm != &kubevirtv1.VirtualMachine{}) {
			attachedPVCString := vm.Annotations[vmAnnotationPVC]
			attachedPVCObj := []*v1.PersistentVolumeClaim{}

			if attachedPVCString != "" {
				err = json.Unmarshal([]byte(attachedPVCString), &attachedPVCObj)
				if err != nil {
					return ctrl.Result{Requeue: true}, err
				}
			}

			err = hvScope.HarvesterClient.KubevirtV1().VirtualMachines(hvScope.HarvesterCluster.Spec.TargetNamespace).Delete(
				hvScope.Ctx, hvScope.HarvesterMachine.Name, metav1.DeleteOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					logger.Error(err, "unable to delete VM, error was different than NotFound")

					return ctrl.Result{Requeue: true}, err
				}

				logger.Info("VM not found, doing nothing")
				time.Sleep(time.Second)
			}

			logger.V(5).Info("VM deleted successfully: " + hvScope.HarvesterMachine.Name)

			for _, pvc := range attachedPVCObj {
				err = hvScope.HarvesterClient.CoreV1().PersistentVolumeClaims(hvScope.HarvesterCluster.Spec.TargetNamespace).Delete(
					hvScope.Ctx, pvc.Name, metav1.DeleteOptions{})
				if err != nil {
					if !apierrors.IsNotFound(err) {
						logger.Error(err, "unable to delete PVC, error was different than NotFound")

						return ctrl.Result{Requeue: true}, err
					}

					logger.Info("attached PVC not found, continuing")
				}

				logger.V(5).Info("PVC deleted successfully: " + pvc.Name)
			}
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
