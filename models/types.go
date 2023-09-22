// Copyright (c) 2023, Oracle and/or its affiliates.

package models

import (
	addonsv1alpha1 "github.com/verrazzano/cluster-api-addon-provider-verrazzano/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type HelmModuleAddons struct {
	// ChartLocation is the URL of the Helm chart repository.
	RepoURL string `json:"repoURL"`

	// ChartName is the name of the Helm chart in the repository.
	ChartName string `json:"chartName"`

	// ReleaseName is the release name of the installed Helm chart. If it is not specified, a name will be generated.
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// ReleaseNamespace is the namespace the Helm release will be installed on each selected
	// Cluster. If it is not specified, it will be set to the default namespace.
	// +optional
	ReleaseNamespace string `json:"namespace,omitempty"`

	// Version is the version of the Helm chart. If it is not specified, the chart will use
	// and be kept up to date with the latest version.
	// +optional
	Version string `json:"version,omitempty"`

	// valuesTemplate is an inline YAML representing the values for the Helm chart. This YAML supports Go templating to reference
	// fields from each selected workload Cluster and programatically create and set values.
	// +optional
	ValuesTemplate string `json:"valuesTemplate,omitempty"`

	// Local indicates whether chart is local or needs to be pulled in from the internet.
	// When Local is set to true the RepoURL would be the local path to the chart directory in the container.
	// By default, Local is set to false.
	// +optional
	Local bool `json:"local,omitempty"`

	// Options are HelmOptions and can be used in the future
	Options *HelmOptions `json:"options,omitempty"`
}

type HelmOptions struct {
	// DisableHooks prevents hooks from running during the Helm install action.
	// +optional
	DisableHooks bool `json:"disableHooks,omitempty"`

	// Wait enables the waiting for resources to be ready after a Helm install/upgrade has been performed.
	// +optional
	Wait bool `json:"wait,omitempty"`

	// WaitForJobs enables waiting for jobs to complete after a Helm install/upgrade has been performed.
	// +optional
	WaitForJobs bool `json:"waitForJobs,omitempty"`

	// DependencyUpdate indicates the Helm install/upgrade action to get missing dependencies.
	// +optional
	DependencyUpdate bool `json:"dependencyUpdate,omitempty"`

	// Timeout is the time to wait for any individual Kubernetes operation (like
	// resource creation, Jobs for hooks, etc.) during the performance of a Helm install action.
	// Defaults to '10 min'.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// SkipCRDs controls whether CRDs should be installed during install/upgrade operation.
	// By default, CRDs are installed if not already present.
	// If set, no CRDs will be installed.
	// +optional
	SkipCRDs bool `json:"skipCRDs,omitempty"`

	// SubNotes determines whether sub-notes should be rendered in the chart.
	// +optional
	SubNotes bool `json:"options,omitempty"`

	// DisableOpenAPIValidation controls whether OpenAPI validation is enforced.
	// +optional
	DisableOpenAPIValidation bool `json:"disableOpenAPIValidation,omitempty"`

	// Atomic indicates the installation/upgrade process to delete the installation or rollback on failure.
	// If 'Atomic' is set, wait will be enabled automatically during helm install/upgrade operation.
	// +optional
	Atomic bool `json:"atomic,omitempty"`

	// Install represents CLI flags passed to Helm install operation which can be used to control
	// behaviour of helm Install operations via options like wait, skipCrds, timeout, waitForJobs, etc.
	// +optional
	Install *HelmInstallOptions `json:"install,omitempty"`

	// Upgrade represents CLI flags passed to Helm upgrade operation which can be used to control
	// behaviour of helm Upgrade operations via options like wait, skipCrds, timeout, waitForJobs, etc.
	// +optional
	Upgrade *HelmUpgradeOptions `json:"upgrade,omitempty"`

	// Uninstall represents CLI flags passed to Helm uninstall operation which can be used to control
	// behaviour of helm Uninstall operation via options like wait, timeout, etc.
	// +optional
	Uninstall *HelmUninstallOptions `json:"uninstall,omitempty"`
}

type HelmInstallOptions struct {
	// CreateNamespace indicates the Helm install/upgrade action to create the
	// VerrazzanoFleetSpec.ReleaseNamespace if it does not exist yet.
	// On uninstall, the namespace will not be garbage collected.
	// If it is not specified by user, will be set to default 'true'.
	// +optional
	CreateNamespace *bool `json:"createNamespace,omitempty"`

	// IncludeCRDs determines whether CRDs stored as a part of helm templates directory should be installed.
	// +optional
	IncludeCRDs bool `json:"includeCRDs,omitempty"`
}

type HelmUpgradeOptions struct {
	// Force indicates to ignore certain warnings and perform the helm release upgrade anyway.
	// This should be used with caution.
	// +optional
	Force bool `json:"force,omitempty"`

	// ResetValues will reset the values to the chart's built-ins rather than merging with existing.
	// +optional
	ResetValues bool `json:"resetValues,omitempty"`

	// ReuseValues will re-use the user's last supplied values.
	// +optional
	ReuseValues bool `json:"reuseValues,omitempty"`

	// Recreate will (if true) recreate pods after a rollback.
	// +optional
	Recreate bool `json:"recreate,omitempty"`

	// MaxHistory limits the maximum number of revisions saved per release
	// +optional
	MaxHistory int `json:"maxHistory,omitempty"`

	// CleanupOnFail indicates the upgrade action to delete newly-created resources on a failed update operation.
	// +optional
	CleanupOnFail bool `json:"cleanupOnFail,omitempty"`
}

type HelmUninstallOptions struct {
	// KeepHistory defines whether historical revisions of a release should be saved.
	// If it's set, helm uninstall operation will not delete the history of the release.
	// The helm storage backend (secret, configmap, etc) will be retained in the cluster.
	// +optional
	KeepHistory bool `json:"keepHistory,omitempty"`

	// Description represents human readable information to be shown on release uninstall.
	// +optional
	Description string `json:"description,omitempty"`
}

type HelmValuesTemplate struct {
	Repository       string                      `json:"repository,omitempty"`
	Tag              string                      `json:"tag,omitempty"`
	PullPolicy       string                      `json:"pullPolicy,omitempty"`
	ImagePullSecrets []addonsv1alpha1.SecretName `json:"imagePullSecrets,omitempty"`
}

type Verrazzano struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Status struct {
		Available  string `json:"available"`
		Conditions []struct {
			LastTransitionTime time.Time `json:"lastTransitionTime"`
			Message            string    `json:"message"`
			Status             string    `json:"status"`
			Type               string    `json:"type"`
		} `json:"conditions"`
		State   string `json:"state"`
		Version string `json:"version"`
	} `json:"status"`
}
