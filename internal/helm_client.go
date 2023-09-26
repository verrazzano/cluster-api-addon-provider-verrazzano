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

package internal

import (
	"context"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	addonsv1alpha1 "github.com/verrazzano/cluster-api-addon-provider-verrazzano/api/v1alpha1"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/models"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/k8sutils"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/semver"
	helmAction "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	helmLoader "helm.sh/helm/v3/pkg/chart/loader"
	helmCli "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"net/url"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v2"
	helmVals "helm.sh/helm/v3/pkg/cli/values"
	helmGetter "helm.sh/helm/v3/pkg/getter"
	helmRelease "helm.sh/helm/v3/pkg/release"
	helmDriver "helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Client interface {
	InstallOrUpgradeHelmRelease(ctx context.Context, kubeconfig, values string, spec *models.HelmModuleAddons, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding) (*helmRelease.Release, error)
	GetHelmRelease(ctx context.Context, kubeconfig string, spec *models.HelmModuleAddons) (*helmRelease.Release, error)
	UninstallHelmRelease(ctx context.Context, kubeconfig string, spec *models.HelmModuleAddons) (*helmRelease.UninstallReleaseResponse, error)
}

type HelmClient struct{}

// GetActionConfig returns a new Helm action configuration.
func GetActionConfig(ctx context.Context, namespace string, config *rest.Config) (*helmAction.Configuration, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(4).Info("Getting action config")
	actionConfig := new(helmAction.Configuration)
	// var cliConfig *genericclioptions.ConfigFlags
	// cliConfig := &genericclioptions.ConfigFlags{
	// 	Namespace:        &env.namespace,
	// 	Context:          &env.KubeContext,
	// 	BearerToken:      &env.KubeToken,
	// 	APIServer:        &env.KubeAPIServer,
	// 	CAFile:           &env.KubeCaFile,
	// 	KubeConfig:       &env.KubeConfig,
	// 	Impersonate:      &env.KubeAsUser,
	// 	ImpersonateGroup: &env.KubeAsGroups,
	// }
	insecure := true
	cliConfig := genericclioptions.NewConfigFlags(false)
	cliConfig.APIServer = &config.Host
	cliConfig.BearerToken = &config.BearerToken
	cliConfig.Namespace = &namespace
	cliConfig.Insecure = &insecure
	// Drop their rest.Config and just return inject own
	wrapper := func(*rest.Config) *rest.Config {
		return config
	}
	cliConfig.WithWrapConfigFn(wrapper)
	// cliConfig.Insecure = &insecure
	// Note: can change this to klog.V(4) or use a debug level
	if err := actionConfig.Init(cliConfig, namespace, "secret", klog.V(4).Infof); err != nil {
		return nil, err
	}
	return actionConfig, nil
}

// HelmInit initializes Helm.
func HelmInit(ctx context.Context, namespace string, kubeconfig string) (*helmCli.EnvSettings, *helmAction.Configuration, error) {
	// log := ctrl.LoggerFrom(ctx)

	settings := helmCli.New()

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, nil, err
	}

	actionConfig, err := GetActionConfig(ctx, namespace, restConfig)
	if err != nil {
		return nil, nil, err
	}

	return settings, actionConfig, nil
}

// InstallOrUpgradeHelmRelease installs a Helm release if it does not exist, or upgrades it if it does and differs from the spec.
// It returns a boolean indicating whether an install or upgrade was performed.
func (c *HelmClient) InstallOrUpgradeHelmRelease(ctx context.Context, kubeconfig, values string, spec *models.HelmModuleAddons, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding) (*helmRelease.Release, error) {
	log := ctrl.LoggerFrom(ctx)

	log.V(2).Info("Installing or upgrading Helm release")

	// historyClient := helmAction.NewHistory(actionConfig)
	// historyClient.Max = 1
	// if _, err := historyClient.Run(spec.ReleaseName); err == helmDriver.ErrReleaseNotFound {
	existingRelease, err := c.GetHelmRelease(ctx, kubeconfig, spec)
	if err != nil {
		if err == helmDriver.ErrReleaseNotFound {
			return c.InstallHelmRelease(ctx, kubeconfig, values, spec)
		}

		return nil, err
	}

	return c.UpgradeHelmReleaseIfChanged(ctx, kubeconfig, values, spec, existingRelease, verrazzanoFleetBinding)
}

