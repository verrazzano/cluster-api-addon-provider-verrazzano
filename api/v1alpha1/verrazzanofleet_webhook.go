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
	"github.com/Masterminds/semver/v3"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/k8sutils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	spec                  = "spec"
	imageConfig           = "image"
	privateRegistryFlag   = "privateRegistry"
	imagePullSecretsArray = "imagePullSecrets"
	clusterSelector       = "clusterSelector"
	name                  = "name"
	verrazzano            = "verrazzano"
)

// log is for logging in this package.
var verrazzanofleetlog = logf.Log.WithName("verrazzanofleet-resource")

func (r *VerrazzanoFleet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-addons-cluster-x-k8s-io-v1alpha1-verrazzanofleet,mutating=true,failurePolicy=fail,sideEffects=None,groups=addons.cluster.x-k8s.io,resources=verrazzanofleets,verbs=create;update,versions=v1alpha1,name=verrazzanofleet.kb.io,admissionReviewVersions=v1
//+kubebuilder:webhook:path=/validate-addons-cluster-x-k8s-io-v1alpha1-verrazzanofleet,mutating=false,failurePolicy=fail,sideEffects=None,groups=addons.cluster.x-k8s.io,resources=verrazzanofleets,verbs=create;update,versions=v1alpha1,name=verrazzanofleet.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &VerrazzanoFleet{}
var _ webhook.Validator = &VerrazzanoFleet{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *VerrazzanoFleet) Default() {
	verrazzanofleetlog.Info("default", "name", in.Name)
	defaultVerrazzanoFleetSpec(&in.Spec)
}

func defaultVerrazzanoFleetSpec(vfs *VerrazzanoFleetSpec) {
	if vfs.Image != nil {
		if vfs.Image.PullPolicy == "" {
			vfs.Image.PullPolicy = "IfNotPresent"
		}
	}
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *VerrazzanoFleet) ValidateCreate() (admission.Warnings, error) {
	verrazzanofleetlog.Info("validate create", "name", in.Name)
	allErrs := validateFleet(in.Spec, field.NewPath("spec"))
	if len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("VerrazzanoFleet").GroupKind(), in.Name, allErrs)
	}
	return nil, nil
}

func validateFleet(s VerrazzanoFleetSpec, pathPrefix *field.Path) field.ErrorList {
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

	if s.ClusterSelector.Name == "" {
		allErrs = append(
			allErrs,
			field.Required(
				pathPrefix.Child("clusterSelector"),
				"a cluster name is required",
			),
		)
	}

	return allErrs
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *VerrazzanoFleet) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	verrazzanofleetlog.Info("validate update", "name", in.Name)

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
	allErrs := validateFleet(in.Spec, field.NewPath("spec"))

	prev, ok := old.(*VerrazzanoFleet)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expecting VerrazzanoFleet but got a %T", old))
	}

	isValid, err := isValidVerrazzanoUpgradeVersion(in.Spec)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	if !isValid {
		return nil, apierrors.NewBadRequest("Invalid version: Verrazzano version on the workload cluster can only be upgraded to match the Verrazzano version in the admin cluster.")
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
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("VerrazzanoFleet").GroupKind(), in.Name, allErrs)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *VerrazzanoFleet) ValidateDelete() (admission.Warnings, error) {
	verrazzanofleetlog.Info("validate delete", "name", r.Name)

	return nil, nil
}

func pathsMatch(allowed, path []string) bool {
	// if either are empty then no match can be made
	if len(allowed) == 0 || len(path) == 0 {
		return false
	}
	i := 0
	for i = range path {
		// reached the end of the allowed path and no match was found
		if i > len(allowed)-1 {
			return false
		}
		if allowed[i] == "*" {
			return true
		}
		if path[i] != allowed[i] {
			return false
		}
	}
	// path has been completely iterated and has not matched the end of the path.
	// e.g. allowed: []string{"a","b","c"}, path: []string{"a"}
	return i >= len(allowed)-1
}

func allowed(allowList [][]string, path []string) bool {
	for _, allowed := range allowList {
		if pathsMatch(allowed, path) {
			return true
		}
	}
	return false
}

// paths builds a slice of paths that are being modified.
func paths(path []string, diff map[string]interface{}) [][]string {
	allPaths := [][]string{}
	for key, m := range diff {
		nested, ok := m.(map[string]interface{})
		if !ok {
			// We have to use a copy of path, because otherwise the slice we append to
			// allPaths would be overwritten in another iteration.
			tmp := make([]string, len(path))
			copy(tmp, path)
			allPaths = append(allPaths, append(tmp, key))
			continue
		}
		allPaths = append(allPaths, paths(append(path, key), nested)...)
	}
	return allPaths
}

func isValidVerrazzanoUpgradeVersion(fleetSpec VerrazzanoFleetSpec) (bool, error) {

	verrazzanoSpec := fleetSpec.Verrazzano.Spec
	vzSpecObject, _ := ConvertRawExtensionToUnstructured(verrazzanoSpec)
	fleetVZVersion, versionExists, _ := unstructured.NestedString(vzSpecObject.Object, "version")
	vzVersionOnAdminCluster, err := k8sutils.GetVerrazzanoVersionOfAdminCluster()
	if err != nil {
		return false, apierrors.NewInternalError(err)
	}
	if vzVersionOnAdminCluster == "" && err != nil {
		return false, nil
	}
	if versionExists {
		fleetVZSemversion, err := semver.NewVersion(fleetVZVersion)
		if err != nil {
			return false, apierrors.NewInternalError(err)
		}
		chartRequestedSemversion, err := semver.NewVersion(vzVersionOnAdminCluster)
		if err != nil {
			return false, apierrors.NewInternalError(err)
		}
		if !fleetVZSemversion.Equal(chartRequestedSemversion) {
			return false, nil
		}
	}
	return true, nil
}

// ConvertRawExtensionToUnstructured converts a runtime.RawExtension to unstructured.Unstructured.
func ConvertRawExtensionToUnstructured(rawExtension *runtime.RawExtension) (*unstructured.Unstructured, error) {
	var obj runtime.Object
	var scope conversion.Scope
	if err := runtime.Convert_runtime_RawExtension_To_runtime_Object(rawExtension, &obj, scope); err != nil {
		return nil, err
	}

	innerObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: innerObj}, nil
}
