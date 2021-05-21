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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServerSpec defines the desired state of Server
type ServerSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// HostPath is the path for the Server on the host
	HostPath string `json:"hostPath"`

	// WorldPath is the relative path to the world, inside the HostPath. Defaults to 'world'
	// +optional
	WorldPath string `json:"worldPath,omitempty"`

	// Mods is a list of minecraft mods to be installed on the Server. Defaults to empty
	// +optional
	Mods []Mod `json:"mods,omitempty"`

	// Enabled defines if the Server should be running or not. Defaults to false
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Version is the minecraft version to run. Defaults to latest
	// +optional
	Version string `json:"version,omitempty"`

	// Flavor is the minecraft flavor to run. Valid values are:
	// - "vanilla" (default)
	// - "spigot"
	// - "paper"
	// - "forge"
	// +optional
	Flavor Flavor `json:"flavor,omitempty"`

	// TODO: property file settings
}

// Flavor describes the minecraft server flavor to be used.
// If no Flavor is specified, the default one is VanillaFlavor
// +kubebuilder:validation:Enum=vanilla;spigot;paper;forge
type Flavor string

const (
	VanillaFlavor Flavor = "vanilla"
	SpigotFlavor  Flavor = "spigot"
	PaperFlavor   Flavor = "paper"
	ForgeFlavor   Flavor = "forge"
)

// Mod defines a minecraft mod to be installed on a Server
type Mod struct {
	// Name is the name of the mod
	Name string `json:"name"`

	// Version is the version of the mod
	Version string `json:"version"`

	// Url is the location where the mod's jar file can be found
	Url string `json:"url"`
}

// ServerStatus defines the observed state of Server
type ServerStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Running shows if the Server is running
	Running bool `json:"running"`

	// Thumbnail is base64 of the thumbnail image for the loaded world
	// +optional
	Thumbnail string `json:"thumbnail,omitempty"`

	// Players is the list of online players
	// +optional
	Players []string `json:"players,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Server is the Schema for the servers API
type Server struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServerSpec   `json:"spec,omitempty"`
	Status ServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServerList contains a list of Server
type ServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Server `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Server{}, &ServerList{})
}
