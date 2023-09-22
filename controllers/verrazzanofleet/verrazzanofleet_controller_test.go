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
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	. "github.com/onsi/gomega"
	addonsv1alpha1 "github.com/verrazzano/cluster-api-addon-provider-verrazzano/api/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ reconcile.Reconciler = &VerrazzanoFleetReconciler{}

var (
	defaultFleet = &addonsv1alpha1.VerrazzanoFleet{
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
				Name: "test-cluster-1",
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

	cluster1 = &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterv1.GroupVersion.String(),
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"test-label": "test-value",
			},
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: &clusterv1.ClusterNetwork{
				APIServerPort: pointer.Int32(1234),
			},
		},
	}

	cluster2 = &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterv1.GroupVersion.String(),
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-2",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"test-label":  "test-value",
				"other-label": "other-value",
			},
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: &clusterv1.ClusterNetwork{
				APIServerPort: pointer.Int32(5678),
			},
		},
	}

	cluster3 = &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterv1.GroupVersion.String(),
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-3",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"other-label": "other-value",
			},
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: &clusterv1.ClusterNetwork{
				APIServerPort: pointer.Int32(6443),
			},
		},
	}

	cluster4 = &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterv1.GroupVersion.String(),
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-4",
			Namespace: "other-namespace",
			Labels: map[string]string{
				"other-label": "other-value",
			},
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: &clusterv1.ClusterNetwork{
				APIServerPort: pointer.Int32(6443),
			},
		},
	}

	vfbReady1 = &addonsv1alpha1.VerrazzanoFleetBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-1",
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
				clusterv1.ClusterNameLabel:              "test-cluster-1",
				addonsv1alpha1.VerrazzanoFleetLabelName: "test-vf",
			},
		},
		Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
			ClusterRef: corev1.ObjectReference{
				APIVersion: clusterv1.GroupVersion.String(),
				Kind:       "Cluster",
				Name:       "test-cluster-1",
				Namespace:  "test-namespace",
			},
		},
		Status: addonsv1alpha1.VerrazzanoFleetBindingStatus{
			Conditions: []clusterv1.Condition{
				{
					Type:   clusterv1.ReadyCondition,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	vfbReady2 = &addonsv1alpha1.VerrazzanoFleetBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-2",
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
				clusterv1.ClusterNameLabel:              "test-cluster-2",
				addonsv1alpha1.VerrazzanoFleetLabelName: "test-vf",
			},
		},
		Spec: addonsv1alpha1.VerrazzanoFleetBindingSpec{
			ClusterRef: corev1.ObjectReference{
				APIVersion: clusterv1.GroupVersion.String(),
				Kind:       "Cluster",
				Name:       "test-cluster-2",
				Namespace:  "test-namespace",
			},
		},
		Status: addonsv1alpha1.VerrazzanoFleetBindingStatus{
			Conditions: []clusterv1.Condition{
				{
					Type:   clusterv1.ReadyCondition,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
)

func TestReconcileNormal(t *testing.T) {
	testcases := []struct {
		name            string
		verrazzanoFleet *addonsv1alpha1.VerrazzanoFleet
		objects         []client.Object
		expect          func(g *WithT, c client.Client, vf *addonsv1alpha1.VerrazzanoFleet)
		expectedError   string
	}{
		{
			name:            "successfully select clusters and install VerrazzanoFleetBindings",
			verrazzanoFleet: defaultFleet,
			objects:         []client.Object{cluster1, cluster2, cluster3, cluster4},
			expect: func(g *WithT, c client.Client, vf *addonsv1alpha1.VerrazzanoFleet) {
				g.Expect(conditions.Has(vf, addonsv1alpha1.VerrazzanoFleetBindingSpecsCreatedOrUpDatedCondition)).To(BeTrue())
				g.Expect(conditions.IsTrue(vf, addonsv1alpha1.VerrazzanoFleetBindingSpecsCreatedOrUpDatedCondition)).To(BeTrue())
				// This is false as the VerrazzanoFleetBindings won't be ready until the VerrazzanoFleetBinding controller runs.
				g.Expect(conditions.Has(vf, addonsv1alpha1.VerrazzanoFleetBindingsReadyCondition)).To(BeFalse())

			},
			expectedError: "",
		},
		{
			name:            "mark VerrazzanoFleet as ready once VerrazzanoFleetBindings ready conditions are true",
			verrazzanoFleet: defaultFleet,
			objects:         []client.Object{cluster1, cluster2, vfbReady1, vfbReady2},
			expect: func(g *WithT, c client.Client, vf *addonsv1alpha1.VerrazzanoFleet) {
				g.Expect(conditions.Has(vf, addonsv1alpha1.VerrazzanoFleetBindingSpecsCreatedOrUpDatedCondition)).To(BeTrue())
				g.Expect(conditions.IsTrue(vf, addonsv1alpha1.VerrazzanoFleetBindingSpecsCreatedOrUpDatedCondition)).To(BeTrue())
				g.Expect(conditions.Has(vf, addonsv1alpha1.VerrazzanoFleetBindingsReadyCondition)).To(BeTrue())
				g.Expect(conditions.IsTrue(vf, addonsv1alpha1.VerrazzanoFleetBindingsReadyCondition)).To(BeTrue())
				// This is marked as false since the Ready Logic has changed
				// Fleet will be Ready when FleetBinding is Ready
				// FleetBinding is ready only when Verrazzano install is Ready on workload cluster
				g.Expect(conditions.Has(vf, clusterv1.ReadyCondition)).To(BeFalse())
				g.Expect(conditions.IsTrue(vf, clusterv1.ReadyCondition)).To(BeFalse())
			},
			expectedError: "",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			t.Parallel()
			request := reconcile.Request{
				NamespacedName: util.ObjectKey(tc.verrazzanoFleet),
			}

			tc.objects = append(tc.objects, tc.verrazzanoFleet)
			r := &VerrazzanoFleetReconciler{
				Client: fake.NewClientBuilder().
					WithScheme(fakeScheme).
					WithObjects(tc.objects...).
					WithStatusSubresource(&addonsv1alpha1.VerrazzanoFleet{}).
					WithStatusSubresource(&addonsv1alpha1.VerrazzanoFleetBinding{}).
					Build(),
			}
			result, err := r.Reconcile(ctx, request)

			if tc.expectedError != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(tc.expectedError), err.Error())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(result).To(Equal(reconcile.Result{}))

				vf := &addonsv1alpha1.VerrazzanoFleet{}
				g.Expect(r.Client.Get(ctx, request.NamespacedName, vf)).To(Succeed())

				tc.expect(g, r.Client, vf)
			}
		})
	}
}