// generateHelmInstallConfig generates default helm install config using helmOptions specified in HCP CR spec.
func generateHelmInstallConfig(actionConfig *helmAction.Configuration, helmOptions *models.HelmOptions) *helmAction.Install {
	installClient := helmAction.NewInstall(actionConfig)
	installClient.CreateNamespace = true
	if helmOptions == nil {
		return installClient
	}

	installClient.DisableHooks = helmOptions.DisableHooks
	installClient.Wait = helmOptions.Wait
	installClient.WaitForJobs = helmOptions.WaitForJobs
	if helmOptions.Timeout != nil {
		installClient.Timeout = helmOptions.Timeout.Duration
	}
	installClient.SkipCRDs = helmOptions.SkipCRDs
	installClient.SubNotes = helmOptions.SubNotes
	installClient.DisableOpenAPIValidation = helmOptions.DisableOpenAPIValidation
	installClient.Atomic = helmOptions.Atomic

	if helmOptions.Install != nil {
		installClient.IncludeCRDs = helmOptions.Install.IncludeCRDs
		// Safety check to avoid panic in case webhook is disabled.
		if helmOptions.Install.CreateNamespace != nil {
			installClient.CreateNamespace = *helmOptions.Install.CreateNamespace
		}
	}

	return installClient
}

// generateHelmUpgradeConfig generates default helm upgrade config using helmOptions specified in HCP CR spec.
func generateHelmUpgradeConfig(actionConfig *helmAction.Configuration, helmOptions *models.HelmOptions) *helmAction.Upgrade {
	upgradeClient := helmAction.NewUpgrade(actionConfig)
	if helmOptions == nil {
		return upgradeClient
	}

	upgradeClient.DisableHooks = helmOptions.DisableHooks
	upgradeClient.Wait = helmOptions.Wait
	upgradeClient.WaitForJobs = helmOptions.WaitForJobs
	if helmOptions.Timeout != nil {
		upgradeClient.Timeout = helmOptions.Timeout.Duration
	}
	upgradeClient.SkipCRDs = helmOptions.SkipCRDs
	upgradeClient.SubNotes = helmOptions.SubNotes
	upgradeClient.DisableOpenAPIValidation = helmOptions.DisableOpenAPIValidation
	upgradeClient.Atomic = helmOptions.Atomic

	if helmOptions.Upgrade != nil {
		upgradeClient.Force = helmOptions.Upgrade.Force
		upgradeClient.ResetValues = helmOptions.Upgrade.ResetValues
		upgradeClient.ReuseValues = helmOptions.Upgrade.ReuseValues
		upgradeClient.MaxHistory = helmOptions.Upgrade.MaxHistory
		upgradeClient.CleanupOnFail = helmOptions.Upgrade.CleanupOnFail
	}

	return upgradeClient
}

// InstallHelmRelease installs a Helm release.
func (c *HelmClient) InstallHelmRelease(ctx context.Context, kubeconfig, values string, spec *models.HelmModuleAddons) (*helmRelease.Release, error) {
	log := ctrl.LoggerFrom(ctx)

	settings, actionConfig, err := HelmInit(ctx, spec.ReleaseNamespace, kubeconfig)
	if err != nil {
		return nil, err
	}

	registryClient, err := newDefaultRegistryClient()
	if err != nil {
		return nil, err
	}

	actionConfig.RegistryClient = registryClient

	chartName, repoURL, err := getHelmChartAndRepoName(spec.ChartName, spec.RepoURL)
	if err != nil {
		return nil, err
	}

	installClient := generateHelmInstallConfig(actionConfig, spec.Options)
	installClient.RepoURL = repoURL
	installClient.Version = spec.Version
	installClient.Namespace = spec.ReleaseNamespace

	if spec.ReleaseName == "" {
		installClient.GenerateName = true
		spec.ReleaseName, _, err = installClient.NameAndChart([]string{chartName})
		if err != nil {
			//return nil, errors.Wrapf(err, "failed to generate release name for chart %s on cluster %s", spec.ChartName, spec.ClusterRef.Name)
			return nil, errors.Wrapf(err, "failed to generate release name for chart %s", spec.ChartName)
		}
	}
	installClient.ReleaseName = spec.ReleaseName

	log.V(2).Info("Locating chart...")
	var cp string
	if spec.Local {
		cp = spec.RepoURL
	} else {
		cp, err = installClient.ChartPathOptions.LocateChart(chartName, settings)
		if err != nil {
			log.Error(err, "Unable to find chart")
			return nil, err
		}

	}

	log.V(2).Info("Located chart at path", "path", cp)

	log.V(2).Info("Writing values to file")
	filename, err := writeValuesToFile(ctx, values, spec)
	if err != nil {
		return nil, err
	}
	defer os.Remove(filename)
	log.V(2).Info("Values written to file", "path", filename)

	p := helmGetter.All(settings)
	valueOpts := &helmVals.Options{
		ValueFiles: []string{filename},
	}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return nil, err
	}
	chartRequested, err := helmLoader.Load(cp)

	if err != nil {
		return nil, err
	}
	log.V(1).Info("Installing with Helm", "chart", spec.ChartName, "repo", spec.RepoURL)

	return installClient.RunWithContext(ctx, chartRequested, vals) // Can return error and a release
}

