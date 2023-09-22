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

package v1alpha1

import (
	"encoding/json"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch/v5"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var verrazzanofleetbindinglog = logf.Log.WithName("verrazzanofleetbinding-resource")

func (r *VerrazzanoFleetBinding) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-addons-cluster-x-k8s-io-v1alpha1-verrazzanofleetbinding,mutating=true,failurePolicy=fail,sideEffects=None,groups=addons.cluster.x-k8s.io,resources=verrazzanofleetbindings,verbs=create;update,versions=v1alpha1,name=verrazzanofleetbinding.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-addons-cluster-x-k8s-io-v1alpha1-verrazzanofleetbinding,mutating=false,failurePolicy=fail,sideEffects=None,groups=addons.cluster.x-k8s.io,resources=verrazzanofleetbindings,verbs=create;update,versions=v1alpha1,name=verrazzanofleetbinding.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &VerrazzanoFleetBinding{}
var _ webhook.Validator = &VerrazzanoFleetBinding{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *VerrazzanoFleetBinding) Default() {
	verrazzanofleetbindinglog.Info("default", "name", in.Name)
	defaultVerrazzanoFleetBindingSpec(&in.Spec)
}

func defaultVerrazzanoFleetBindingSpec(vfs *VerrazzanoFleetBindingSpec) {

	if vfs.Image != nil {
		if vfs.Image.PullPolicy == "" {
			vfs.Image.PullPolicy = "IfNotPresent"
		}
	}

}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *VerrazzanoFleetBinding) ValidateCreate() (admission.Warnings, error) {
	verrazzanofleetbindinglog.Info("validate create", "name", in.Name)

	verrazzanofleetlog.Info("validate create", "name", in.Name)
	allErrs := validateFleetBinding(in.Spec, in.GetNamespace(), field.NewPath("spec"))
	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("VerrazzanoFleet").GroupKind(), in.Name, allErrs)
	}
	return nil, nil
}

func validateFleetBinding(s VerrazzanoFleetBindingSpec, namespace string, pathPrefix *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if s.PrivateRegistry != nil {
		if s.PrivateRegistry.Enabled {
			if s.Image == nil {
				allErrs = append(
					allErrs,
					field.Required(
						pathPrefix.Child("image"),
						"image needs to be set if private registry is enabled",
					),
				)
			} else {
				if s.Image.Repository == "" {
					allErrs = append(
						allErrs,
						field.Required(
							pathPrefix.Child("image.repository"),
							"image repository cannot be empty if image is specified",
						),
					)
				}
				if s.Image.Tag == "" {
					allErrs = append(
						allErrs,
						field.Required(
							pathPrefix.Child("image.tag"),
							"image tag cannot be empty if image is specified",
						),
					)
				}

			}

		}
	}

	if s.Verrazzano != nil {
		if s.Verrazzano.Spec == nil {
			allErrs = append(
				allErrs,
				field.Required(
					pathPrefix.Child("verrazzano.spec"),
					"verrazano spec is required",
				),
			)
		}
	}

	if s.ClusterRef.Namespace != namespace {
		allErrs = append(
			allErrs,
			field.Required(
				pathPrefix.Child("clusterRef"),
				"cluster reference needs to point to cluster in same namespace as verrazzano binding",
			),
		)
	}
	if s.ClusterRef.APIVersion != "cluster.x-k8s.io/v1beta1" {
		allErrs = append(
			allErrs,
			field.Required(
				pathPrefix.Child("clusterRef"),
				"cluster reference api version needs to be of type cluster.x-k8s.io/v1beta1",
			),
		)
	}

	if s.ClusterRef.Kind != "Cluster" {
		allErrs = append(
			allErrs,
			field.Required(
				pathPrefix.Child("clusterRef"),
				"cluster reference kind needs to be of 'Cluster'",
			),
		)
	}

	return allErrs
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *VerrazzanoFleetBinding) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	verrazzanofleetbindinglog.Info("validate update", "name", in.Name)

	var allErrs field.ErrorList

	// add a * to indicate everything beneath is ok.
	// For example, {"spec", "*"} will allow any path under "spec" to change.
	allowedPaths := [][]string{
		{"metadata", "*"},
		{spec, imageConfig, "repository"},
		{spec, imageConfig, "pullPolicy"},
		{spec, imageConfig, "tag"},
		{spec, privateRegistryFlag, "enabled"},
		{spec, imagePullSecretsArray, "*"},
		{spec, verrazzano},
		{spec, verrazzano, spec},
		{spec, verrazzano, spec, "*"},
		{spec, clusterSelector},
		{spec, clusterSelector, name},
	}

	allErrs = validateFleetBinding(in.Spec, in.GetNamespace(), field.NewPath("spec"))

	prev, ok := old.(*VerrazzanoFleetBinding)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expecting VerrazzanoFleetBinding but got a %T", old))
	}

	originalJSON, err := json.Marshal(prev)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	modifiedJSON, err := json.Marshal(in)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	diff, err := jsonpatch.CreateMergePatch(originalJSON, modifiedJSON)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	jsonPatch := map[string]interface{}{}
	if err := json.Unmarshal(diff, &jsonPatch); err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	// Build a list of all paths that are trying to change
	diffpaths := paths([]string{}, jsonPatch)
	// Every path in the diff must be valid for the update function to work.
	for _, path := range diffpaths {
		// Ignore paths that are empty
		if len(path) == 0 {
			continue
		}
		if !allowed(allowedPaths, path) {
			if len(path) == 1 {
				allErrs = append(allErrs, field.Forbidden(field.NewPath(path[0]), "cannot be modified"))
				continue
			}
			allErrs = append(allErrs, field.Forbidden(field.NewPath(path[0], path[1:]...), "cannot be modified"))
		}
	}

	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("VerrazzanoFleetBinding").GroupKind(), in.Name, allErrs)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *VerrazzanoFleetBinding) ValidateDelete() (admission.Warnings, error) {
	verrazzanofleetbindinglog.Info("validate delete", "name", r.Name)
	return nil, nil
}
