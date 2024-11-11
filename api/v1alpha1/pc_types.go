// providerconfig_types.go

package v1alpha1

//go:generate angryjet generate-methodsets .

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProviderConfigSpec defines the configuration for Proxmox in the ProviderConfig.
type ProviderConfigSpec struct {
	Endpoint    string               `json:"endpoint"`    // Endpoint for the Proxmox API
	Credentials xpv1.SecretReference `json:"credentials"` // Credentials to connect to Proxmox
}

// ProviderConfigStatus represents connection or configuration status.
type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// ProviderConfig configures a Proxmox provider.
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProviderConfigSpec   `json:"spec"`
	Status            ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList is a list of ProviderConfigs.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
}