// newDefaultRegistryClient creates registry client object with default config which can be used to install/upgrade helm charts.
func newDefaultRegistryClient() (*registry.Client, error) {
	// Create a new registry client
	registryClient, err := registry.NewClient(
		registry.ClientOptDebug(true),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stderr),
	)
	if err != nil {
		return nil, err
	}

	return registryClient, nil
}

// getHelmChartAndRepoName returns chartName, repoURL as per the format requirred in helm install/upgrade config.
// For OCI charts, chartName needs to have whole URL path. e.g. oci://repo-url/chart-name
func getHelmChartAndRepoName(chartName, repoURL string) (string, string, error) {
	if registry.IsOCI(repoURL) {
		u, err := url.Parse(repoURL)
		if err != nil {
			return "", "", err
		}

		u.Path = path.Join(u.Path, chartName)
		chartName = u.String()
		repoURL = ""
	}

	return chartName, repoURL, nil
}

// UpgradeHelmReleaseIfChanged upgrades a Helm release. The boolean refers to if an upgrade was attempted.
func (c *HelmClient) UpgradeHelmReleaseIfChanged(ctx context.Context, kubeconfig, values string, spec *models.HelmModuleAddons, existing *helmRelease.Release, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding) (*helmRelease.Release, error) {
	log := ctrl.LoggerFrom(ctx)

	settings, actionConfig, err := HelmInit(ctx, spec.ReleaseNamespace, kubeconfig)
	if err != nil {
		return nil, err
	}

	registryClient, err := newDefaultRegistryClient()
	if err != nil {
		return nil, err
	}

	actionConfig.RegistryClient = registryClient

	chartName, repoURL, err := getHelmChartAndRepoName(spec.ChartName, spec.RepoURL)
	if err != nil {
		return nil, err
	}

	upgradeClient := generateHelmUpgradeConfig(actionConfig, spec.Options)
	upgradeClient.RepoURL = repoURL
	upgradeClient.Version = spec.Version
	upgradeClient.Namespace = spec.ReleaseNamespace

	log.V(2).Info("Locating chart...")
	var cp string
	if spec.Local {
		cp = spec.RepoURL
	} else {
		cp, err = upgradeClient.ChartPathOptions.LocateChart(chartName, settings)
		if err != nil {
			klog.Error(err, "Unable to find chart")
			return nil, err
		}

	}
	log.V(2).Info("Located chart at path", "path", cp)

	log.V(2).Info("Writing values to file")
	filename, err := writeValuesToFile(ctx, values, spec)
	if err != nil {
		return nil, err
	}
	defer os.Remove(filename)
	log.V(2).Info("Values written to file", "path", filename)

	p := helmGetter.All(settings)
	valueOpts := &helmVals.Options{
		ValueFiles: []string{filename},
	}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return nil, err
	}
	chartRequested, err := helmLoader.Load(cp)
	if err != nil {
		return nil, err
	}
	if chartRequested == nil {
		return nil, errors.Errorf("failed to load request chart %s", chartName)
	}

	shouldUpgrade, err := shouldUpgradeHelmRelease(ctx, *existing, chartRequested, vals, verrazzanoFleetBinding)
	if err != nil {
		return nil, err
	}
	if !shouldUpgrade {
		log.V(2).Info(fmt.Sprintf("Release `%s` is up to date, no upgrade requried, revision = %d", existing.Name, existing.Version))
		return existing, nil
	}

	log.V(1).Info("Upgrading with Helm", "release", spec.ReleaseName, "repo", spec.RepoURL)
	release, err := upgradeClient.RunWithContext(ctx, spec.ReleaseName, chartRequested, vals)

	return release, err
	// Should we force upgrade if it failed previously?
}

