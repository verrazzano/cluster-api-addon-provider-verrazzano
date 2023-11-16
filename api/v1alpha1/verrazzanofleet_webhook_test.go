// Copyright (c) 2023, Oracle and/or its affiliates.

package v1alpha1

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/cluster-api-addon-provider-verrazzano/pkg/utils/k8sutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//func TestVerrazzanoFleetValidateCreate(t *testing.T) {
//	valid := &VerrazzanoFleet{
//		TypeMeta: metav1.TypeMeta{
//			APIVersion: GroupVersion.String(),
//			Kind:       "VerrazzanoFleet",
//		},
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "test-vf",
//			Namespace: "test-namespace",
//		},
//		Spec: VerrazzanoFleetSpec{
//			ClusterSelector: &ClusterName{
//				Name: "test-cluster",
//			},
//		},
//	}
//
//	validPrivateRegistry := valid.DeepCopy()
//	validPrivateRegistry.Spec.PrivateRegistry = &PrivateRegistry{
//		Enabled: true,
//	}
//	validPrivateRegistry.Spec.Image = &ImageMeta{
//		Repository: "docker.io",
//		Tag:        "v0.0.1",
//		PullPolicy: "Always",
//	}
//	validPrivateRegistry.Spec.Verrazzano = &Verrazzano{
//		Spec: &runtime.RawExtension{
//			Object: nil,
//		},
//	}
//
//	inValidPrivateRegistry1 := valid.DeepCopy()
//	inValidPrivateRegistry1.Spec.PrivateRegistry = &PrivateRegistry{
//		Enabled: true,
//	}
//
//	inValidPrivateRegistry2 := valid.DeepCopy()
//	inValidPrivateRegistry2.Spec.PrivateRegistry = &PrivateRegistry{
//		Enabled: true,
//	}
//	inValidPrivateRegistry2.Spec.Image = &ImageMeta{
//		Tag:        "v0.0.1",
//		PullPolicy: "Always",
//	}
//
//	inValidPrivateRegistry3 := valid.DeepCopy()
//	inValidPrivateRegistry3.Spec.PrivateRegistry = &PrivateRegistry{
//		Enabled: true,
//	}
//	inValidPrivateRegistry3.Spec.Image = &ImageMeta{
//		Repository: "docker.io",
//		PullPolicy: "Always",
//	}
//
//	inValidFleet := valid.DeepCopy()
//
//	tests := []struct {
//		name      string
//		expectErr bool
//		vzfleet   *VerrazzanoFleet
//	}{
//
//		{
//			name:      "should succeed when private registry is enabled",
//			expectErr: false,
//			vzfleet:   validPrivateRegistry,
//		},
//		{
//			name:      "should fail when private registry is enabled and image repo is missing",
//			expectErr: true,
//			vzfleet:   inValidPrivateRegistry1,
//		},
//		{
//			name:      "should fail when private registry is enabled and image tag is missing",
//			expectErr: true,
//			vzfleet:   inValidPrivateRegistry2,
//		},
//		{
//			name:      "should fail when private registry is disabled and image repo is provided",
//			expectErr: true,
//			vzfleet:   inValidPrivateRegistry3,
//		},
//
//		{
//			name:      "should fail when no verrazzano spec is provided",
//			expectErr: true,
//			vzfleet:   inValidFleet,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			g := NewWithT(t)
//			warnings, err := tt.vzfleet.ValidateCreate()
//			if tt.expectErr {
//				g.Expect(err).To(HaveOccurred())
//			} else {
//				g.Expect(err).NotTo(HaveOccurred())
//			}
//			g.Expect(warnings).To(BeEmpty())
//		})
//	}
//}

func TestVerrazzanoFleetValidateUpdate(t *testing.T) {
	before := &VerrazzanoFleet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "VerrazzanoFleet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vf",
			Namespace: "test-namespace",
		},
		Spec: VerrazzanoFleetSpec{
			ClusterSelector: &ClusterName{
				Name: "test-cluster",
			},
			Verrazzano: &Verrazzano{
				Spec: &runtime.RawExtension{
					Raw: []byte(`{"version": "v2.0.0", "profile": "none", "components": {"certManager": {"enabled": true}}}`),
				},
			},
			Image: &ImageMeta{
				Repository: "docker.io",
				Tag:        "v0.0.1",
				PullPolicy: "Always",
			},
			ImagePullSecrets: []SecretName{
				{
					Name: "test-secret",
				},
			},

			PrivateRegistry: &PrivateRegistry{
				Enabled: true,
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
			Raw: []byte(`{"version": "v2.0.0", "profile": "none", "components": {"certManager": {"enabled": true}}}`),
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

	clusterNameChangeFleet := before.DeepCopy()
	clusterNameChangeFleet.Spec.ClusterSelector.Name = "new-cluster"

	tests := []struct {
		name         string
		expectErr    bool
		before       *VerrazzanoFleet
		afterupgrade *VerrazzanoFleet
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
		{
			name:         "should pass when cluster name is changed",
			expectErr:    false,
			before:       before,
			afterupgrade: clusterNameChangeFleet,
		},
	}

	k8sutils.GetVerrazzanoVersionOfAdminClusterFunc = func() (string, error) {
		return "v2.0.0", nil
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
