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

package verrazzanofleet

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	addonsv1alpha1 "github.com/verrazzano/cluster-api-addon-provider-verrazzano/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ reconcile.Reconciler = &VerrazzanoFleetReconciler{}

var (
	fakeVerrazzanoFleet1 = &addonsv1alpha1.VerrazzanoFleet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: addonsv1alpha1.GroupVersion.String(),
			Kind:       "VerrazzanoFleet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vf",
			Namespace: "test-namespace",
		},
		Spec: addonsv1alpha1.VerrazzanoFleetSpec{
			ClusterSelector: &addonsv1alpha1.ClusterName{
				Name: "test-cluster",
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
				addonsv1alpha1.SecretName{
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

	//fakeVerrazzanoFleet2 = &addonsv1alpha1.VerrazzanoFleet{
	//	TypeMeta: metav1.TypeMeta{
	//		APIVersion: addonsv1alpha1.GroupVersion.String(),
	//		Kind:       "VerrazzanoFleet",
	//	},
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      "test-vf",
	//		Namespace: "test-namespace",
	//	},
	//	Spec: addonsv1alpha1.VerrazzanoFleetSpec{
	//		ReleaseName:      "test-release-name",
	//		ChartName:        "test-chart-name",
	//		RepoURL:          "https://test-repo-url",
	//		ReleaseNamespace: "test-release-namespace",
	//		Version:          "test-version",
	//		ValuesTemplate:   "cidrBlockList: {{ .Cluster.spec.clusterNetwork.pods.cidrBlocks | join \",\" }}",
	//		Options:          &addonsv1alpha1.HelmOptions{},
	//	},
	//}

	//fakeInvalidVerrazzanoFleet = &addonsv1alpha1.VerrazzanoFleet{
	//	TypeMeta: metav1.TypeMeta{
	//		APIVersion: addonsv1alpha1.GroupVersion.String(),
	//		Kind:       "VerrazzanoFleet",
	//	},
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      "test-vf",
	//		Namespace: "test-namespace",
	//	},
	//	Spec: addonsv1alpha1.VerrazzanoFleetSpec{
	//		ReleaseName:      "test-release-name",
	//		ChartName:        "test-chart-name",
	//		RepoURL:          "https://test-repo-url",
	//		ReleaseNamespace: "test-release-namespace",
	//		Version:          "test-version",
	//		ValuesTemplate:   "apiServerPort: {{ .Cluster.invalid-path }}",
	//		Options:          &addonsv1alpha1.HelmOptions{},
	//	},
	//}

	fakeReinstallVerrazzanoFleet = &addonsv1alpha1.VerrazzanoFleet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: addonsv1alpha1.GroupVersion.String(),
			Kind:       "VerrazzanoFleet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vf",
			Namespace: "test-namespace",
		},
		Spec: addonsv1alpha1.VerrazzanoFleetSpec{
			ClusterSelector: &addonsv1alpha1.ClusterName{
				Name: "test-cluster",
			},
			Image: &addonsv1alpha1.ImageMeta{
				Repository: "ghcr.io",
				Tag:        "v0.1.1",
				PullPolicy: "Always",
			},
			PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
				Enabled: true,
			},
			ImagePullSecrets: []addonsv1alpha1.SecretName{
				addonsv1alpha1.SecretName{
					Name: "test-secret1",
				},
			},
			Verrazzano: &addonsv1alpha1.Verrazzano{
				Spec: &runtime.RawExtension{
					Object: nil,
				},
			},
		},
	}

	fakeCluster1 = &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterv1.GroupVersion.String(),
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-namespace",
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: &clusterv1.ClusterNetwork{
				APIServerPort: pointer.Int32(6443),
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"10.0.0.0/16", "20.0.0.0/16"},
				},
			},
		},
	}

	fakeCluster2 = &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterv1.GroupVersion.String(),
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-namespace",
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: &clusterv1.ClusterNetwork{
				APIServerPort: pointer.Int32(1234),
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"10.0.0.0/16", "20.0.0.0/16"},
				},
			},
		},
	}

	fakeVerrazzanoFleetBinding = &addonsv1alpha1.VerrazzanoFleetBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-generated-name",
			Namespace: "test-namespace",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         addonsv1alpha1.GroupVersion.String(),
					Kind:               "VerrazzanoFleet",
					Name:               "test-vf",
					Controller:         pointer.Bool(true),
					BlockOwnerDeletion: pointer.Bool(true),
				},
			},
			Labels: map[string]string{
				clusterv1.ClusterNameLabel:              "test-cluster",
				addonsv1alpha1.VerrazzanoFleetLabelName: "test-vf",
			},
		},
		Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
			ClusterRef: corev1.ObjectReference{
				APIVersion: clusterv1.GroupVersion.String(),
				Kind:       "Cluster",
				Name:       "test-cluster",
				Namespace:  "test-namespace",
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
				addonsv1alpha1.SecretName{
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
)

func TestReconcileForCluster(t *testing.T) {
	testcases := []struct {
		name                                string
		verrazzanoFleet                     *addonsv1alpha1.VerrazzanoFleet
		existingVerrazzanoFleetBinding      *addonsv1alpha1.VerrazzanoFleetBinding
		cluster                             *clusterv1.Cluster
		expect                              func(g *WithT, vf *addonsv1alpha1.VerrazzanoFleet, vfb *addonsv1alpha1.VerrazzanoFleetBinding)
		expectVerrazzanoFleetBindingToExist bool
		expectedError                       string
	}{
		{
			name:                                "creates a VerrazzanoFleetBinding for a VerrazzanoFleet",
			verrazzanoFleet:                     fakeVerrazzanoFleet1,
			cluster:                             fakeCluster1,
			expectVerrazzanoFleetBindingToExist: true,
			expect: func(g *WithT, vf *addonsv1alpha1.VerrazzanoFleet, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
				g.Expect(vfb.Spec.ClusterRef.Name).To(Equal("test-cluster"))

			},
			expectedError: "",
		},

		{
			name:                                "updates a VerrazzanoFleetBinding when Cluster value changes",
			verrazzanoFleet:                     fakeVerrazzanoFleet1,
			existingVerrazzanoFleetBinding:      fakeVerrazzanoFleetBinding,
			cluster:                             fakeCluster2,
			expectVerrazzanoFleetBindingToExist: true,
			expect: func(g *WithT, vf *addonsv1alpha1.VerrazzanoFleet, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
				g.Expect(vfb.Spec.ClusterRef.Name).To(Equal("test-cluster"))

			},
			expectedError: "",
		},
		//{
		//	name:                                "set condition for reinstalling when requeueing after a deletion",
		//	verrazzanoFleet:                     fakeReinstallVerrazzanoFleet,
		//	existingVerrazzanoFleetBinding:      fakeVerrazzanoFleetBinding,
		//	cluster:                             fakeCluster1,
		//	expectVerrazzanoFleetBindingToExist: false,
		//	expect: func(g *WithT, vf *addonsv1alpha1.VerrazzanoFleet, vfb *addonsv1alpha1.VerrazzanoFleetBinding) {
		//		g.Expect(conditions.Has(vf, addonsv1alpha1.VerrazzanoFleetBindingSpecsCreatedOrUpDatedCondition)).To(BeFalse())
		//		specsReady := conditions.Get(vf, addonsv1alpha1.VerrazzanoFleetBindingSpecsCreatedOrUpDatedCondition)
		//		//g.Expect(specsReady.Status).To(Equal(corev1.ConditionFalse))
		//		g.Expect(specsReady.Reason).To(Equal(addonsv1alpha1.VerrazzanoFleetBindingReinstallingReason))
		//		g.Expect(specsReady.Severity).To(Equal(clusterv1.ConditionSeverityInfo))
		//		g.Expect(specsReady.Message).To(Equal(fmt.Sprintf("VerrazzanoFleetBinding on cluster '%s' successfully deleted, preparing to reinstall", fakeCluster1.Name)))
		//	},
		//	expectedError: "",
		//},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			t.Parallel()

			objects := []client.Object{tc.verrazzanoFleet, tc.cluster}
			if tc.existingVerrazzanoFleetBinding != nil {
				objects = append(objects, tc.existingVerrazzanoFleetBinding)
			}
			r := &VerrazzanoFleetReconciler{
				Client: fake.NewClientBuilder().
					WithScheme(fakeScheme).
					WithObjects(objects...).
					WithStatusSubresource(&addonsv1alpha1.VerrazzanoFleet{}).
					WithStatusSubresource(&addonsv1alpha1.VerrazzanoFleetBinding{}).
					Build(),
			}
			err := r.reconcileForCluster(ctx, tc.verrazzanoFleet, tc.cluster)

			if tc.expectedError != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(tc.expectedError), err.Error())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				var vfb *addonsv1alpha1.VerrazzanoFleetBinding
				var err error
				if tc.expectVerrazzanoFleetBindingToExist {
					vfb, err = r.getExistingVerrazzanoFleetBinding(ctx, tc.verrazzanoFleet, tc.cluster)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(vfb).NotTo(BeNil())
				}
				tc.expect(g, tc.verrazzanoFleet, vfb)
			}
		})
	}
}

