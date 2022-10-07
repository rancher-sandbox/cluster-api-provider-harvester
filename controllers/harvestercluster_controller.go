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

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrastructurev1alpha1 "github.com/belgaied2/cluster-api-provider-harvester/api/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	configSecretDataKey = "kubeconfig"
)

// HarvesterClusterReconciler reconciles a HarvesterCluster object
type HarvesterClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=harvesterclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HarvesterCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *HarvesterClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cluster infrastructurev1alpha1.HarvesterCluster
	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {

		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	defer func() {
		if err := r.Patch(ctx, &cluster); err != nil {
			clusterString := cluster.Namespace + "/" + cluster.Name
			logger.Error(err, "unable to patch", "cluster", clusterString)
		}
	}()

	clusterOwner, err := util.GetOwnerCluster(ctx, r.Client, cluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}

	harvClusterOwnerRef := metav1.NewControllerRef(clusterOwner, cluster.GroupVersionKind())
	cluster.OwnerReferences = append(cluster.OwnerReferences, *harvClusterOwnerRef)

	secret := &apiv1.Secret{}
	secretKey := client.ObjectKey(cluster.Spec.IdentitySecret)

	if err := r.Client.Get(ctx, secretKey, secret); err != nil {
		return ctrl.Result{}, err
	}

	kubeconfig := secret.Data[configSecretDataKey]

	var config *clientcmdapi.Config
	if config, err = clientcmd.Load(kubeconfig); err != nil {
		cluster.Status.FailureReason = "MalformedKubeconfig"
		cluster.Status.FailureMessage = "unable to Load a valid Harvester config from the referenced Secret"

		return ctrl.Result{}, errors.Wrapf(err, "unable to Load a valid Harvester config from the referenced Secret %s", ctx)
	}

	configCluster := config.Clusters[config.CurrentContext]

	configServer := configCluster.Server

	if cluster.Spec.Server == "" {
		cluster.Spec.Server = configServer
	}

	cluster.Status.Ready = true

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HarvesterClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha1.HarvesterCluster{}).
		Complete(r)
}

func (r *HarvesterClusterReconciler) Patch(ctx context.Context, cluster *infrastructurev1alpha1.HarvesterCluster) error {
	patchHelper, err := patch.NewHelper(cluster, r.Client)
	if err != nil {
		return err
	}

	if err := patchHelper.Patch(ctx, cluster); err != nil {
		return errors.Wrapf(err, "couldn't patch harvester cluster %q", cluster.Name)
	}
	return nil
}
