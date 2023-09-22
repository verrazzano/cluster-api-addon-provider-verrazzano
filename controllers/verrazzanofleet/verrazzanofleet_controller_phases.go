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

package verrazzanofleet

import (
	"context"
	"github.com/pkg/errors"
	addonsv1alpha1 "github.com/verrazzano/cluster-api-addon-provider-verrazzano/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deleteOrphanedVerrazzanoFleetBindings deletes any VerrazzanoFleetBinding resources that belong to a Cluster that is not selected by its parent VerrazzanoFleet.
func (r *VerrazzanoFleetReconciler) deleteOrphanedVerrazzanoFleetBindings(ctx context.Context, verrazzanoFleet *addonsv1alpha1.VerrazzanoFleet, clusters *clusterv1.Cluster, verrazzanoFleetBindings []addonsv1alpha1.VerrazzanoFleetBinding) error {
	log := ctrl.LoggerFrom(ctx)

	releasesToDelete := getOrphanedVerrazzanoFleetBindings(ctx, clusters, verrazzanoFleetBindings)
	log.V(2).Info("Deleting orphaned releases")
	for _, release := range releasesToDelete {
		log.V(2).Info("Deleting release", "release", release)
		if err := r.deleteVerrazzanoFleetBinding(ctx, &release); err != nil {
			conditions.MarkFalse(verrazzanoFleet, addonsv1alpha1.VerrazzanoFleetBindingSpecsCreatedOrUpDatedCondition, addonsv1alpha1.VerrazzanoFleetBindingDeletionFailedReason, clusterv1.ConditionSeverityError, err.Error())
			return err
		}
	}

	return nil
}

// reconcileForCluster will create or update a VerrazzanoFleetBinding for the given cluster.
func (r *VerrazzanoFleetReconciler) reconcileForCluster(ctx context.Context, verrazzanoFleet *addonsv1alpha1.VerrazzanoFleet, cluster *clusterv1.Cluster) error {
	log := ctrl.LoggerFrom(ctx)

	existingVerrazzanoFleetBinding, err := r.getExistingVerrazzanoFleetBinding(ctx, verrazzanoFleet, cluster)
	if err != nil {
		// TODO: Should we set a condition here?
		return errors.Wrapf(err, "failed to get VerrazzanoFleetBinding for cluster %s", cluster.Name)
	}

	// Currently this helps in logging and prints out spec changes if any
	if existingVerrazzanoFleetBinding != nil && shouldFleetBindingChange(ctx, existingVerrazzanoFleetBinding, verrazzanoFleet) {
		log.V(2).Info("VerrazzanoFleetBinding found", "verrazzanoFleetBinding", existingVerrazzanoFleetBinding.Name, "cluster", cluster.Name)
	}

	//log.V(2).Info("Values for cluster", "cluster", cluster.Name, "values", values)
	log.V(2).Info("Values for cluster", "cluster", cluster.Name)
	if err := r.createOrUpdateVerrazzanoFleetBinding(ctx, existingVerrazzanoFleetBinding, verrazzanoFleet, cluster); err != nil {
		conditions.MarkFalse(verrazzanoFleet, addonsv1alpha1.VerrazzanoFleetBindingSpecsCreatedOrUpDatedCondition, addonsv1alpha1.VerrazzanoFleetBindingCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return errors.Wrapf(err, "failed to create or update VerrazzanoFleetBinding on cluster %s", cluster.Name)
	}
	return nil
}

// getExistingVerrazzanoFleetBinding returns the VerrazzanoFleetBinding for the given cluster if it exists.
func (r *VerrazzanoFleetReconciler) getExistingVerrazzanoFleetBinding(ctx context.Context, verrazzanoFleet *addonsv1alpha1.VerrazzanoFleet, cluster *clusterv1.Cluster) (*addonsv1alpha1.VerrazzanoFleetBinding, error) {
	log := ctrl.LoggerFrom(ctx)

	verrazzanoFleetBindingList := &addonsv1alpha1.VerrazzanoFleetBindingList{}

	listOpts := []client.ListOption{
		client.MatchingLabels{
			clusterv1.ClusterNameLabel:              cluster.Name,
			addonsv1alpha1.VerrazzanoFleetLabelName: verrazzanoFleet.Name,
		},
	}

	// TODO: Figure out if we want this search to be cross-namespaces.

	log.V(2).Info("Attempting to fetch existing VerrazzanoFleetBinding with Cluster and VerrazzanoFleet labels", "cluster", cluster.Name, "verrazzanoFleet", verrazzanoFleet.Name)
	if err := r.Client.List(context.TODO(), verrazzanoFleetBindingList, listOpts...); err != nil {
		return nil, err
	}

	if verrazzanoFleetBindingList.Items == nil || len(verrazzanoFleetBindingList.Items) == 0 {
		log.V(2).Info("No VerrazzanoFleetBinding found matching the cluster and VerrazzanoFleet", "cluster", cluster.Name, "verrazzanoFleet", verrazzanoFleet.Name)
		return nil, nil
	} else if len(verrazzanoFleetBindingList.Items) > 1 {
		log.V(2).Info("Multiple VerrazzanoFleetBindings found matching the cluster and VerrazzanoFleet", "cluster", cluster.Name, "verrazzanoFleet", verrazzanoFleet.Name)
		return nil, errors.Errorf("multiple VerrazzanoFleetBindings found matching the cluster and VerrazzanoFleet")
	}

	log.V(2).Info("Found existing matching VerrazzanoFleetBinding", "cluster", cluster.Name, "verrazzanoFleet", verrazzanoFleet.Name)

	return &verrazzanoFleetBindingList.Items[0], nil
}

// createOrUpdateVerrazzanoFleetBinding creates or updates the VerrazzanoFleetBinding for the given cluster.
func (r *VerrazzanoFleetReconciler) createOrUpdateVerrazzanoFleetBinding(ctx context.Context, existing *addonsv1alpha1.VerrazzanoFleetBinding, verrazzanoFleet *addonsv1alpha1.VerrazzanoFleet, cluster *clusterv1.Cluster) error {
	log := ctrl.LoggerFrom(ctx)
	verrazzanoFleetBinding := constructVerrazzanoFleetBinding(existing, verrazzanoFleet, cluster)
	if verrazzanoFleetBinding == nil {
		log.V(2).Info("VerrazzanoFleetBinding is up to date, nothing to do", "verrazzanoFleetBinding", existing.Name, "cluster", cluster.Name)
		return nil
	}
	if existing == nil {
		if err := r.Client.Create(ctx, verrazzanoFleetBinding); err != nil {
			if apierrors.IsAlreadyExists(err) {
				log.V(2).Info("verrazzanoFleetBinding already exists", "verrazzanoFleetBinding name", verrazzanoFleetBinding.Name, "cluster name", cluster.Name)
				conditions.MarkTrue(verrazzanoFleet, addonsv1alpha1.VerrazzanoFleetBindingExistsReason)
				return nil
			}
			return errors.Wrapf(err, "failed to create VerrazzanoFleetBinding '%s' for cluster: %s/%s", verrazzanoFleetBinding.Name, cluster.Namespace, cluster.Name)
		}
	} else {
		// TODO: should this use patchVerrazzanoFleetBinding() instead of Update() in case there's a race condition?
		if err := r.Client.Update(ctx, verrazzanoFleetBinding); err != nil {
			return errors.Wrapf(err, "failed to update VerrazzanoFleetBinding '%s' for cluster: %s/%s", verrazzanoFleetBinding.Name, cluster.Namespace, cluster.Name)
		}
	}

	return nil
}

// deleteVerrazzanoFleetBinding deletes the VerrazzanoFleetBinding for the given cluster.
func (r *VerrazzanoFleetReconciler) deleteVerrazzanoFleetBinding(ctx context.Context, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding) error {
	log := ctrl.LoggerFrom(ctx)

	if err := r.Client.Delete(ctx, verrazzanoFleetBinding); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(2).Info("VerrazzanoFleetBinding already deleted, nothing to do", "verrazzanoFleetBinding", verrazzanoFleetBinding.Name)
			return nil
		}
		return errors.Wrapf(err, "failed to delete verrazzanoFleetBinding: %s", verrazzanoFleetBinding.Name)
	}

	return nil
}