func TestConstructVerrazzanoFleetBinding(t *testing.T) {
	testCases := []struct {
		name            string
		existing        *addonsv1alpha1.VerrazzanoFleetBinding
		verrazzanoFleet *addonsv1alpha1.VerrazzanoFleet
		cluster         *clusterv1.Cluster
		expected        *addonsv1alpha1.VerrazzanoFleetBinding
	}{
		{
			name: "existing up to date, nothing to do",
			existing: &addonsv1alpha1.VerrazzanoFleetBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-generated-name",
					Namespace: "test-namespace",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         addonsv1alpha1.GroupVersion.String(),
							Kind:               "VerrazzanoFleet",
							Name:               "test-vf",
							Controller:         pointer.Bool(true),
							BlockOwnerDeletion: pointer.Bool(true),
						},
					},
					Labels: map[string]string{
						clusterv1.ClusterNameLabel:              "test-cluster",
						addonsv1alpha1.VerrazzanoFleetLabelName: "test-vf",
					},
				},
				Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
					ClusterRef: corev1.ObjectReference{
						APIVersion: clusterv1.GroupVersion.String(),
						Kind:       "Cluster",
						Name:       "test-cluster",
						Namespace:  "test-namespace",
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
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			verrazzanoFleet: &addonsv1alpha1.VerrazzanoFleet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: addonsv1alpha1.GroupVersion.String(),
					Kind:       "VerrazzanoFleet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vf",
					Namespace: "test-namespace",
				},
				Spec: addonsv1alpha1.VerrazzanoFleetSpec{
					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			cluster: &clusterv1.Cluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: clusterv1.GroupVersion.String(),
					Kind:       "Cluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
			},
			expected: nil,
		},
		{
			name:     "construct verrazzano fleet binding without existing",
			existing: nil,
			verrazzanoFleet: &addonsv1alpha1.VerrazzanoFleet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: addonsv1alpha1.GroupVersion.String(),
					Kind:       "VerrazzanoFleet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vf",
					Namespace: "test-namespace",
				},
				Spec: addonsv1alpha1.VerrazzanoFleetSpec{
					ClusterSelector: &addonsv1alpha1.ClusterName{
						Name: "test-cluster",
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
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			cluster: &clusterv1.Cluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: clusterv1.GroupVersion.String(),
					Kind:       "Cluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
				},
			},
			expected: &addonsv1alpha1.VerrazzanoFleetBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-namespace",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         addonsv1alpha1.GroupVersion.String(),
							Kind:               "VerrazzanoFleet",
							Name:               "test-vf",
							Controller:         pointer.Bool(true),
							BlockOwnerDeletion: pointer.Bool(true),
						},
					},
					Labels: map[string]string{
						clusterv1.ClusterNameLabel:              "test-cluster",
						addonsv1alpha1.VerrazzanoFleetLabelName: "test-vf",
					},
				},
				Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
					ClusterRef: corev1.ObjectReference{
						APIVersion: clusterv1.GroupVersion.String(),
						Kind:       "Cluster",
						Name:       "test-cluster",
						Namespace:  "test-namespace",
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
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			result := constructVerrazzanoFleetBinding(tc.existing, tc.verrazzanoFleet, tc.cluster)
			diff := cmp.Diff(tc.expected, result)
			g.Expect(diff).To(BeEmpty())
		})
	}
}

