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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// VerrazzanoFleetFinalizer is the finalizer used by the VerrazzanoFleet controller to cleanup add-on resources when
	// a VerrazzanoFleet is being deleted.
	VerrazzanoFleetFinalizer = "verrazzanofleet.addons.cluster.x-k8s.io"
)

// VerrazzanoFleetSpec defines the desired state of VerrazzanoFleet.
type VerrazzanoFleetSpec struct {
	// ClusterSelector selects a single Cluster in the same namespace with specified cluster name.
	ClusterSelector *ClusterName `json:"clusterSelector"`

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

type ImageMeta struct {
	// Repository sets the container registry to pull images from.
	// if not set, the Repository defined in OCNEMeta will be used instead.
	// +optional
	Repository string `json:"repository,omitempty"`

	// Tag allows to specify a tag for the image.
	// if not set, the Tag defined in OCNEMeta will be used instead.
	// +optional
	Tag string `json:"tag,omitempty"`

	// PullPolicy allows to specify an image pull policy for the container images.
	// if not set, the PullPolicy is IfNotPresent.
	// +optional
	PullPolicy string `json:"pullPolicy,omitempty"`
}

type ClusterName struct {
	// Name is name cluster where verrazzano will be installed
	// +optional
	Name string `json:"name,omitempty"`
}

type SecretName struct {
	// Name is name of the secret to be used as image pull secret
	// +optional
	Name string `json:"name,omitempty"`
}

type PrivateRegistry struct {
	// Enabled sets a flag to determine if a private registry will be used when installing Verrazzano.
	// if not set, the Enabled is set to false.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

type Verrazzano struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Spec *runtime.RawExtension `json:"spec"`
}

// VerrazzanoFleetStatus defines the observed state of VerrazzanoFleet.
type VerrazzanoFleetStatus struct {
	// Conditions defines current state of the VerrazzanoFleet.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Message",type="string",priority=1,JSONPath=".status.conditions[?(@.type=='Ready')].message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of VerrazzanoFleet"
// +kubebuilder:resource:shortName=vf;vfs,scope=Namespaced,categories=cluster-api

// VerrazzanoFleet is the Schema for the verrazzanofleets API
type VerrazzanoFleet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoFleetSpec   `json:"spec,omitempty"`
	Status VerrazzanoFleetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VerrazzanoFleetList contains a list of VerrazzanoFleet
type VerrazzanoFleetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerrazzanoFleet `json:"items"`
}

// GetConditions returns the list of conditions for an VerrazzanoFleet API object.
func (c *VerrazzanoFleet) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

// SetConditions will set the given conditions on an VerrazzanoFleet object.
func (c *VerrazzanoFleet) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&VerrazzanoFleet{}, &VerrazzanoFleetList{})
}