// constructVerrazzanoFleetBinding constructs a new VerrazzanoFleetBinding for the given Cluster or updates the existing VerrazzanoFleetBinding if needed.
// If no update is needed, this returns nil. Note that this does not check if we need to reinstall the VerrazzanoFleetBinding, i.e. immutable fields changed.
func constructVerrazzanoFleetBinding(existing *addonsv1alpha1.VerrazzanoFleetBinding, verrazzanoFleet *addonsv1alpha1.VerrazzanoFleet, cluster *clusterv1.Cluster) *addonsv1alpha1.VerrazzanoFleetBinding {
	verrazzanoFleetBinding := &addonsv1alpha1.VerrazzanoFleetBinding{}
	if existing == nil {
		verrazzanoFleetBinding.Name = cluster.Name
		verrazzanoFleetBinding.Namespace = verrazzanoFleet.Namespace
		verrazzanoFleetBinding.OwnerReferences = util.EnsureOwnerRef(verrazzanoFleetBinding.OwnerReferences, *metav1.NewControllerRef(verrazzanoFleet, verrazzanoFleet.GroupVersionKind()))

		newLabels := map[string]string{}
		newLabels[clusterv1.ClusterNameLabel] = cluster.Name
		newLabels[addonsv1alpha1.VerrazzanoFleetLabelName] = verrazzanoFleet.Name

		for k, v := range verrazzanoFleet.Labels {
			if _, ok := verrazzanoFleet.Labels[k]; ok {
				newLabels[k] = v
			}
		}
		verrazzanoFleetBinding.Labels = newLabels

		verrazzanoFleetBinding.Spec.ClusterRef = corev1.ObjectReference{
			Kind:       cluster.Kind,
			APIVersion: cluster.APIVersion,
			Name:       cluster.Name,
			Namespace:  cluster.Namespace,
		}

		verrazzanoFleet.ObjectMeta.SetAnnotations(verrazzanoFleetBinding.Annotations)
	} else {
		verrazzanoFleetBinding = existing
		changed := false

		if existing.Spec.Image != nil {
			if verrazzanoFleet.Spec.Image != nil {
				if existing.Spec.Image.Repository != verrazzanoFleet.Spec.Image.Repository {
					changed = true
				}
			}
		}

		if existing.Spec.PrivateRegistry != nil {
			if verrazzanoFleet.Spec.PrivateRegistry != nil {
				if existing.Spec.PrivateRegistry.Enabled != verrazzanoFleet.Spec.PrivateRegistry.Enabled {
					changed = true
				}
			}
		}

		if existing.Spec.ImagePullSecrets != nil {
			if verrazzanoFleet.Spec.ImagePullSecrets != nil {
				if existing.Spec.ImagePullSecrets[0].Name != verrazzanoFleet.Spec.ImagePullSecrets[0].Name {
					changed = true
				}
			}
		}

		if existing.Spec.Verrazzano != nil {
			if existing.Spec.Verrazzano.Spec != nil {
				if verrazzanoFleet.Spec.Verrazzano.Spec != nil {
					if existing.Spec.Verrazzano.Spec.String() != verrazzanoFleet.Spec.Verrazzano.Spec.String() {
						changed = true
					}
				}
			}
		}

		if !changed {
			return nil
		}
	}

	if verrazzanoFleet.Spec.Image != nil {
		verrazzanoFleetBinding.Spec.Image = verrazzanoFleet.Spec.Image
	}

	if verrazzanoFleet.Spec.Verrazzano != nil {
		if verrazzanoFleet.Spec.Verrazzano.Spec != nil {
			verrazzanoFleetBinding.Spec.Verrazzano = verrazzanoFleet.Spec.Verrazzano
		}
	}

	if verrazzanoFleet.Spec.ImagePullSecrets != nil {
		verrazzanoFleetBinding.Spec.ImagePullSecrets = verrazzanoFleet.Spec.ImagePullSecrets
	}

	if verrazzanoFleet.Spec.PrivateRegistry != nil {
		verrazzanoFleetBinding.Spec.PrivateRegistry = verrazzanoFleet.Spec.PrivateRegistry
	}

	return verrazzanoFleetBinding
}

