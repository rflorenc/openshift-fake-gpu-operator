/*
Copyright 2026.

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

type FakeGPUConfigSpec struct {
	// +kubebuilder:validation:Enum=a100;h100;h200;b200;gb200;l40s;t4
	// +optional
	GPUProfile string `json:"gpuProfile,omitempty"`

	// +kubebuilder:default=8
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=32
	GPUCount int32 `json:"gpuCount,omitempty"`

	// +optional
	Custom *CustomGPUSpec `json:"custom,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +optional
	MIG *MIGSpec `json:"mig,omitempty"`

	// +optional
	Components *ComponentsSpec `json:"components,omitempty"`

	// +optional
	Images *ImagesSpec `json:"images,omitempty"`
}

type ImagesSpec struct {
	// +optional
	Registry string `json:"registry,omitempty"`

	// +optional
	Tag string `json:"tag,omitempty"`

	// +optional
	Overrides map[string]string `json:"overrides,omitempty"`
}

type CustomGPUSpec struct {
	GPUProduct string `json:"gpuProduct"`

	// +kubebuilder:validation:Minimum=1
	GPUMemory int32 `json:"gpuMemory"`
}

type MIGSpec struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// +kubebuilder:default=mixed
	// +kubebuilder:validation:Enum=mixed;single
	Strategy string `json:"strategy,omitempty"`

	// +optional
	Devices []MIGDevice `json:"devices,omitempty"`
}

type MIGDevice struct {
	Name  string `json:"name"`
	Count int32  `json:"count"`
}

type ComponentsSpec struct {
	// +kubebuilder:default=true
	DevicePlugin bool `json:"devicePlugin,omitempty"`

	// +kubebuilder:default=true
	MetricsExporter bool `json:"metricsExporter,omitempty"`

	// +kubebuilder:default=true
	TopologyServer bool `json:"topologyServer,omitempty"`

	// +kubebuilder:default=true
	StatusUpdater bool `json:"statusUpdater,omitempty"`

	// +kubebuilder:default=false
	DRA bool `json:"dra,omitempty"`

	// +kubebuilder:default=false
	ComputeDomainController bool `json:"computeDomainController,omitempty"`
}

type FakeGPUConfigStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	ReadyNodes int32 `json:"readyNodes,omitempty"`

	TotalNodes int32 `json:"totalNodes,omitempty"`

	// +optional
	GPUProfile string `json:"gpuProfile,omitempty"`

	GPUCountPerNode int32 `json:"gpuCountPerNode,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Profile",type=string,JSONPath=`.spec.gpuProfile`
// +kubebuilder:printcolumn:name="GPUs",type=integer,JSONPath=`.spec.gpuCount`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyNodes`
// +kubebuilder:printcolumn:name="Total",type=integer,JSONPath=`.status.totalNodes`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

type FakeGPUConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FakeGPUConfigSpec   `json:"spec,omitempty"`
	Status FakeGPUConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type FakeGPUConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FakeGPUConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FakeGPUConfig{}, &FakeGPUConfigList{})
}
