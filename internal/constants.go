// Copyright (c) 2023, Oracle and/or its affiliates.

package internal

const (
	// EnvVarKubeConfig Name of Environment Variable for KUBECONFIG
	EnvVarKubeConfig = "KUBECONFIG"

	// EnvVarTestKubeConfig Name of Environment Variable for test KUBECONFIG
	EnvVarTestKubeConfig = "TEST_KUBECONFIG"
	APIServerBurst       = 150
	APIServerQPS         = 100

	DefaultImagePullPolicy                           = "IfNotPresent"
	VerrazzanoPlatformOperatorRepo                   = "ghcr.io"
	VerrazzanoPlatformOperatorChartPath              = "/tmp/charts/verrazzano-platform-operator/"
	VerrazzanoPlatformOperatorChartName              = "verrazzano-platform-operator"
	VerrazzanoPlatformOperatorNameSpace              = "verrazzano-install"
	VerrazzanoPlatformOperatorHelmChartConfigMapName = "vpo-helm-chart"

	VERRAZZANO_UNINSTALL_TIMEOUT_MINUTES = "30m"

	// Min value used in WaitRandom
	Min = 10

	// Max value used in WaitRandom
	Max = 25

	//VerazzanoSpec constants
	VerrazzanoDomainKind       = "Verrazzano"
	VerrazzanoInstallName      = "verrazzano"
	VerrazzanoInstallNamespace = "default"
	APIGroup                   = "install.verrazzano.io"
	APIVersionBeta1            = "v1beta1"
	Spec                       = "spec"
	VerrazzanoResource         = "verrazzanos"
)
