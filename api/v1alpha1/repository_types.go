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

// RepositorySpec defines the desired state of Repository
type RepositorySpec struct {
	// +kubebuilder:validation:MaxLength 100
	// Organization is the name of the Github organization
	Organization string `json:"organization"`

	// +kubebuilder:validation:MaxLength 80
	// +optional
	// Description is the description of the repository
	Description string `json:"description,omitempty"`

	// +optional
	// Homepage is the location where documentation can be found
	Homepage string `json:"homepage,omitempty"`

	// +optional
	// Settings contains all the settings repository settings
	Settings RepositorySettings `json:"settings,omitempty"`
}

// RepositorySettings defines the desired settings
type RepositorySettings struct {
	// +optional
	// Private means it will create a private repo
	Private bool `json:"private,omitempty"`

	// +optional
	// Issues means the project has Github issues enabled
	Issues bool `json:"issues,omitempty"`

	// +optional
	// Projects means the project has Github projects enabled
	Projects bool `json:"projects,omitempty"`

	// +optional
	// Wiki means the project has Github wiki enabled
	Wiki bool `json:"wiki,omitempty"`

	// +optional
	// Template means the project is a template
	Template bool `json:"template,omitempty"`
}

// StatusReason returns the Status options
type StatusReason string

const (
	// SyncedStatus means the repository is in perfect synced status
	SyncedStatus StatusReason = "Synced"

	// CreatingStatus means the repository is in creating status
	CreatingStatus StatusReason = "Creating"

	// WaitingStatus means the repository is in waiting status
	WaitingStatus StatusReason = "Waiting"

	// UpdatingStatus means the repository is in updating status
	UpdatingStatus StatusReason = "Updating"

	// DeletingStatus means the repository is in deleting status
	DeletingStatus StatusReason = "Deleting"
)

// RepositoryStatus defines the observed state of Repository
type RepositoryStatus struct {
	// +optional
	// Status stores the status of the repository
	Status StatusReason `json:"status,omitempty"`

	// +optional
	// URL stores the URL of the repos
	URL string `json:"url,omitempty"`

	// +optional
	// ForkCount is the amount of forks when this was last synced
	ForkCount int `json:"forkCount,omitempty"`

	// +optional
	// StargazersCount is amount of stars when it was last synced
	StargazersCount int `json:"stargazersCount,omitempty"`

	// +optional
	// WatchersCount is amount of watchers when it was last synced
	WatchersCount int `json:"watchersCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=.status.status,description="Status of the Repository",name=Status,priority=0,type=string

// Repository is the Schema for the repositories API
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepositoryList contains a list of Repository
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
}
