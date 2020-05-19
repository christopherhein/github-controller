/*
Copyright 2019 Christopher Hein <me@chrishein.com>.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KeySpec defines the desired state of Key
type KeySpec struct {
	// +optional
	// ReadOnly determines whether the key has write access to the repository
	ReadOnly bool `json:"readOnly"`

	// +kubebuilder:validation:MaxLength 253
	// RepositoryRef points to a Repository in the same Namespace that the Key is for
	RepositoryRef string `json:"repositoryRef"`

	// +optional
	// SecretTemplate sets annotations and labels on the resulting Secret of this Key
	SecretTemplate KeySecretTemplate `json:"secretTemplate,omitempty"`
}

// KeySecretTemplate is a template for creating Secrets that hold Key data
type KeySecretTemplate struct {
	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`

	// TargetNamespace optionally specifies the namespace the Key Secret is provisioned to
	// The default behavior results in a Secret namespace matching the metadata.namespace of the Key object
	// This can be used to achieve a "namespace delegation" pattern
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// NameOverride optionally specifies the name the Key Secret
	// The default behavior results in a Secret name matching metadata.name of the Key object
	// For example, this can be used in combination with `targetNamespace` to place multiple
	// Secrets of the same name into different Namespaces from the same managing Namespace.
	NameOverride string `json:"nameOverride,omitempty"`
}

// KeyStatus defines the observed state of Key
type KeyStatus struct {
	// +optional
	// Status stores the status of the Key
	Status StatusReason `json:"status,omitempty"`

	// +optional
	// URL stores the URL of the Key
	URL string `json:"url,omitempty"`

	// +optional
	// GitHubKeyID stores the GitHub API ID of the Key.
	// It is used to ensure deletion of the proper GitHub API Object.
	GitHubKeyID int64 `json:"gitHubKeyID,omitempty"`

	// +optional
	// GitHubRepository stores the current repository the key is applicable for.
	// It is used to ensure proper deletion in absence of a valid `KeySpec.RepositoryRef`.
	GitHubRepository string `json:"gitHubRepository,omitempty"`

	// +optional
	// GitHubOrganization stores the current organization expected to contain the applicable repository.
	// It is used to ensure proper deletion in absence of a valid `KeySpec.RepositoryRef`.
	GitHubOrganization string `json:"gitHubOrganization,omitempty"`

	// +optional
	// PublicKey holds the key contents matching the SSH private key.
	// It is used by the Key controller to track correctness of the child Secret object.
	PublicKey string `json:"publicKey"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=.status.status,description="Status of the Key",name=Status,priority=0,type=string

// Key is the Schema for the keys API
type Key struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeySpec   `json:"spec,omitempty"`
	Status KeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeyList contains a list of Key
type KeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Key `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Key{}, &KeyList{})
}
