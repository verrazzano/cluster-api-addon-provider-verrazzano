// Copyright (c) 2023, Oracle and/or its affiliates.

package v1alpha1

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/k8sutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestVerrazzanoFleetBindingValidateCreate(t *testing.T) {
	valid := &VerrazzanoFleetBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "VerrazzanoFleetBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vfb",
			Namespace: "test-namespace",
		},
		Spec: VerrazzanoFleetBindingSpec{
			ClusterRef: corev1.ObjectReference{
				Name:       "test-cluster",
				Kind:       "Cluster",
				APIVersion: "cluster.x-k8s.io/v1beta1",
				Namespace:  "test-namespace",
			},
		},
	}

	validPrivateRegistry := valid.DeepCopy()
	validPrivateRegistry.Spec.PrivateRegistry = &PrivateRegistry{
		Enabled: true,
	}
	validPrivateRegistry.Spec.Image = &ImageMeta{
		Repository: "docker.io",
		Tag:        "v0.0.1",
		PullPolicy: "Always",
	}
	validPrivateRegistry.Spec.Verrazzano = &Verrazzano{
		Spec: &runtime.RawExtension{
			Object: nil,
		},
	}

	inValidPrivateRegistry1 := valid.DeepCopy()
	inValidPrivateRegistry1.Spec.PrivateRegistry = &PrivateRegistry{
		Enabled: true,
	}

	inValidPrivateRegistry2 := valid.DeepCopy()
	inValidPrivateRegistry2.Spec.PrivateRegistry = &PrivateRegistry{
		Enabled: true,
	}
	inValidPrivateRegistry2.Spec.Image = &ImageMeta{
		Tag:        "v0.0.1",
		PullPolicy: "Always",
	}

	inValidPrivateRegistry3 := valid.DeepCopy()
	inValidPrivateRegistry3.Spec.PrivateRegistry = &PrivateRegistry{
		Enabled: true,
	}
	inValidPrivateRegistry3.Spec.Image = &ImageMeta{
		Repository: "docker.io",
		PullPolicy: "Always",
	}

	inValidFleet := valid.DeepCopy()
	inValidFleet.Spec.Verrazzano = &Verrazzano{
		Spec: nil,
	}

	tests := []struct {
		name      string
		expectErr bool
		vzfleet   *VerrazzanoFleetBinding
	}{

		{
			name:      "should succeed when private registry is enabled",
			expectErr: false,
			vzfleet:   validPrivateRegistry,
		},
		{
			name:      "should fail when private registry is enabled and image repo is missing",
			expectErr: true,
			vzfleet:   inValidPrivateRegistry1,
		},
		{
			name:      "should fail when private registry is enabled and image tag is missing",
			expectErr: true,
			vzfleet:   inValidPrivateRegistry2,
		},
		{
			name:      "should fail when private registry is disabled and image repo is provided",
			expectErr: true,
			vzfleet:   inValidPrivateRegistry3,
		},

		{
			name:      "should fail when no verrazzano spec is provided",
			expectErr: true,
			vzfleet:   inValidFleet,
		},
	}

	k8sutils.GetVerrazzanoVersionOfAdminClusterFunc = func() (string, error) {
		return "v1.7.0", nil
	}
	defer k8sutils.ResetGetVerrazzanoVersionOfAdminClusterFunc()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			warnings, err := tt.vzfleet.ValidateCreate()
			if tt.expectErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(warnings).To(BeEmpty())
		})
	}
}

func TestVerrazzanoFleetBindingValidateUpdate(t *testing.T) {
	before := &VerrazzanoFleetBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "VerrazzanoFleetBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vfb",
			Namespace: "test-namespace",
		},
		Spec: VerrazzanoFleetBindingSpec{
			Verrazzano: &Verrazzano{
				Spec: &runtime.RawExtension{
					Object: nil,
				},
			},

			Image: &ImageMeta{
				Repository: "docker.io",
				Tag:        "v0.0.1",
				PullPolicy: "Always",
			},
			ImagePullSecrets: []SecretName{
				SecretName{
					Name: "test-secret",
				},
			},
			PrivateRegistry: &PrivateRegistry{
				Enabled: true,
			},
			ClusterRef: corev1.ObjectReference{
				Name:       "test-cluster",
				Kind:       "Cluster",
				APIVersion: "cluster.x-k8s.io/v1beta1",
				Namespace:  "test-namespace",
			},
		},
	}

	validPrivateRegistry := before.DeepCopy()

	disablePrivateRegistry := before.DeepCopy()
	disablePrivateRegistry.Spec.PrivateRegistry = &PrivateRegistry{
		Enabled: false,
	}

	changeVerrazzanoSpec := before.DeepCopy()
	changeVerrazzanoSpec.Spec.Verrazzano = &Verrazzano{
		Spec: &runtime.RawExtension{
			Object: nil,
		},
	}

	changeImageRepo := before.DeepCopy()
	changeImageRepo.Spec.Image = &ImageMeta{
		Repository: "ghcr.io",
		Tag:        "v0.0.1",
		PullPolicy: "Always",
	}

	changeImageTag := before.DeepCopy()
	changeImageTag.Spec.Image = &ImageMeta{
		Repository: "ghcr.io",
		Tag:        "v0.0.2",
		PullPolicy: "Always",
	}

	changeImagePullPolicy := before.DeepCopy()
	changeImagePullPolicy.Spec.Image = &ImageMeta{
		Repository: "ghcr.io",
		Tag:        "v0.0.2",
		PullPolicy: "IfNotPresent",
	}

	inValidFleet := before.DeepCopy()
	inValidFleet.Spec.Verrazzano = &Verrazzano{
		Spec: nil,
	}

	tests := []struct {
		name         string
		expectErr    bool
		before       *VerrazzanoFleetBinding
		afterupgrade *VerrazzanoFleetBinding
	}{

		{
			name:         "should succeed when private registry is changed",
			expectErr:    false,
			before:       before,
			afterupgrade: validPrivateRegistry,
		},
		{
			name:         "should succeed with private registry disabled",
			expectErr:    false,
			before:       before,
			afterupgrade: disablePrivateRegistry,
		},
		{
			name:         "should succeed with verrazzano spec changed",
			expectErr:    false,
			before:       before,
			afterupgrade: changeVerrazzanoSpec,
		},
		{
			name:         "should succeed with image repository changed",
			expectErr:    false,
			before:       before,
			afterupgrade: changeImageRepo,
		},
		{
			name:         "should succeed with image tag changed",
			expectErr:    false,
			before:       before,
			afterupgrade: changeImageTag,
		},
		{
			name:         "should succeed with image pull policy changed",
			expectErr:    false,
			before:       before,
			afterupgrade: changeImagePullPolicy,
		},
		{
			name:         "should fail when verrazzano spec is removed",
			expectErr:    true,
			before:       before,
			afterupgrade: inValidFleet,
		},
	}

	k8sutils.GetVerrazzanoVersionOfAdminClusterFunc = func() (string, error) {
		return "v1.7.0", nil
	}
	defer k8sutils.ResetGetVerrazzanoVersionOfAdminClusterFunc()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			warnings, err := tt.afterupgrade.ValidateUpdate(tt.before.DeepCopy())
			if tt.expectErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			g.Expect(warnings).To(BeEmpty())
		})
	}
}
