/*
Copyright 2024.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ResourceNamespace struct {
	Namespace string              `json:"namespace,omitempty"`
	Resources map[string][]string `json:"resources,omitempty"`
}

// IndexSpec defines the desired state of Index
type IndexSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ServiceAccount string                       `json:"serviceAccount,omitempty"`
	NamespaceMap   map[string]ResourceNamespace `json:"namespaceMap,omitempty"`
}

// IndexStatus defines the observed state of Index
type IndexStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"` // Update accoridng to comparison of LastUpdated wrt other instances (propogation)
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Index is the Schema for the indices API
type Index struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IndexSpec   `json:"spec,omitempty"`
	Status IndexStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IndexList contains a list of Index
type IndexList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Index `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Index{}, &IndexList{})
}
