/*
Copyright 2021.

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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OperatorConfigSpec defines the desired state of OperatorConfig
type OperatorConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ServerJarsPVC is the name of the PVC that holds the server JARs
	ServerJarsPVC string `json:"server-jars-pvc"`

	// ModJarsPVC is the name of the PVC that holds the mod JARs
	ModJarsPVC string `json:"mod-jars-pvc"`

	// ServersPV is the name of the PV that holds the PVCs for the Servers
	ServersPV *v1.PersistentVolume `json:"servers-pv"`

	// InitContainerImage is the name of the docker image to use for the init container. Defaults to busybox
	// +optional
	InitContainerImage string `json:"init-container-image"`
}

// OperatorConfigStatus defines the observed state of OperatorConfig
type OperatorConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OperatorConfig is the Schema for the operatorconfigs API
type OperatorConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatorConfigSpec   `json:"spec,omitempty"`
	Status OperatorConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OperatorConfigList contains a list of OperatorConfig
type OperatorConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OperatorConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OperatorConfig{}, &OperatorConfigList{})
}