// shouldFleetBindingChange returns true if the VerrazzanoFleetBinding needs to be reinstalled. This is the case if any of the immutable fields changed.
func shouldFleetBindingChange(ctx context.Context, existing *addonsv1alpha1.VerrazzanoFleetBinding, verrazzanoFleet *addonsv1alpha1.VerrazzanoFleet) bool {
	log := ctrl.LoggerFrom(ctx)

	log.V(2).Info("Checking if VerrazzanoFleetBinding needs to be reinstalled by by checking if immutable fields changed", "verrazzanoFleetBinding", existing.Name)

	if existing.Spec.Image != nil {
		if verrazzanoFleet.Spec.Image != nil {
			switch {
			case existing.Spec.Image.Repository != verrazzanoFleet.Spec.Image.Repository:
				log.V(2).Info("Image Repository changed", "existing", existing.Spec.Image.Repository, "verrazzanoFleet", verrazzanoFleet.Spec.Image.Repository)
				return true
			case existing.Spec.Image.Tag != verrazzanoFleet.Spec.Image.Tag:
				log.V(2).Info("Image Tag changed", "existing", existing.Spec.Image.Tag, "verrazzanoFleet", verrazzanoFleet.Spec.Image.Tag)
				return true
			case existing.Spec.Image.PullPolicy != verrazzanoFleet.Spec.Image.PullPolicy:
				log.V(2).Info("Image PullPolicy changed", "existing", existing.Spec.Image.PullPolicy, "verrazzanoFleet", verrazzanoFleet.Spec.Image.PullPolicy)
				return true
			}
		}
	}

	if existing.Spec.ImagePullSecrets != nil {
		if verrazzanoFleet.Spec.ImagePullSecrets != nil {
			if existing.Spec.ImagePullSecrets[0].Name != verrazzanoFleet.Spec.ImagePullSecrets[0].Name {
				log.V(2).Info("ImagePullSecret Name changed", "existing", existing.Spec.ImagePullSecrets[0].Name, "verrazzanoFleet", verrazzanoFleet.Spec.ImagePullSecrets[0].Name)
				return true
			}
		}
	}
	if existing.Spec.Verrazzano != nil {
		if existing.Spec.Verrazzano.Spec != nil {
			if verrazzanoFleet.Spec.Verrazzano.Spec != nil {
				if existing.Spec.Verrazzano.Spec.String() != verrazzanoFleet.Spec.Verrazzano.Spec.String() {
					existingData, err := existing.Spec.Verrazzano.Spec.Marshal()
					if err != nil {
						log.Error(err, "existing object failed in marshall")
						return false
					}
					newData, err := verrazzanoFleet.Spec.Verrazzano.Spec.Marshal()
					if err != nil {
						log.Error(err, "new object failed in marshall")
						return false
					}
					log.V(2).Info("Verrazzano spec has changed", "existing", string(existingData), "verrazzanoFleet", string(newData))
					return true
				}
			}
		}
	}

	if existing.Spec.PrivateRegistry != nil {
		if verrazzanoFleet.Spec.PrivateRegistry != nil {
			if existing.Spec.PrivateRegistry.Enabled != verrazzanoFleet.Spec.PrivateRegistry.Enabled {
				log.V(2).Info("Private registry has changed", "existing", existing.Spec.PrivateRegistry.Enabled, "verrazzanoFleet", verrazzanoFleet.Spec.PrivateRegistry.Enabled)
				return true
			}
		}
	}

	return false
}

