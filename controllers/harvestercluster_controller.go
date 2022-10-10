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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

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
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

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

	// clusterOwner, err := util.GetOwnerCluster(ctx, r.Client, cluster.ObjectMeta)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	// harvClusterOwnerRef := metav1.NewControllerRef(clusterOwner.GetObjectMeta(), cluster.GroupVersionKind())
	// cluster.OwnerReferences = append(cluster.OwnerReferences, *harvClusterOwnerRef)
	// logger.Info("This is clusterOwner.GetObjectMeta()", "clusterOwner", clusterOwner)
	// logger.Info("This is cluster.GetObjectMeta()", "cluster", cluster)
	// if clusterOwner != nil {
	// 	controllerutil.SetControllerReference(clusterOwner.GetObjectMeta(), cluster.GetObjectMeta(), r.Scheme)
	// }

	secret := &apiv1.Secret{}
	secretKey := client.ObjectKey(cluster.Spec.IdentitySecret)

	var err error
	// var cl client.Client

	// cl, err = client.New(config.GetConfigOrDie(), client.Options{})
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	err = r.Get(ctx, secretKey, secret, &client.GetOptions{})
	if (err != nil || secret == &apiv1.Secret{}) {

		cluster.Status = infrastructurev1alpha1.HarvesterClusterStatus{
			FailureReason:  "IndentitySecretUnavailable",
			FailureMessage: "unable to find the IdentitySecret for Harvester",
			Ready:          false,
		}

		if err := r.Status().Update(ctx, &cluster, &client.UpdateOptions{}); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to update status")
		}

		return ctrl.Result{}, errors.Wrapf(err, "unable to find the IdentitySecret for Harvester %s", ctx)
	}

	kubeconfig := secret.Data[configSecretDataKey]

	var config *clientcmdapi.Config
	if config, err = clientcmd.Load(kubeconfig); err != nil {
		cluster.Status = infrastructurev1alpha1.HarvesterClusterStatus{
			FailureReason:  "MalformedIdentitySecret",
			FailureMessage: "unable to Load a valid Harvester config from the referenced Secret",
			Ready:          false,
		}

		if err := r.Status().Update(ctx, &cluster, &client.UpdateOptions{}); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to update status")
		}

		return ctrl.Result{}, errors.Wrapf(err, "unable to Load a valid Harvester config from the referenced Secret %s", ctx)
	}

	secretVersion := secret.ResourceVersion
	logger.Info("Secret ResourceVersion is:" + secretVersion)

	if cluster.Annotations == nil {
		cluster.Annotations = make(map[string]string)
	}
	cluster.Annotations["harvester.secret.version"] = secretVersion

	logger.Info("Setting Server value from Kubeconfig if needed..")
	configCluster := config.Clusters[config.CurrentContext]

	configServer := configCluster.Server

	if cluster.Spec.Server == "" {
		cluster.Spec.Server = configServer
	}

	statusReady := infrastructurev1alpha1.HarvesterClusterStatus{
		Ready: true,
	}
	// if err := r.Patch(ctx, &cluster); err != nil {
	// 	clusterString := cluster.Namespace + "/" + cluster.Name
	// 	logger.Error(err, "unable to patch", "cluster", clusterString)
	// }

	if cluster.Status != statusReady {
		cluster.Status = statusReady
		if err := r.Status().Update(ctx, &cluster, &client.UpdateOptions{}); err != nil {
			return ctrl.Result{Requeue: true}, errors.Wrapf(err, "failed to update status")
		}
	}

	logger.Info("Updating Spec fields..")
	if err := r.Update(ctx, &cluster, &client.UpdateOptions{}); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to update HarvesterCluster resource")
	}

	return ctrl.Result{}, nil
}

const (
	secretIdField = ".spec.identitySecret.name"
)

// SetupWithManager sets up the controller with the Manager.
func (r *HarvesterClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &infrastructurev1alpha1.HarvesterCluster{}, secretIdField, func(obj client.Object) []string {

		cluster := obj.(*infrastructurev1alpha1.HarvesterCluster)

		if (cluster.Spec.IdentitySecret == infrastructurev1alpha1.SecretKey{}) || cluster.Spec.IdentitySecret.Name == "" {
			return nil
		}

		clusterSecretName := cluster.Spec.IdentitySecret.Name
		return []string{clusterSecretName}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha1.HarvesterCluster{}).
		Watches(
			&source.Kind{Type: &apiv1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *HarvesterClusterReconciler) findObjectsForSecret(secret client.Object) []reconcile.Request {
	attachedClusters := &infrastructurev1alpha1.HarvesterClusterList{}
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
