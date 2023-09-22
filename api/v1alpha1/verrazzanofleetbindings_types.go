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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// VerrazzanoFleetBindingFinalizer is the finalizer used by the VerrazzanoFleetBinding controller to cleanup add-on resources when
	// a VerrazzanoFleetBinding is being deleted.
	VerrazzanoFleetBindingFinalizer = "verrazzanofleetbinding.addons.cluster.x-k8s.io"

	// VerrazzanoFleetLabelName is the label signifying which VerrazzanoFleet a VerrazzanoFleetBinding is associated with.
	VerrazzanoFleetLabelName = "verrazzanofleetbinding.addons.cluster.x-k8s.io/verrazzanofleet-name"

	// IsReleaseNameGeneratedAnnotation is the annotation signifying the Helm release name is auto-generated.
	IsReleaseNameGeneratedAnnotation = "verrazzanofleetbinding.addons.cluster.x-k8s.io/is-release-name-generated"
)

// VerrazzanoFleetBindingSpec defines the desired state of VerrazzanoFleetBinding.
type VerrazzanoFleetBindingSpec struct {
	// ClusterRef is a reference to the Cluster to install the Helm release on.
	ClusterRef corev1.ObjectReference `json:"clusterRef"`

	// Image is used to set various attributes regarding a specific module.
	// If not set, they are set as per the ImageMeta definitions.
	// +optional
	Image *ImageMeta `json:"image,omitempty"`

	// ImagePullSecrets allows to specify secrets if the image is being pulled from an authenticated private registry.
	// if not set, it will be assumed the images are public.
	// +optional
	ImagePullSecrets []SecretName `json:"imagePullSecrets,omitempty"`

	// Verrazzano is a verrazzano spec for installation on remote cluster.
	Verrazzano *Verrazzano `json:"verrazzano"`

	// PrivateRegistry sets the private registry settings for installing Verrazzano.
	// +optional
	PrivateRegistry *PrivateRegistry `json:"privateRegistry,omitempty"`
}

// VerrazzanoFleetBindingStatus defines the observed state of VerrazzanoFleetBinding.
type VerrazzanoFleetBindingStatus struct {
	// Conditions defines current state of the VerrazzanoFleetBinding.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// Status is the current status of the Helm release.
	// +optional
	Status string `json:"status,omitempty"`

	// Revision is the current revision of the Helm release.
	// +optional
	Revision int `json:"revision,omitempty"`

	// VerrazzanoStatus objects displays verrazzano specific status
	Verrazzano VerrazzanoStatus `json:"verrazzano,omitempty"`
}

type VerrazzanoStatus struct {
	// ComponentsAvailable displays the health of Verrazzano component
	ComponentsAvailable string `json:"componentsAvailable,omitempty"`

	// Version depicts the version of Verrazzano installed on the cluster.
	Version string `json:"version,omitempty"`

	// Stats depicts the state of  the Verrazzano installed on the cluster.
	State string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".spec.clusterRef.name",description="Cluster to which this VerrazzanoFleetBinding belongs"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",priority=1,JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Message",type="string",priority=1,JSONPath=".status.conditions[?(@.type=='Ready')].message"
// +kubebuilder:printcolumn:name="Operator",type="string",priority=1,JSONPath=".status.conditions[?(@.type=='VerrazzanoOperatorReady')].status"
// +kubebuilder:printcolumn:name="Revision",type="string",priority=1,JSONPath=".status.revision"
// +kubebuilder:printcolumn:name="Verrazzano",type="string",priority=1,JSONPath=".status.verrazzano.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of VerrazzanoFleet"
// +kubebuilder:resource:shortName=vfb;vfbs,scope=Namespaced,categories=cluster-api

// VerrazzanoFleetBinding is the Schema for the verrazzanofleetbindings API
type VerrazzanoFleetBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoFleetBindingSpec   `json:"spec,omitempty"`
	Status VerrazzanoFleetBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VerrazzanoFleetBindingList contains a list of VerrazzanoFleetBinding
type VerrazzanoFleetBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoFleetBinding `json:"items"`
}

// GetConditions returns the list of conditions for an VerrazzanoFleetBinding API object.
func (r *VerrazzanoFleetBinding) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions will set the given conditions on an VerrazzanoFleetBinding object.
func (r *VerrazzanoFleetBinding) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

// SetReleaseStatus will set the given status on an VerrazzanoFleetBinding object.
func (r *VerrazzanoFleetBinding) SetReleaseStatus(status string) {
	r.Status.Status = status // See pkg/release/status.go in Helm for possible values
	// r.Status.Status = release.Info.Status.String() // See pkg/release/status.go in Helm for possible values
}

// SetReleaseRevision will set the given revision on an VerrazzanoFleetBinding object.
func (r *VerrazzanoFleetBinding) SetReleaseRevision(version int) {
	r.Status.Revision = version
}

func init() {
	SchemeBuilder.Register(&VerrazzanoFleetBinding{}, &VerrazzanoFleetBindingList{})
}
