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

	// Image is the docker image to run.
	Image string `json:"image"`

	// ModJars is a list of minecraft mods to be installed on the Server. Defaults to empty
	// +optional
	ModJars []string `json:"mod-jars,omitempty"`

	// Enabled defines if the Server should be running or not. Defaults to false
	Enabled bool `json:"enabled"`

	// Properties file settings
	// +optional
	Properties map[string]string `json:"properties"`

	// Max memory (Xmx), in MB
	MaxMemory int32 `json:"maxMemoryMB"`

	// Initial memory (Xms), in MB
	InitMemory int32 `json:"initMemoryMB"`

	// The JAR file to run
	ServerJar string `json:"server-jar"`

	// HostPort defines the host port to bind to. Defaults to empty/disabled
	// +optional
	HostPort int32 `json:"hostPort"`

	// IdleTimeoutSeconds will, when set, disable the server after the server has been without users for the timeout period.
	// When it's not set (which is the default), it will not automatically disable the server, and it will keep running.
	// +optional
	IdleTimeoutSeconds int64 `json:"idleTimeoutSeconds,omitempty"`
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

	//LastPong is the timestamp of the last checked pong
	// +optional
	LastPong int64 `json:"lastPong,omitempty"`

	//IdleTime is the timestamp when we last saw players
	// +optional
	IdleTime int64 `json:"idleTime,omitempty"`
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