// writeValuesToFile writes the Helm values to a temporary file.
func writeValuesToFile(ctx context.Context, values string, spec *models.HelmModuleAddons) (string, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(2).Info("Writing values to file")
	valuesFile, err := os.CreateTemp("", spec.ChartName+"-"+spec.ReleaseName+"-*.yaml")
	if err != nil {
		return "", err
	}

	if _, err := valuesFile.Write([]byte(values)); err != nil {
		return "", err
	}

	return valuesFile.Name(), nil
}

// shouldUpgradeHelmRelease determines if a Helm release should be upgraded.
func shouldUpgradeHelmRelease(ctx context.Context, existing helmRelease.Release, chartRequested *chart.Chart, values map[string]interface{}, verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding) (bool, error) {
	log := ctrl.LoggerFrom(ctx)
	existing.Info.Status.String()
	var vzSemVersionWorkloadCluster *semver.SemVersion

	verrazzanoSpec := verrazzanoFleetBinding.Spec.Verrazzano.Spec
	vzSpecObject, err := utils.ConvertRawExtensionToUnstructured(verrazzanoSpec)
	if err != nil {
		log.Error(err, "Failed to convert raw extension to unstructured data")
		return false, err
	}
	fleetVZVersion, versionExists, _ := unstructured.NestedString(vzSpecObject.Object, "version")
	if !versionExists {
		log.V(2).Info("'version' field not found in verrazzanoSpec.")
	}
	if fleetVZVersion == "" {
		log.V(2).Info("version is same or not specified, skipping upgrade...")
		return false, nil
	}

	if verrazzanoFleetBinding.Status.Verrazzano != (addonsv1alpha1.VerrazzanoStatus{}) {
		vzVersionWorkloadCluster := verrazzanoFleetBinding.Status.Verrazzano.Version
		vzSemVersionWorkloadCluster, err = semver.NewSemVersion(vzVersionWorkloadCluster)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to parse verrazzano version on workload cluster")
		}
	}

	if existing.Chart == nil || existing.Chart.Metadata == nil {
		return false, errors.New("Failed to resolve chart version of existing release")
	}

	vzVersionOnAdminCluster, err := k8sutils.GetVerrazzanoVersionOfAdminCluster()
	if err != nil {
		return false, errors.Wrapf(err, "Failed to get Verrazzano version from admin cluster")
	}
	if vzVersionOnAdminCluster == "" && err != nil {
		return false, nil
	}
	vzSemVersionAdminCluster, err := semver.NewSemVersion(vzVersionOnAdminCluster)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to parse verrazzano version on admin cluster")
	}

	fleetVZSemversion, err := semver.NewSemVersion(fleetVZVersion)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to parse fleet binding verrazzano version")
	}
	if !fleetVZSemversion.IsEqualTo(vzSemVersionAdminCluster) {
		log.V(2).Info("Verrazzano version on the workload cluster can only be upgraded to the Verrazzano version in the admin cluster, skipping upgrade...", "flelet", fleetVZSemversion, "admin", vzSemVersionAdminCluster)
		return false, nil
	}

	// Helm chart versions do not have version sha, therefore this condition handles upgrade for same release version but different commit
	// This is useful for development testing.
	if existing.Chart.Metadata.Version == chartRequested.Metadata.Version && !vzSemVersionWorkloadCluster.IsEqualTo(vzSemVersionAdminCluster) {
		log.V(3).Info("Versions are different, upgrading")
		return true, nil
	}

	fleetVZVersionParts := strings.Split(fleetVZVersion, "-")
	// Checks existing release and requested metadata versions
	// Also, ensures latest VPO is not installed when vz version in the FleetBinding spec does not match admin cluster verrazzano version
	if (existing.Chart.Metadata.Version != chartRequested.Metadata.Version) && (existing.Chart.Metadata.Version != fleetVZVersionParts[0]) {
		log.V(2).Info("Versions are different, upgrading")
		return true, nil
	}

	if existing.Info.Status == helmRelease.StatusFailed {
		log.Info("Release is in failed state, attempting upgrade to fix it")
		return true, nil
	}

	klog.V(2).Infof("Diff between values is:\n%s", cmp.Diff(existing.Config, values))

	// TODO: Comparing yaml is not ideal, but it's the best we can do since DeepEquals fails. This is because int64 types
	// are converted to float64 when returned from the helm API.
	oldValues, err := yaml.Marshal(existing.Config)
	if err != nil {
		return false, errors.Wrapf(err, "failed to marshal existing release values")
	}
	newValues, err := yaml.Marshal(values)
	if err != nil {
		return false, errors.Wrapf(err, "failed to new release values")
	}

	return !cmp.Equal(oldValues, newValues), nil
}

