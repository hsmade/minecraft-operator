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

// ServerSpec defines the desired state of Server
type ServerSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// HostPath is the path for the Server on the host
	HostPath string `json:"hostPath"`

	// Image is the docker image to run. It should have a shell, curl and, of course, java
	Image string `json:"image"`

	// Mods is a list of minecraft mods to be installed on the Server. Defaults to empty
	// +optional
	Mods []Mod `json:"mods,omitempty"`

	// Enabled defines if the Server should be running or not. Defaults to false
	Enabled bool `json:"enabled"`

	// Version is the minecraft version to run.
	Version string `json:"version"`

	// Flavor is the minecraft flavor to run. Valid values are:
	// - "vanilla"
	// - "spigot"
	// - "paper"
	// - "forge"
	Flavor Flavor `json:"flavor"`

	// Properties file settings
	Properties Properties `json:"properties"`

	// Max memory (Xmx), in MB
	MaxMemory int32 `json:"maxMemoryMB"`

	// Initial memory (Xms), in MB
	InitMemory int32 `json:"initMemoryMB"`

	// The site to get the server.jar from. It should have these in a directory named after the flavor, and the files
	// should be named server-<version>.jar. So for the vanilla 1.16.5 the path is:
	// <JarSite>/vanilla/server-1.16.5.jar
	JarSite string `json:"jarSite"`

	// HostPort defines the host port to bind to. Defaults to disabled
	// +optional
	HostPort int32 `json:"hostPort,omitempty"`

	// IdleTimeoutSeconds will, when set, disable the server after the server has been without users for the timeout period.
	// When it's not set (which is the default), it will not automatically disable the server, and it will keep running.
	// +optional
	IdleTimeoutSeconds int64 `json:"idleTimeoutSeconds"`
}

// Flavor describes the minecraft server flavor to be used.
// +kubebuilder:validation:Enum=vanilla;spigot;paper;forge
type Flavor string

const (
	Vanilla Flavor = "vanilla"
	Spigot  Flavor = "spigot"
	Paper   Flavor = "paper"
	Forge   Flavor = "forge"
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

// Properties defines the entries for server.properties that we support
type Properties struct {
	GameMode      GameMode   `json:"gamemode"`
	Difficulty    Difficulty `json:"difficulty"`
	SpawnMonsters bool       `json:"spawn-monsters"`
	SpawnNpcs     bool       `json:"spawn-npcs"`
	SpawnAnimals  bool       `json:"spawn-animals"`
	Motd          string     `json:"motd"`
}

// GameMode describes the minecraft server game mode to be used.
// +kubebuilder:validation:Enum=creative;survival;adventure
type GameMode string

const (
	Creative  GameMode = "creative"
	Survival  GameMode = "survival"
	Adventure GameMode = "adventure"
)

// Difficulty describes the minecraft server difficulty to be used.
// +kubebuilder:validation:Enum=peaceful;easy;normal;hard
type Difficulty string

const (
	Peaceful Difficulty = "peaceful"
	Easy     Difficulty = "easy"
	Normal   Difficulty = "normal"
	Hard     Difficulty = "hard"
)

// ServerStatus defines the observed state of Server
type ServerStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Running shows if the Server is running
	Running bool `json:"running"`

	// Thumbnail is base64 of the thumbnail image for the loaded world
	// +optional
	Thumbnail string `json:"thumbnail,omitempty"`

	// Players is the list of online players
	Players []string `json:"players"`

	//LastPong is the timestamp of the last checked pong
	LastPong int64 `json:"lastPong"`
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