func TestShouldReinstallHelmRelease(t *testing.T) {
	testCases := []struct {
		name                   string
		verrazzanoFleetBinding *addonsv1alpha1.VerrazzanoFleetBinding
		verrazzanoFleet        *addonsv1alpha1.VerrazzanoFleet
		reinstall              bool
	}{
		{
			name: "nothing to do",
			verrazzanoFleetBinding: &addonsv1alpha1.VerrazzanoFleetBinding{
				Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			verrazzanoFleet: &addonsv1alpha1.VerrazzanoFleet{
				Spec: addonsv1alpha1.VerrazzanoFleetSpec{

					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			reinstall: false,
		},
		{
			name: "image repository has changed, should reinstall",
			verrazzanoFleetBinding: &addonsv1alpha1.VerrazzanoFleetBinding{
				Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			verrazzanoFleet: &addonsv1alpha1.VerrazzanoFleet{
				Spec: addonsv1alpha1.VerrazzanoFleetSpec{
					Image: &addonsv1alpha1.ImageMeta{
						Repository: "docker.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			reinstall: true,
		},
		{
			name: "image tag name changed, should reinstall",
			verrazzanoFleetBinding: &addonsv1alpha1.VerrazzanoFleetBinding{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						addonsv1alpha1.IsReleaseNameGeneratedAnnotation: "true",
					},
				},
				Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			verrazzanoFleet: &addonsv1alpha1.VerrazzanoFleet{
				Spec: addonsv1alpha1.VerrazzanoFleetSpec{
					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.2.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			reinstall: true,
		},
		{
			name: "image pull policy name unchanged, nothing to do",
			verrazzanoFleetBinding: &addonsv1alpha1.VerrazzanoFleetBinding{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						addonsv1alpha1.IsReleaseNameGeneratedAnnotation: "true",
					},
				},
				Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			verrazzanoFleet: &addonsv1alpha1.VerrazzanoFleet{
				Spec: addonsv1alpha1.VerrazzanoFleetSpec{
					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "IfNotPresent",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			reinstall: true,
		},
		{
			name: "private registry  changed, should reinstall",
			verrazzanoFleetBinding: &addonsv1alpha1.VerrazzanoFleetBinding{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						addonsv1alpha1.IsReleaseNameGeneratedAnnotation: "true",
					},
				},
				Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{

					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			verrazzanoFleet: &addonsv1alpha1.VerrazzanoFleet{
				Spec: addonsv1alpha1.VerrazzanoFleetSpec{

					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: false,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			reinstall: true,
		},
		{
			name: "image pull secret name changed, should reinstall",
			verrazzanoFleetBinding: &addonsv1alpha1.VerrazzanoFleetBinding{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						addonsv1alpha1.IsReleaseNameGeneratedAnnotation: "true",
					},
				},
				Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{

					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			verrazzanoFleet: &addonsv1alpha1.VerrazzanoFleet{
				Spec: addonsv1alpha1.VerrazzanoFleetSpec{

					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "another-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			reinstall: true,
		},
		{
			name: "verrazzano spec name changed, should reinstall",
			verrazzanoFleetBinding: &addonsv1alpha1.VerrazzanoFleetBinding{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						addonsv1alpha1.IsReleaseNameGeneratedAnnotation: "true",
					},
				},
				Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{

					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "test-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			verrazzanoFleet: &addonsv1alpha1.VerrazzanoFleet{
				Spec: addonsv1alpha1.VerrazzanoFleetSpec{

					Image: &addonsv1alpha1.ImageMeta{
						Repository: "ghcr.io",
						Tag:        "v0.1.0",
						PullPolicy: "Always",
					},
					PrivateRegistry: &addonsv1alpha1.PrivateRegistry{
						Enabled: true,
					},
					ImagePullSecrets: []addonsv1alpha1.SecretName{
						addonsv1alpha1.SecretName{
							Name: "another-secret",
						},
					},
					Verrazzano: &addonsv1alpha1.Verrazzano{
						Spec: &runtime.RawExtension{
							Object: nil,
						},
					},
				},
			},
			reinstall: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			result := shouldFleetBindingChange(ctx, tc.verrazzanoFleetBinding, tc.verrazzanoFleet)
			g.Expect(result).To(Equal(tc.reinstall))
		})
	}
}

func TestGetOrphanedVerrazzanoFleetBindings(t *testing.T) {
	testCases := []struct {
		name                    string
		selectedClusters        clusterv1.Cluster
		verrazzanoFleetBindings []addonsv1alpha1.VerrazzanoFleetBinding
		releasesToDelete        []addonsv1alpha1.VerrazzanoFleetBinding
	}{
		{
			name: "nothing to do",
			selectedClusters: clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1",
					Namespace: "test-namespace-1",
				},
			},
			verrazzanoFleetBindings: []addonsv1alpha1.VerrazzanoFleetBinding{
				{
					Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
						ClusterRef: corev1.ObjectReference{
							Name:      "test-cluster-1",
							Namespace: "test-namespace-1",
						},
					},
				},
			},
			releasesToDelete: []addonsv1alpha1.VerrazzanoFleetBinding{},
		},
		{
			name: "delete one release",
			selectedClusters: clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1",
					Namespace: "test-namespace-1",
				},
			},
			verrazzanoFleetBindings: []addonsv1alpha1.VerrazzanoFleetBinding{
				{
					Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
						ClusterRef: corev1.ObjectReference{
							Name:      "test-cluster-1",
							Namespace: "test-namespace-1",
						},
					},
				},
			},
			releasesToDelete: []addonsv1alpha1.VerrazzanoFleetBinding{},
		},
		{
			name:             "delete both releases",
			selectedClusters: clusterv1.Cluster{},
			verrazzanoFleetBindings: []addonsv1alpha1.VerrazzanoFleetBinding{
				{
					Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
						ClusterRef: corev1.ObjectReference{
							Name:      "test-cluster-1",
							Namespace: "test-namespace-1",
						},
					},
				},
			},
			releasesToDelete: []addonsv1alpha1.VerrazzanoFleetBinding{
				{
					Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
						ClusterRef: corev1.ObjectReference{
							Name:      "test-cluster-1",
							Namespace: "test-namespace-1",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			result := getOrphanedVerrazzanoFleetBindings(ctx, &tc.selectedClusters, tc.verrazzanoFleetBindings)
			g.Expect(result).To(Equal(tc.releasesToDelete))
		})
	}
}