// GetHelmRelease returns a Helm release if it exists.
func (c *HelmClient) GetHelmRelease(ctx context.Context, kubeconfig string, spec *models.HelmModuleAddons) (*helmRelease.Release, error) {
	if spec.ReleaseName == "" {
		return nil, helmDriver.ErrReleaseNotFound
	}

	_, actionConfig, err := HelmInit(ctx, spec.ReleaseNamespace, kubeconfig)
	if err != nil {
		return nil, err
	}
	getClient := helmAction.NewGet(actionConfig)
	release, err := getClient.Run(spec.ReleaseName)
	if err != nil {
		return nil, err
	}

	return release, nil
}

// ListHelmReleases lists all Helm releases in a namespace.
func (c *HelmClient) ListHelmReleases(ctx context.Context, kubeconfig string, spec *models.HelmModuleAddons) ([]*helmRelease.Release, error) {
	_, actionConfig, err := HelmInit(ctx, spec.ReleaseNamespace, kubeconfig)
	if err != nil {
		return nil, err
	}
	listClient := helmAction.NewList(actionConfig)
	releases, err := listClient.Run()
	if err != nil {
		return nil, err
	}

	return releases, nil
}

// generateHelmUninstallConfig generates default helm uninstall config using helmOptions specified in HCP CR spec.
func generateHelmUninstallConfig(actionConfig *helmAction.Configuration, helmOptions *models.HelmOptions) *helmAction.Uninstall {
	uninstallClient := helmAction.NewUninstall(actionConfig)
	if helmOptions == nil {
		return uninstallClient
	}

	uninstallClient.DisableHooks = helmOptions.DisableHooks
	uninstallClient.Wait = helmOptions.Wait
	if helmOptions.Timeout != nil {
		uninstallClient.Timeout = helmOptions.Timeout.Duration
	}

	if helmOptions.Uninstall != nil {
		uninstallClient.KeepHistory = helmOptions.Uninstall.KeepHistory
		uninstallClient.Description = helmOptions.Uninstall.Description
	}

	return uninstallClient
}

// UninstallHelmRelease uninstalls a Helm release.
func (c *HelmClient) UninstallHelmRelease(ctx context.Context, kubeconfig string, spec *models.HelmModuleAddons) (*helmRelease.UninstallReleaseResponse, error) {
	_, actionConfig, err := HelmInit(ctx, spec.ReleaseNamespace, kubeconfig)
	if err != nil {
		return nil, err
	}

	uninstallClient := generateHelmUninstallConfig(actionConfig, spec.Options)

	response, err := uninstallClient.Run(spec.ReleaseName)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// RollbackHelmRelease rolls back a Helm release.
func (c *HelmClient) RollbackHelmRelease(ctx context.Context, kubeconfig string, spec *models.HelmModuleAddons) error {
	_, actionConfig, err := HelmInit(ctx, spec.ReleaseNamespace, kubeconfig)
	if err != nil {
		return err
	}

	rollbackClient := helmAction.NewRollback(actionConfig)
	return rollbackClient.Run(spec.ReleaseName)
}
