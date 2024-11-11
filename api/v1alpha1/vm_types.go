// Package v1alpha1 contains API Schema definitions for the Proxmox API
package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualMachineSpec defines the desired state of VirtualMachine.
type VirtualMachineSpec struct {
	ProviderConfigReference          *xpv1.Reference                  `json:"providerConfigReference"` // Link to provider configuration
	WriteConnectionSecretToReference *xpv1.SecretReference            `json:"writeConnectionSecretToReference,omitempty"`
	PublishConnectionDetailsTo       *xpv1.PublishConnectionDetailsTo `json:"publishConnectionDetailsTo,omitempty"`
	DeletionPolicy                   xpv1.DeletionPolicy              `json:"deletionPolicy,omitempty"`
	ManagementPolicies               xpv1.ManagementPolicies          `json:"managementPolicies,omitempty"`

	VMID    int    `json:"vmid"`    // Unique VM ID in Proxmox
	Name    string `json:"name"`    // Name of the virtual machine
	Memory  int    `json:"memory"`  // Memory size in MB
	Cores   int    `json:"cores"`   // Number of CPU cores
	CPU     string `json:"cpu"`     // CPU model type (e.g., x86-64-v2-AES)
	Sockets int    `json:"sockets"` // Number of CPU sockets
	IDE2    string `json:"ide2"`    // IDE configuration for CD-ROM, formatted as "<path>,media=cdrom"
	Net0    string `json:"net0"`    // Network configuration, e.g., "virtio,bridge=vmbr0"
	Numa    bool   `json:"numa"`    // Enable or disable NUMA (converted to 0 or 1 in payload)
	OSType  string `json:"ostype"`  // OS type (e.g., l26 for Linux)
	Scsi0   string `json:"scsi0"`   // Primary disk configuration
	ScsiHW  string `json:"scsihw"`  // SCSI hardware type

	// Proxmox VM specific configurations
	/*
		VMID          int    `json:"vmid"`                    // Unique VM ID in Proxmox
		Name          string `json:"name"`                    // Name of the VM
		Memory        int    `json:"memory"`                  // Memory in MB
		Cores         int    `json:"cores"`                   // CPU cores
		CPU           string `json:"cpu"`                     // CPU model (e.g., kvm64)
		Sockets       int    `json:"sockets"`                 // CPU sockets
		Numa          int    `json:"numa"`                    // Enable NUMA
		OSType        string `json:"ostype"`                  // OS type (e.g., l26)
		DiskConfig    string `json:"diskConfig,omitempty"`    // Disk config (e.g., virtio0:local-lvm,20G)
		ScsiHardware  string `json:"scsiHardware,omitempty"`  // SCSI hardware controller (e.g., virtio-scsi-single)
		AutoStart     bool   `json:"autoStart"`               // Enable auto-start on Proxmox boot
		IDE2          string `json:"ide2,omitempty"`          // CD-ROM IDE configuration (e.g., local:iso/ubuntu.iso)
		Net0          string `json:"net0"`                    // Network config (e.g., virtio,bridge=vmbr0)
		Scsi0         string `json:"scsi0,omitempty"`         // SCSI disk config (e.g., local-lvm:20G)
		ScsiHW        string `json:"scsihw,omitempty"`        // SCSI hardware type (e.g., virtio-scsi-pci)
		IsoPath       string `json:"isoPath,omitempty"`       // ISO path (optional)
		NetworkConfig string `json:"networkConfig,omitempty"` // Additional network config (if needed)

		// Optional Proxmox-specific fields
		BootDisk string `json:"bootDisk,omitempty"` // Boot disk (e.g., scsi0)
		DiskSize string `json:"diskSize,omitempty"` // Disk size (optional)
		CPUFlags string `json:"cpuFlags,omitempty"` // Optional CPU flags

	*/
}

// VirtualMachineStatus represents the observed state of the VM.
type VirtualMachineStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	State               string `json:"state,omitempty"`    // Current state of the VM
	Hostname            string `json:"hostname,omitempty"` // VM hostname
	ID                  int    `json:"id,omitempty"`       // VM ID
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VirtualMachine represents a Proxmox virtual machine
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec,omitempty"`
	Status VirtualMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualMachineList contains a list of VirtualMachine instances
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
}

// Crossplane Managed methods implementation

// GetCondition of this VirtualMachine.
func (vm *VirtualMachine) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return vm.Status.GetCondition(ct)
}

// SetConditions of this VirtualMachine.
func (vm *VirtualMachine) SetConditions(c ...xpv1.Condition) {
	vm.Status.SetConditions(c...)
}

// GetDeletionPolicy of this VirtualMachine.
func (vm *VirtualMachine) GetDeletionPolicy() xpv1.DeletionPolicy {
	return vm.Spec.DeletionPolicy
}

// SetDeletionPolicy of this VirtualMachine.
func (vm *VirtualMachine) SetDeletionPolicy(p xpv1.DeletionPolicy) {
	vm.Spec.DeletionPolicy = p
}

// GetManagementPolicies of this VirtualMachine.
func (vm *VirtualMachine) GetManagementPolicies() xpv1.ManagementPolicies {
	return vm.Spec.ManagementPolicies
}

// SetManagementPolicies of this VirtualMachine.
func (vm *VirtualMachine) SetManagementPolicies(p xpv1.ManagementPolicies) {
	vm.Spec.ManagementPolicies = p
}

// GetProviderConfigReference of this VirtualMachine.
func (vm *VirtualMachine) GetProviderConfigReference() *xpv1.Reference {
	return vm.Spec.ProviderConfigReference
}

// SetProviderConfigReference of this VirtualMachine.
func (vm *VirtualMachine) SetProviderConfigReference(r *xpv1.Reference) {
	vm.Spec.ProviderConfigReference = r
}

// GetPublishConnectionDetailsTo of this VirtualMachine.
func (vm *VirtualMachine) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return vm.Spec.PublishConnectionDetailsTo
}

// SetPublishConnectionDetailsTo of this VirtualMachine.
func (vm *VirtualMachine) SetPublishConnectionDetailsTo(r *xpv1.PublishConnectionDetailsTo) {
	vm.Spec.PublishConnectionDetailsTo = r
}

// GetWriteConnectionSecretToReference of this VirtualMachine.
func (vm *VirtualMachine) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return vm.Spec.WriteConnectionSecretToReference
}

// SetWriteConnectionSecretToReference of this VirtualMachine.
func (vm *VirtualMachine) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	vm.Spec.WriteConnectionSecretToReference = r
}
