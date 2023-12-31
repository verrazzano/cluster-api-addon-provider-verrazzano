/*
Copyright 2022 The Kubernetes Authors.

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

// This file from the cluster-api community (https://github.com/kubernetes-sigs/cluster-api) has been modified by Oracle.

package verrazzanofleetbinding

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	addonsv1alpha1 "github.com/verrazzano/cluster-api-addon-provider-verrazzano/api/v1alpha1"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/internal"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/k8sutils"
	helmRelease "helm.sh/helm/v3/pkg/release"
	helmDriver "helm.sh/helm/v3/pkg/storage/driver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	VerrazzanoInstallNamespace = "verrazzano-install"
)

// VerrazzanoFleetBindingReconciler reconciles a VerrazzanoFleetBinding object
type VerrazzanoFleetBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// WatchFilterValue is the label value used to filter events prior to reconciliation.
	WatchFilterValue string
}

// SetupWithManager sets up the controller with the Manager.
func (r *VerrazzanoFleetBindingReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&addonsv1alpha1.VerrazzanoFleetBinding{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(log, r.WatchFilterValue)).
		Complete(r)
}

//+kubebuilder:rbac:groups=addons.cluster.x-k8s.io,resources=verrazzanofleetbindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=addons.cluster.x-k8s.io,resources=verrazzanofleetbindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=addons.cluster.x-k8s.io,resources=verrazzanofleetbindings/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;watch
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io;bootstrap.cluster.x-k8s.io;controlplane.cluster.x-k8s.io;clusterctl.cluster.x-k8s.io,resources=*,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *VerrazzanoFleetBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	log.V(2).Info("Beginning reconcilation for VerrazzanoFleetBinding", "requestNamespace", req.Namespace, "requestName", req.Name)

	// Fetch the VerrazzanoFleetBinding instance.
	verrazzanoFleetBinding := &addonsv1alpha1.VerrazzanoFleetBinding{}
	if err := r.Client.Get(ctx, req.NamespacedName, verrazzanoFleetBinding); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(2).Info("VerrazzanoFleetBinding resource not found, skipping reconciliation", "verrazzanoFleetBinding", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// TODO: should patch helper return an error when the object has been deleted?
	patchHelper, err := patch.NewHelper(verrazzanoFleetBinding, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to init patch helper")
	}

	initializeConditions(ctx, patchHelper, verrazzanoFleetBinding)

	defer func() {
		log.V(2).Info("Preparing to patch VerrazzanoFleetBinding with return error", "verrazzanoFleetBinding", verrazzanoFleetBinding.Name, "reterr", reterr)
		if err := patchVerrazzanoFleetBinding(ctx, patchHelper, verrazzanoFleetBinding); err != nil && reterr == nil {
			reterr = err
			log.Error(err, "failed to patch VerrazzanoFleetBinding", "verrazzanoFleetBinding", verrazzanoFleetBinding.Name)
			return
		}
		log.V(2).Info("Successfully patched VerrazzanoFleetBinding", "verrazzanoFleetBinding", verrazzanoFleetBinding.Name)
	}()

	cluster := &clusterv1.Cluster{}
	clusterKey := client.ObjectKey{
		Namespace: verrazzanoFleetBinding.Spec.ClusterRef.Namespace,
		Name:      verrazzanoFleetBinding.Spec.ClusterRef.Name,
	}

	k := internal.GetterFunc
	c := &internal.HelmClient{}

	// examine DeletionTimestamp to determine if object is under deletion
	if verrazzanoFleetBinding.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoFleetBindingFinalizer) {
			controllerutil.AddFinalizer(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoFleetBindingFinalizer)
			if err := patchVerrazzanoFleetBinding(ctx, patchHelper, verrazzanoFleetBinding); err != nil {
				// TODO: Should we try to set the error here? If we can't remove the finalizer we likely can't update the status either.
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoFleetBindingFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.Client.Get(ctx, clusterKey, cluster); err == nil {
				log.V(2).Info("Getting kubeconfig for cluster", "cluster", cluster.Name)
				kubeconfig, err := k.GetClusterKubeconfig(ctx, cluster)
				if err != nil {
					wrappedErr := errors.Wrapf(err, "failed to get kubeconfig for cluster")
					conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.ClusterAvailableCondition, addonsv1alpha1.GetKubeconfigFailedReason, clusterv1.ConditionSeverityError, wrappedErr.Error())

					return ctrl.Result{}, wrappedErr
				}
				conditions.MarkTrue(verrazzanoFleetBinding, addonsv1alpha1.ClusterAvailableCondition)

				if err := r.reconcileDelete(ctx, verrazzanoFleetBinding, c, kubeconfig); err != nil {
					// if fail to delete the external dependency here, return with error
					// so that it can be retried
					return ctrl.Result{}, err
				}
			} else if apierrors.IsNotFound(err) {
				// Cluster is gone, so we should remove our finalizer from the list and delete
				log.V(2).Info("Cluster not found, no need to delete external dependency", "cluster", cluster.Name)
				// TODO: should we set a condition here?
			} else {
				wrappedErr := errors.Wrapf(err, "failed to get cluster %s/%s", clusterKey.Namespace, clusterKey.Name)
				conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.ClusterAvailableCondition, addonsv1alpha1.GetClusterFailedReason, clusterv1.ConditionSeverityError, wrappedErr.Error())

				return ctrl.Result{}, wrappedErr
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoFleetBindingFinalizer)
			if err := patchVerrazzanoFleetBinding(ctx, patchHelper, verrazzanoFleetBinding); err != nil {
				// TODO: Should we try to set the error here? If we can't remove the finalizer we likely can't update the status either.
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	if err := r.Client.Get(ctx, clusterKey, cluster); err != nil {
		// TODO: add check to tell if Cluster is deleted so we can remove the VerrazzanoFleetBinding.
		wrappedErr := errors.Wrapf(err, "failed to get cluster %s/%s", clusterKey.Namespace, clusterKey.Name)
		conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.ClusterAvailableCondition, addonsv1alpha1.GetClusterFailedReason, clusterv1.ConditionSeverityError, wrappedErr.Error())

		return ctrl.Result{}, wrappedErr
	}

	log.V(2).Info("Getting kubeconfig for cluster", "cluster", cluster.Name)
	kubeconfig, err := k.GetClusterKubeconfig(ctx, cluster)
	if err != nil {
		wrappedErr := errors.Wrapf(err, "failed to get kubeconfig for cluster")
		conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.ClusterAvailableCondition, addonsv1alpha1.GetKubeconfigFailedReason, clusterv1.ConditionSeverityError, wrappedErr.Error())

		return ctrl.Result{}, wrappedErr
	}
	conditions.MarkTrue(verrazzanoFleetBinding, addonsv1alpha1.ClusterAvailableCondition)

	log.V(2).Info("Reconciling VerrazzanoFleetBinding", "releaseProxyName", verrazzanoFleetBinding.Name)
	err = r.reconcileNormal(ctx, verrazzanoFleetBinding, c, kubeconfig)

	return ctrl.Result{Requeue: true}, err
}

// reconcileNormal handles VerrazzanoFleetBinding reconciliation when it is not being deleted. This will install or upgrade the VerrazzanoFleetBinding on the Cluster.
// It will set the ReleaseName on the VerrazzanoFleetBinding if the name is generated and also set the release status and release revision.
func (r *VerrazzanoFleetBindingReconciler) reconcileNormal(ctx context.Context, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding, client internal.Client, kubeconfig string) error {
	log := ctrl.LoggerFrom(ctx)
	k := internal.GetterFunc

	log.V(2).Info("Reconciling VerrazzanoFleetBinding on cluster", "VerrazzanoFleetBinding", verrazzanoFleetBinding.Name, "cluster", verrazzanoFleetBinding.Spec.ClusterRef.Name)

	addonsSpec, err := internal.GetVerrazzanoPlatformOperatorAddons(ctx, verrazzanoFleetBinding)
	if err != nil {
		log.Error(err, "failed to generate data")
		return err
	}

	release, err := client.InstallOrUpgradeHelmRelease(ctx, kubeconfig, addonsSpec.ValuesTemplate, addonsSpec, verrazzanoFleetBinding)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to install or upgrade release for binding '%s' on cluster %s", verrazzanoFleetBinding.GetObjectMeta().GetName(), verrazzanoFleetBinding.Spec.ClusterRef.Name))
		//log.Error(err, fmt.Sprintf("Failed to install or upgrade release '%s' on cluster %s", verrazzanoFleetBinding.Spec.ReleaseName, verrazzanoFleetBinding.Spec.ClusterRef.Name))
		conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.HelmInstallOrUpgradeFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return err
	}
	if release != nil {
		log.V(2).Info(fmt.Sprintf("Release '%s' exists on cluster %s, revision = %d", release.Name, verrazzanoFleetBinding.Spec.ClusterRef.Name, release.Version))

		status := release.Info.Status
		//verrazzanoFleetBinding.SetReleaseStatus(status.String())
		verrazzanoFleetBinding.SetReleaseRevision(release.Version)

		//if status == helmRelease.StatusDeployed {
		//	conditions.MarkTrue(verrazzanoFleetBinding, addonsv1alpha1.HelmReleaseReadyCondition)
		//} else
		if status.IsPending() {
			conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.HelmReleasePendingReason, clusterv1.ConditionSeverityInfo, fmt.Sprintf("Helm release is in a pending state: %s", status))
		} else if status == helmRelease.StatusFailed && err == nil {
			log.Info("Helm release failed without error, this might be unexpected", "release", release.Name, "cluster", verrazzanoFleetBinding.Spec.ClusterRef.Name)
			conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.HelmInstallOrUpgradeFailedReason, clusterv1.ConditionSeverityError, fmt.Sprintf("Helm release failed: %s", status))
			return err
			// TODO: should we set the error state again here?
		}

		err = setVerrazzanoPlatformOperatorConditions(ctx, client, verrazzanoFleetBinding, kubeconfig)
		if err != nil {
			log.Error(err, "Failed to update conditions for VPO")
			return err
		}
	}

	if verrazzanoFleetBinding.Spec.Verrazzano != nil {
		if verrazzanoFleetBinding.Spec.Verrazzano.Spec != nil {
			err = k.CreateOrUpdateVerrazzano(ctx, client, verrazzanoFleetBinding.Name, kubeconfig, verrazzanoFleetBinding.Spec.ClusterRef.Name, verrazzanoFleetBinding.Spec.Verrazzano.Spec)
			if err != nil {
				log.Error(err, "Failed to create or update verrazzano")
				return err
			}

			err = setVerrazzanoConditions(ctx, client, verrazzanoFleetBinding, kubeconfig)
			if err != nil {
				log.Error(err, "Failed to update conditions for verrazzano install")
				return err
			}
		}
	}

	return err
}

// reconcileDelete handles VerrazzanoFleetBinding deletion. This will uninstall the VerrazzanoFleetBinding on the Cluster or return nil if the VerrazzanoFleetBinding is not found.
func (r *VerrazzanoFleetBindingReconciler) reconcileDelete(ctx context.Context, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding, client internal.Client, kubeconfig string) error {
	log := ctrl.LoggerFrom(ctx)

	log.V(2).Info("Deleting VerrazzanoFleetBinding on cluster", "VerrazzanoFleetBinding", verrazzanoFleetBinding.Name, "cluster", verrazzanoFleetBinding.Spec.ClusterRef.Name)

	vz, err := internal.GetVerrazzanoFromRemoteCluster(ctx, client, verrazzanoFleetBinding.Name, kubeconfig, verrazzanoFleetBinding.Spec.ClusterRef.Name)
	if err != nil {
		log.Error(err, "unable to fetch verrazzano install from workload cluster")
		return err
	}

	if vz != nil {
		err = internal.DeleteVerrazzanoFromRemoteCluster(ctx, client, vz, verrazzanoFleetBinding.Name, kubeconfig, verrazzanoFleetBinding.Spec.ClusterRef.Name)
		if err != nil {
			log.Error(err, "failed to delete verrazzano")
			return err
		}

		err = internal.WaitForVerrazzanoUninstallCompletion(ctx, client, verrazzanoFleetBinding.Name, kubeconfig, verrazzanoFleetBinding.Spec.ClusterRef.Name)
		if err != nil {
			log.Error(err, "failed to uninstall verrazzano completely")
			return err
		}
	}

	log.V(1).Info(fmt.Sprintf("No Verrazzano detected on workload cluster %s", verrazzanoFleetBinding.Spec.ClusterRef.Name))

	addonsSpec, err := internal.GetVerrazzanoPlatformOperatorAddons(ctx, verrazzanoFleetBinding)
	if err != nil {
		log.Error(err, "failed to generate data")
		return err
	}

	_, err = client.GetHelmRelease(ctx, kubeconfig, addonsSpec)
	if err != nil {
		log.V(2).Error(err, "error getting release from cluster", "cluster", verrazzanoFleetBinding.Spec.ClusterRef.Name)

		if errors.Is(err, helmDriver.ErrReleaseNotFound) {
			log.V(2).Info(fmt.Sprintf("Release binding '%s' not found on cluster %s, nothing to do for uninstall", verrazzanoFleetBinding.GetObjectMeta().GetName(), verrazzanoFleetBinding.Spec.ClusterRef.Name))
			conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.HelmReleaseDeletedReason, clusterv1.ConditionSeverityInfo, "")

			return nil
		}

		conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.HelmReleaseGetFailedReason, clusterv1.ConditionSeverityError, err.Error())

		return err
	}

	log.V(2).Info("Preparing to uninstall release on cluster", "releasebindingName", verrazzanoFleetBinding.GetObjectMeta().GetName(), "clusterName", verrazzanoFleetBinding.Spec.ClusterRef.Name)

	response, err := client.UninstallHelmRelease(ctx, kubeconfig, addonsSpec)
	if err != nil {
		log.V(2).Info("Error uninstalling chart with Helm:", err)
		conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.HelmReleaseDeletionFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return errors.Wrapf(err, "error uninstalling chart with Helm on cluster %s", verrazzanoFleetBinding.Spec.ClusterRef.Name)
	}

	log.V(2).Info(fmt.Sprintf("Binding Chart '%s' successfully uninstalled on cluster %s", verrazzanoFleetBinding.GetObjectMeta().GetName(), verrazzanoFleetBinding.Spec.ClusterRef.Name))
	conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.HelmReleaseDeletedReason, clusterv1.ConditionSeverityInfo, "")
	if response != nil && response.Info != "" {
		log.V(2).Info(fmt.Sprintf("Response is %s", response.Info))
	}

	return nil
}

func initializeConditions(ctx context.Context, patchHelper *patch.Helper, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding) {
	log := ctrl.LoggerFrom(ctx)
	if len(verrazzanoFleetBinding.GetConditions()) == 0 {
		conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.PreparingToHelmInstallReason, clusterv1.ConditionSeverityInfo, "Preparing to to install Helm chart")
		if err := patchVerrazzanoFleetBinding(ctx, patchHelper, verrazzanoFleetBinding); err != nil {
			log.Error(err, "failed to patch VerrazzanoFleetBinding with initial conditions")
		}
	}
}

// patchVerrazzanoFleetBinding patches the VerrazzanoFleetBinding object and sets the ReadyCondition as an aggregate of the other condition set.
// TODO: Is this preferrable to client.Update() calls? Based on testing it seems like it avoids race conditions.
func patchVerrazzanoFleetBinding(ctx context.Context, patchHelper *patch.Helper, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding) error {
	// Update overall FleetBinding condition to Ready based on verrazano status
	if verrazzanoFleetBinding.Status.Verrazzano.State == "Ready" {
		conditions.SetSummary(verrazzanoFleetBinding,
			conditions.WithConditions(
				addonsv1alpha1.ClusterAvailableCondition,
			),
		)
	}

	// Update FleetBinding state status to match verrazano status
	verrazzanoFleetBinding.SetReleaseStatus(verrazzanoFleetBinding.Status.Verrazzano.State)

	// Patch the object, ignoring conflicts on the conditions owned by this controller.
	return patchHelper.Patch(
		ctx,
		verrazzanoFleetBinding,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			addonsv1alpha1.ClusterAvailableCondition,
		}},
		patch.WithStatusObservedGeneration{},
	)
}

func setVerrazzanoPlatformOperatorConditions(ctx context.Context, c internal.Client, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding, kubeconfig string) error {
	log := ctrl.LoggerFrom(ctx)
	k8sclient, err := c.GetWorkloadClusterK8sClient(ctx, kubeconfig)
	if err != nil {
		log.Error(err, "Unable to get k8s client")
		return err
	}

	vpoPods, err := k8sclient.CoreV1().Pods(VerrazzanoInstallNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.VerrazzanoPlatformOperatorNotUPReason, clusterv1.ConditionSeverityError, "Verrazzano Platform Operator pods are not running")
			return nil
		} else {
			log.Error(err, "Unable to fetch vpo pods from workload cluster")
			return err
		}
	}
	var podReadyVPO []bool
	for _, pod := range vpoPods.Items {
		if pod.Status.Phase != "Running" {
			if strings.Contains(pod.Name, "webhook") {
				// webhook pod
				conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.VerrazzanoPlatformOperatorWebhookNotRunningReason, clusterv1.ConditionSeverityError, "Verrazzano Platform Operator Webhook pods are not running")
			} else {
				// this will be the vpo pod
				conditions.MarkFalse(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition, addonsv1alpha1.VerrazzanoPlatformOperatorNotRunningReason, clusterv1.ConditionSeverityError, "Verrazzano Platform Operator pods are not running")
			}
			podReadyVPO = append(podReadyVPO, false)
		} else {
			podReadyVPO = append(podReadyVPO, k8sutils.IsPodReady(ctx, &pod))
		}
	}

	for _, status := range podReadyVPO {
		if !status {
			return errors.New("Not all pods for VPO are ready")
		}
	}
	// At this point all VPO pods are up and running
	conditions.MarkTrue(verrazzanoFleetBinding, addonsv1alpha1.VerrazzanoOperatorReadyCondition)
	return nil
}

func setVerrazzanoConditions(ctx context.Context, c internal.Client, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding, kubeconfig string) error {
	log := ctrl.LoggerFrom(ctx)
	vz, err := internal.GetVerrazzanoFromRemoteCluster(ctx, c, verrazzanoFleetBinding.Name, kubeconfig, verrazzanoFleetBinding.Spec.ClusterRef.Name)
	if err != nil {
		log.Error(err, "unable to fetch verrazzano install from workload cluster")
		return err
	}
	verrazzanoFleetBinding.Status.Verrazzano.Version = vz.Status.Version
	verrazzanoFleetBinding.Status.Verrazzano.ComponentsAvailable = vz.Status.Available
	verrazzanoFleetBinding.Status.Verrazzano.State = vz.Status.State
	return nil
}