// getOrphanedVerrazzanoFleetBindings returns a list of VerrazzanoFleetBindings that are not associated with any of the selected Clusters for a given VerrazzanoFleet.
func getOrphanedVerrazzanoFleetBindings(ctx context.Context, cluster *clusterv1.Cluster, verrazzanoFleetBindings []addonsv1alpha1.VerrazzanoFleetBinding) []addonsv1alpha1.VerrazzanoFleetBinding {
	log := ctrl.LoggerFrom(ctx)
	log.V(2).Info("Getting VerrazzanoFleetBindings to delete")

	selectedClusters := map[string]struct{}{}
	key := cluster.GetNamespace() + "/" + cluster.GetName()
	selectedClusters[key] = struct{}{}
	log.V(2).Info("Selected clusters", "clusters", selectedClusters)

	releasesToDelete := []addonsv1alpha1.VerrazzanoFleetBinding{}
	for _, verrazzanoFleetBinding := range verrazzanoFleetBindings {
		clusterRef := verrazzanoFleetBinding.Spec.ClusterRef
		key := clusterRef.Namespace + "/" + clusterRef.Name
		if _, ok := selectedClusters[key]; !ok {
			releasesToDelete = append(releasesToDelete, verrazzanoFleetBinding)
		}
	}

	names := make([]string, len(releasesToDelete))
	for _, release := range releasesToDelete {
		names = append(names, release.Name)
	}
	log.V(2).Info("Releases to delete", "releases", names)

	return releasesToDelete
}
