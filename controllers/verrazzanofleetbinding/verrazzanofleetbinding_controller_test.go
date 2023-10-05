/*
Copyright 2023 The Kubernetes Authors.

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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	addonsv1alpha1 "github.com/verrazzano/cluster-api-addon-provider-verrazzano/api/v1alpha1"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/internal"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/internal/mocks"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/constants"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/k8sutils"
	"go.uber.org/mock/gomock"
	helmRelease "helm.sh/helm/v3/pkg/release"
	helmDriver "helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfakedynamic "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	corev1Cli "k8s.io/client-go/kubernetes/typed/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

type v8ospec struct {
	Version string `json:"version"`
}

var (
	kubeconfig = "test-kubeconfig"

	defaultProxy = &addonsv1alpha1.VerrazzanoFleetBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VerrazzanoFleetBinding",
			APIVersion: "addons.cluster.x-k8s.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-proxy",
			Namespace: "default",
		},
		Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
			ClusterRef: corev1.ObjectReference{
				APIVersion: "cluster.x-k8s.io/v1beta1",
				Kind:       "Cluster",
				Namespace:  "default",
				Name:       "test-cluster",
			},
			Image: &addonsv1alpha1.ImageMeta{
				Repository: "ghcr.io",
				Tag:        "v0.1.0",
				PullPolicy: "Always",
			},
			PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
				Enabled: true,
			},
			ImagePullSecrets: []addonsv1alpha1.SecretName{
				{
					Name: "test-secret",
				},
			},
			Verrazzano: &addonsv1alpha1.Verrazzano{
				Spec: &runtime.RawExtension{
					Raw: []byte(`{"version": "v2.0.0", "profile": "none", "components": {"certManager": {"enabled": true}}}`),
				},
			},
		},
	}

	fleetBinding = &addonsv1alpha1.VerrazzanoFleetBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VerrazzanoFleetBinding",
			APIVersion: "addons.cluster.x-k8s.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-proxy",
			Namespace: "default",
		},
		Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
			ClusterRef: corev1.ObjectReference{
				APIVersion: "cluster.x-k8s.io/v1beta1",
				Kind:       "Cluster",
				Namespace:  "default",
				Name:       "test-cluster",
			},
			Image: &addonsv1alpha1.ImageMeta{
				Repository: "ghcr.io",
				Tag:        "v0.1.0",
				PullPolicy: "Always",
			},
			PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
				Enabled: true,
			},
			ImagePullSecrets: []addonsv1alpha1.SecretName{
				{
					Name: "test-secret",
				},
			},
			Verrazzano: &addonsv1alpha1.Verrazzano{
				Spec: &runtime.RawExtension{
					Object: nil,
				},
			},
		},
	}

	errInternal = fmt.Errorf("internal error")

	SchemeGroupVersion = schema.GroupVersion{Group: "install.verrazzano.io", Version: "v1beta1"}
	SchemeBuilder      = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	AddToScheme        = SchemeBuilder.AddToScheme
)

func newTestVZ() *unstructured.Unstructured {
	vz := &unstructured.Unstructured{
		Object: make(map[string]interface{}),
	}
	vz.SetAPIVersion(fmt.Sprintf("%s/%s", constants.APIGroup, constants.APIVersionBeta1))
	vz.SetKind(constants.VerrazzanoDomainKind)
	vz.SetName(constants.VerrazzanoInstallName)
	vz.SetNamespace(constants.VerrazzanoInstallNamespace)
	return vz
}

func generateVPOData() map[string]string {
	data := make(map[string]string)
	data[".helmignore"] = "Data"
	data["Chart.yaml"] = ""
	data["NOTES.txt"] = ""
	data["values.yaml"] = "image: ghcr.io/verrazzano/verrazzano-platform-operator:v2.0.0-20230927171927-9593a071"
	data["crds...install.verrazzano.io_verrazzanos.yaml"] = ""
	data["templates...clusterrole.yaml"] = ""
	data["templates...clusterrolebinding.yaml"] = ""
	data["templates...deployment.yaml"] = ""
	data["templates...mutatingWebHookConfiguration.yaml"] = ""
	data["templates...namespace.yaml"] = ""
	data["templates...service.yaml"] = ""
	data["templates...serviceaccount.yaml"] = ""
	data["templates...validatingwebhookconfiguration.yaml"] = ""

	//return &internal.HelmModuleAddons{
	//	ChartName:        internal.VerrazzanoPlatformOperatorChartName,
	//	ReleaseName:      internal.VerrazzanoPlatformOperatorChartName,
	//	ReleaseNamespace: internal.VerrazzanoPlatformOperatorNameSpace,
	//	RepoURL:          internal.VerrazzanoPlatformOperatorChartPath,
	//	Local:            true,
	//	ValuesTemplate:   string(out),
	//}, data
	return data

}

func TestReconcileNormal(t *testing.T) {
	var dynClient *k8sfakedynamic.FakeDynamicClient

	// Initialize scheme for all test cases
	scheme := runtime.NewScheme()
	_ = AddToScheme(scheme)

	var getWorkloadClusterK8sClientMock = func(g *WithT, c *mocks.MockClientMockRecorder) {
		c.GetWorkloadClusterK8sClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(k8sfake.NewSimpleClientset(), nil).Times(1)
	}

	var getWorkloadClusterDynamicK8sClientMock = func(g *WithT, c *mocks.MockClientMockRecorder) {
		c.GetWorkloadClusterDynamicK8sClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(dynClient, nil).Times(2)
	}

	testcases := []struct {
		name                   string
		verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding
		clientExpect           []func(g *WithT, c *mocks.MockClientMockRecorder)
		expect                 func(g *WithT, vfb *addonsv1alpha1.VerrazzanoFleetBinding)
		expectedError          string
	}{
		{
			name:                   "successfully install a Helm release",
			verrazzanoFleetBinding: defaultProxy.DeepCopy(),
			clientExpect: []func(g *WithT, c *mocks.MockClientMockRecorder){
				func(g *WithT, c *mocks.MockClientMockRecorder) {
					c.InstallOrUpgradeHelmRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&helmRelease.Release{
						Name:    "test-release",
						Version: 1,
						Info: &helmRelease.Info{
							Status: helmRelease.StatusDeployed,
						},
					}, nil).Times(1)
				},
				getWorkloadClusterK8sClientMock,
				getWorkloadClusterDynamicK8sClientMock,
			},
			expect: func(g *WithT, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
				g.Expect(vfb.Status.Revision).To(Equal(1))
				g.Expect(conditions.Has(vfb, addonsv1alpha1.VerrazzanoOperatorReadyCondition)).To(BeTrue())
				g.Expect(conditions.IsTrue(vfb, addonsv1alpha1.VerrazzanoOperatorReadyCondition)).To(BeTrue())

			},
			expectedError: "",
		},
		{
			name:                   "successfully install a Helm release with a generated name",
			verrazzanoFleetBinding: fleetBinding,
			clientExpect: []func(g *WithT, c *mocks.MockClientMockRecorder){
				func(g *WithT, c *mocks.MockClientMockRecorder) {
					c.InstallOrUpgradeHelmRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&helmRelease.Release{
						Name:    "test-release",
						Version: 1,
						Info: &helmRelease.Info{
							Status: helmRelease.StatusDeployed,
						},
					}, nil).Times(1)
				},
				getWorkloadClusterK8sClientMock,
			},
			expect: func(g *WithT, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
				_, ok := vfb.Annotations[addonsv1alpha1.IsReleaseNameGeneratedAnnotation]
				g.Expect(ok).To(BeTrue())
				g.Expect(vfb.Status.Revision).To(Equal(1))
				g.Expect(vfb.Status.Status).To(BeEquivalentTo(helmRelease.StatusDeployed))

				g.Expect(conditions.Has(vfb, addonsv1alpha1.HelmReleaseReadyCondition)).To(BeTrue())
				g.Expect(conditions.IsTrue(vfb, addonsv1alpha1.HelmReleaseReadyCondition)).To(BeTrue())
			},
			expectedError: "",
		},
		{
			name:                   "Helm release pending",
			verrazzanoFleetBinding: defaultProxy.DeepCopy(),
			clientExpect: []func(g *WithT, c *mocks.MockClientMockRecorder){
				func(g *WithT, c *mocks.MockClientMockRecorder) {
					c.InstallOrUpgradeHelmRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&helmRelease.Release{
						Name:    "test-release",
						Version: 1,
						Info: &helmRelease.Info{
							Status: helmRelease.StatusPendingInstall,
						},
					}, nil).Times(1)
				},
				getWorkloadClusterK8sClientMock,
			},
			expect: func(g *WithT, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
				t.Logf("VerrazzanoFleetBinding: %+v", vfb)
				_, ok := vfb.Annotations[addonsv1alpha1.IsReleaseNameGeneratedAnnotation]
				g.Expect(ok).To(BeFalse())
				g.Expect(vfb.Status.Revision).To(Equal(1))
				g.Expect(vfb.Status.Status).To(BeEquivalentTo(helmRelease.StatusPendingInstall))

				releaseReady := conditions.Get(vfb, addonsv1alpha1.HelmReleaseReadyCondition)
				g.Expect(releaseReady.Status).To(Equal(corev1.ConditionFalse))
				g.Expect(releaseReady.Reason).To(Equal(addonsv1alpha1.HelmReleasePendingReason))
				g.Expect(releaseReady.Severity).To(Equal(clusterv1.ConditionSeverityInfo))
			},
			expectedError: "",
		},
		{
			name:                   "Helm client returns error",
			verrazzanoFleetBinding: defaultProxy.DeepCopy(),
			clientExpect: []func(g *WithT, c *mocks.MockClientMockRecorder){
				func(g *WithT, c *mocks.MockClientMockRecorder) {
					c.InstallOrUpgradeHelmRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errInternal).Times(1)
				},
				getWorkloadClusterK8sClientMock,
			},
			expect: func(g *WithT, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
				_, ok := vfb.Annotations[addonsv1alpha1.IsReleaseNameGeneratedAnnotation]
				g.Expect(ok).To(BeFalse())

				releaseReady := conditions.Get(vfb, addonsv1alpha1.HelmReleaseReadyCondition)
				g.Expect(releaseReady.Status).To(Equal(corev1.ConditionFalse))
				g.Expect(releaseReady.Reason).To(Equal(addonsv1alpha1.HelmInstallOrUpgradeFailedReason))
				g.Expect(releaseReady.Severity).To(Equal(clusterv1.ConditionSeverityError))
				g.Expect(releaseReady.Message).To(Equal(errInternal.Error()))

			},
			expectedError: errInternal.Error(),
		},
		{
			name:                   "Helm release in a failed state, no client error",
			verrazzanoFleetBinding: defaultProxy.DeepCopy(),
			clientExpect: []func(g *WithT, c *mocks.MockClientMockRecorder){
				func(g *WithT, c *mocks.MockClientMockRecorder) {
					c.InstallOrUpgradeHelmRelease(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&helmRelease.Release{
						Name:    "test-release",
						Version: 1,
						Info: &helmRelease.Info{
							Status: helmRelease.StatusFailed,
						},
					}, nil).Times(1)
				},
				getWorkloadClusterK8sClientMock,
			},
			expect: func(g *WithT, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
				_, ok := vfb.Annotations[addonsv1alpha1.IsReleaseNameGeneratedAnnotation]
				g.Expect(ok).To(BeFalse())

				releaseReady := conditions.Get(vfb, addonsv1alpha1.HelmReleaseReadyCondition)
				g.Expect(releaseReady.Status).To(Equal(corev1.ConditionFalse))
				g.Expect(releaseReady.Reason).To(Equal(addonsv1alpha1.HelmInstallOrUpgradeFailedReason))
				g.Expect(releaseReady.Severity).To(Equal(clusterv1.ConditionSeverityError))
				g.Expect(releaseReady.Message).To(Equal(fmt.Sprintf("Helm release failed: %s", helmRelease.StatusFailed)))

			},
			expectedError: "",
		},
	}

	for _, tc := range testcases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			t.Parallel()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			clientMock := mocks.NewMockClient(mockCtrl)
			dynClient = k8sfakedynamic.NewSimpleDynamicClient(scheme, newTestVZ())

			internal.GetCoreV1Func = func() (corev1Cli.CoreV1Interface, error) {
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.VerrazzanoPlatformOperatorHelmChartConfigMapName,
						Namespace: constants.VerrazzanoPlatformOperatorNameSpace,
					},
					Data: generateVPOData(),
				}
				return k8sfake.NewSimpleClientset(configMap).CoreV1(), nil
			}
			defer func() { internal.GetCoreV1Func = k8sutils.GetCoreV1Client }()

			r := &VerrazzanoFleetBindingReconciler{
				Client: fake.NewClientBuilder().
					WithScheme(fakeScheme).
					WithStatusSubresource(&addonsv1alpha1.VerrazzanoFleetBinding{}).
					Build(),
			}

			for _, i := range tc.clientExpect {
				i(g, clientMock.EXPECT())
			}

			err := r.reconcileNormal(ctx, tc.verrazzanoFleetBinding, clientMock, kubeconfig)
			if tc.expectedError != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(tc.expectedError), err.Error())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				tc.expect(g, tc.verrazzanoFleetBinding)
			}
		})
	}
}

func TestReconcileDelete(t *testing.T) {
	testcases := []struct {
		name                   string
		verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding
		clientExpect           func(g *WithT, c *mocks.MockClientMockRecorder)
		expect                 func(g *WithT, vfb *addonsv1alpha1.VerrazzanoFleetBinding)
		expectedError          string
	}{
		{
			name:                   "succesfully uninstall a Helm release",
			verrazzanoFleetBinding: defaultProxy.DeepCopy(),
			clientExpect: func(g *WithT, c *mocks.MockClientMockRecorder) {
				c.GetHelmRelease(ctx, kubeconfig, defaultProxy.DeepCopy().Spec).Return(&helmRelease.Release{
					Name:    "test-release",
					Version: 1,
					Info: &helmRelease.Info{
						Status: helmRelease.StatusDeployed,
					},
				}, nil).Times(1)
				c.UninstallHelmRelease(ctx, kubeconfig, defaultProxy.DeepCopy().Spec).Return(&helmRelease.UninstallReleaseResponse{}, nil).Times(1)
			},
			expect: func(g *WithT, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
				g.Expect(conditions.Has(vfb, addonsv1alpha1.HelmReleaseReadyCondition)).To(BeTrue())
				releaseReady := conditions.Get(vfb, addonsv1alpha1.HelmReleaseReadyCondition)
				g.Expect(releaseReady.Status).To(Equal(corev1.ConditionFalse))
				g.Expect(releaseReady.Reason).To(Equal(addonsv1alpha1.HelmReleaseDeletedReason))
				g.Expect(releaseReady.Severity).To(Equal(clusterv1.ConditionSeverityInfo))
			},
			expectedError: "",
		},
		{
			name:                   "Helm release already uninstalled",
			verrazzanoFleetBinding: defaultProxy.DeepCopy(),
			clientExpect: func(g *WithT, c *mocks.MockClientMockRecorder) {
				c.GetHelmRelease(ctx, kubeconfig, defaultProxy.DeepCopy().Spec).Return(nil, helmDriver.ErrReleaseNotFound).Times(1)
			},
			expect: func(g *WithT, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
				g.Expect(conditions.Has(vfb, addonsv1alpha1.HelmReleaseReadyCondition)).To(BeTrue())
				releaseReady := conditions.Get(vfb, addonsv1alpha1.HelmReleaseReadyCondition)
				g.Expect(releaseReady.Status).To(Equal(corev1.ConditionFalse))
				g.Expect(releaseReady.Reason).To(Equal(addonsv1alpha1.HelmReleaseDeletedReason))
				g.Expect(releaseReady.Severity).To(Equal(clusterv1.ConditionSeverityInfo))
			},
			expectedError: "",
		},
		{
			name:                   "error attempting to get Helm release",
			verrazzanoFleetBinding: defaultProxy.DeepCopy(),
			clientExpect: func(g *WithT, c *mocks.MockClientMockRecorder) {
				c.GetHelmRelease(ctx, kubeconfig, defaultProxy.DeepCopy().Spec).Return(nil, errInternal).Times(1)
			},
			expect: func(g *WithT, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
				g.Expect(conditions.Has(vfb, addonsv1alpha1.HelmReleaseReadyCondition)).To(BeTrue())
				releaseReady := conditions.Get(vfb, addonsv1alpha1.HelmReleaseReadyCondition)
				g.Expect(releaseReady.Status).To(Equal(corev1.ConditionFalse))
				g.Expect(releaseReady.Reason).To(Equal(addonsv1alpha1.HelmReleaseReadyCondition))
				g.Expect(releaseReady.Severity).To(Equal(clusterv1.ConditionSeverityError))
			},
			expectedError: errInternal.Error(),
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			t.Parallel()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			clientMock := mocks.NewMockClient(mockCtrl)
			tc.clientExpect(g, clientMock.EXPECT())

			r := &VerrazzanoFleetBindingReconciler{
				Client: fake.NewClientBuilder().
					WithScheme(fakeScheme).
					WithStatusSubresource(&addonsv1alpha1.VerrazzanoFleetBinding{}).
					Build(),
			}

			err := r.reconcileDelete(ctx, tc.verrazzanoFleetBinding, clientMock, kubeconfig)
			if tc.expectedError != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(tc.expectedError), err.Error())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				tc.expect(g, tc.verrazzanoFleetBinding)
			}
		})
	}
}
