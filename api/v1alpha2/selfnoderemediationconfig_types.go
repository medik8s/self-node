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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	ConfigCRName                          = "self-node-remediation-config"
	defaultWatchdogPath                   = "/dev/watchdog"
	defaultSafetToAssumeNodeRebootTimeout = 180
	defaultIsSoftwareRebootEnabled        = true
)

// SelfNodeRemediationConfigSpec defines the desired state of SelfNodeRemediationConfig
type SelfNodeRemediationConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// WatchdogFilePath is the watchdog file path that should be available on each node, e.g. /dev/watchdog
	// +kubebuilder:default=/dev/watchdog
	WatchdogFilePath string `json:"watchdogFilePath,omitempty"`

	// SafeTimeToAssumeNodeRebootedSeconds is the time after which the healthy self node remediation
	// agents will assume the unhealthy node has been rebooted and it is safe to remove the node
	// from the cluster. This is extremely important. Deleting a node while the workload is still
	// running there might lead to data corruption and violation of run-once semantic.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=180
	SafeTimeToAssumeNodeRebootedSeconds int `json:"safeTimeToAssumeNodeRebootedSeconds,omitempty"`

	// Valid time units are "ms", "s", "m", "h".
	// +optional
	// +kubebuilder:default:="5s"
	// +kubebuilder:validation:Pattern="^(0|([0-9]+(\\.[0-9]+)?(ms|s|m|h)))$"
	// +kubebuilder:validation:Type:=string
	PeerApiServerTimeout *metav1.Duration `json:"peerApiServerTimeout,omitempty"`

	// the frequency for api-server connectivity check
	// Valid time units are "ms", "s", "m", "h".
	// +optional
	// +kubebuilder:default:="15s"
	// +kubebuilder:validation:Pattern="^(0|([0-9]+(\\.[0-9]+)?(ms|s|m|h)))$"
	// +kubebuilder:validation:Type:=string
	// the frequency for api-server connectivity check
	ApiCheckInterval *metav1.Duration `json:"apiCheckInterval,omitempty"`

	// Valid time units are "ms", "s", "m", "h".
	// +optional
	// +kubebuilder:default:="15m"
	// +kubebuilder:validation:Pattern="^(0|([0-9]+(\\.[0-9]+)?(ms|s|m|h)))$"
	// +kubebuilder:validation:Type:=string
	PeerUpdateInterval *metav1.Duration `json:"peerUpdateInterval,omitempty"`

	// Valid time units are "ms", "s", "m", "h".
	// +optional
	// +kubebuilder:default:="5s"
	// +kubebuilder:validation:Pattern="^(0|([0-9]+(\\.[0-9]+)?(ms|s|m|h)))$"
	// +kubebuilder:validation:Type:=string
	// timeout for each api-connectivity check
	ApiServerTimeout *metav1.Duration `json:"apiServerTimeout,omitempty"`

	// Valid time units are "ms", "s", "m", "h".
	// +optional
	// +kubebuilder:default:="5s"
	// +kubebuilder:validation:Pattern="^(0|([0-9]+(\\.[0-9]+)?(ms|s|m|h)))$"
	// +kubebuilder:validation:Type:=string
	// timeout for establishing connection to peer
	PeerDialTimeout *metav1.Duration `json:"peerDialTimeout,omitempty"`

	// Valid time units are "ms", "s", "m", "h".
	// +optional
	// +kubebuilder:default:="5s"
	// +kubebuilder:validation:Pattern="^(0|([0-9]+(\\.[0-9]+)?(ms|s|m|h)))$"
	// +kubebuilder:validation:Type:=string
	// timeout for each peer request
	PeerRequestTimeout *metav1.Duration `json:"peerRequestTimeout,omitempty"`

	// +optional
	// +kubebuilder:default:=3
	// +kubebuilder:validation:Minimum=1
	// after this threshold, the node will start contacting its peers
	MaxApiErrorThreshold int `json:"maxApiErrorThreshold,omitempty"`

	// IsSoftwareRebootEnabled indicates whether self node remediation agent will do software reboot,
	// if the watchdog device can not be used or will use watchdog only,
	// without a fallback to software reboot
	// +kubebuilder:default=true
	IsSoftwareRebootEnabled bool `json:"isSoftwareRebootEnabled,omitempty"`
}

// SelfNodeRemediationConfigStatus defines the observed state of SelfNodeRemediationConfig
type SelfNodeRemediationConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=snrc;snrconfig
//+kubebuilder:storageversion

// SelfNodeRemediationConfig is the Schema for the selfnoderemediationconfigs API in which a user can configure the self node remediation agents
// +operator-sdk:csv:customresourcedefinitions:resources={{"SelfNodeRemediationConfig","v1alpha2","selfnoderemediationconfigs"}}
type SelfNodeRemediationConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SelfNodeRemediationConfigSpec   `json:"spec,omitempty"`
	Status SelfNodeRemediationConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SelfNodeRemediationConfigList contains a list of SelfNodeRemediationConfig
type SelfNodeRemediationConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SelfNodeRemediationConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SelfNodeRemediationConfig{}, &SelfNodeRemediationConfigList{})
}

func NewDefaultSelfNodeRemediationConfig() SelfNodeRemediationConfig {
	return SelfNodeRemediationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: ConfigCRName},
		Spec: SelfNodeRemediationConfigSpec{
			WatchdogFilePath:                    defaultWatchdogPath,
			SafeTimeToAssumeNodeRebootedSeconds: defaultSafetToAssumeNodeRebootTimeout,
			IsSoftwareRebootEnabled:             defaultIsSoftwareRebootEnabled,
		},
	}
}
