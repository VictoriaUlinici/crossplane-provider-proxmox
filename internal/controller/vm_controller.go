package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	proxmoxv1alpha1 "provider-proxmox/api/v1alpha1"
	"provider-proxmox/internal/proxmoxclient"
)

type VirtualMachineController struct{}

func (c *VirtualMachineController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(strings.ToLower(fmt.Sprintf("%s.%s", proxmoxv1alpha1.VirtualMachineKind, proxmoxv1alpha1.GroupVersion.Group))).
		For(&proxmoxv1alpha1.VirtualMachine{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(proxmoxv1alpha1.VirtualMachineGroupVersionKind),
			managed.WithExternalConnecter(&connecter{client: mgr.GetClient()}),
		))
}

type connecter struct {
	client client.Client
}

func (c *connecter) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	log := log.FromContext(ctx)
	vm, ok := mg.(*proxmoxv1alpha1.VirtualMachine)
	if !ok {
		return nil, errors.New("managed resource is not a VirtualMachine")
	}

	// Fetch ProviderConfig
	log.Info("Fetching ProviderConfig", "ProviderConfig", vm.Spec.ProviderConfigReference.Name)
	pc := &proxmoxv1alpha1.ProviderConfig{}
	pcName := types.NamespacedName{
		Name: vm.Spec.ProviderConfigReference.Name,
	}
	if err := c.client.Get(ctx, pcName, pc); err != nil {
		return nil, errors.Wrap(err, "cannot get ProviderConfig")
	}

	// Fetch credentials secret
	log.Info("Fetching credentials secret", "Namespace", pc.Spec.Credentials.Namespace, "Name", pc.Spec.Credentials.Name)
	creds := &corev1.Secret{}
	credsName := types.NamespacedName{
		Namespace: pc.Spec.Credentials.Namespace,
		Name:      pc.Spec.Credentials.Name,
	}
	if err := c.client.Get(ctx, credsName, creds); err != nil {
		return nil, errors.Wrap(err, "cannot get credentials secret")
	}

	username := string(creds.Data["username"])
	password := string(creds.Data["password"])

	log.Info("Creating Proxmox client")
	client, err := proxmoxclient.NewClientWithCredentials(pc.Spec.Endpoint, username, password)
	return &external{client: client, kube: c.client, log: log}, errors.Wrap(err, "cannot create Proxmox client")
}

type external struct {
	client *proxmoxclient.ProxmoxClient
	kube   client.Client //il client Kubernetes per aggiornare i finalizer
	log    logr.Logger
}

const finalizerName = "finalizer.proxmox.crossplane.io"

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	// Cast la risorsa gestita a VirtualMachine
	vm, ok := mg.(*proxmoxv1alpha1.VirtualMachine)
	if !ok {
		return managed.ExternalObservation{}, errors.New("managed resource is not a VirtualMachine")
	}

	// Usa il client Proxmox per ottenere lo stato attuale della VM
	existing, err := e.client.GetVMStatus(ctx, vm.Spec.VMID)
	if proxmoxclient.IsNotFound(err) {
		// La VM non esiste su Proxmox, il reconciler tenterà di crearla
		e.log.Info("VM non trovata su Proxmox; necessaria creazione", "VMID", vm.Spec.VMID)
		return managed.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: false,
		}, nil
	}
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "errore nel controllare lo stato della VM su Proxmox")
	}

	// La VM esiste. Aggiorna i campi di stato con i dati attuali della risorsa su Proxmox
	vm.Status.State = existing.Status
	vm.Status.Hostname = existing.Hostname
	vm.Status.ID = existing.ID

	// Imposta lo stato "Ready" della VM in base al suo stato attuale su Proxmox
	switch vm.Status.State {
	case proxmoxclient.StatusRunning:
		vm.SetConditions(xpv1.Available()) // Stato "Available" quando è in esecuzione
	case proxmoxclient.StatusCreating:
		vm.SetConditions(xpv1.Creating()) // Stato "Creating" durante la creazione
	case proxmoxclient.StatusDeleting:
		vm.SetConditions(xpv1.Deleting()) // Stato "Deleting" durante l'eliminazione
	default:
		vm.SetConditions(xpv1.Unavailable()) // Stato "Unavailable" se in stato sconosciuto
	}

	// Osserva se la configurazione è aggiornata confrontando lo stato attuale con quello desiderato
	isUpToDate := existing.ConfigurationMatches(vm.Spec)

	// Ritorna l'osservazione della risorsa senza innescare altre azioni
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate,
	}, nil
}

/*
// Empty Observe method for testing purposes
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	e.log.Info("Observe method temporarily disabled for testing Create")
	return managed.ExternalObservation{ResourceExists: false}, nil
}
*/

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	e.log.Info("Creating VirtualMachine resource in Proxmox")

	vm, ok := mg.(*proxmoxv1alpha1.VirtualMachine)
	if !ok {
		return managed.ExternalCreation{}, errors.New("managed resource is not a VirtualMachine")
	}

	e.log.Info("Preparing VM creation payload", "VMID", vm.Spec.VMID, "Name", vm.Spec.Name)

	// Prepare payload for Proxmox API request
	payload := map[string]interface{}{
		"vmid":    vm.Spec.VMID,
		"name":    vm.Spec.Name,
		"memory":  vm.Spec.Memory,
		"cores":   vm.Spec.Cores,
		"cpu":     vm.Spec.CPU,
		"sockets": vm.Spec.Sockets,
		"ide2":    vm.Spec.IDE2,
		"net0":    vm.Spec.Net0,
		"numa":    boolToProxmoxString(vm.Spec.Numa),
		"ostype":  vm.Spec.OSType,
		"scsi0":   vm.Spec.Scsi0,
		"scsihw":  vm.Spec.ScsiHW,
	}

	// Call Proxmox client to create VM
	err := e.client.Create(payload)
	if err == nil {
		e.log.Info("VM creation initiated successfully", "VMID", vm.Spec.VMID)
		vm.SetConditions(xpv1.Creating())
	} else {
		e.log.Error(err, "Failed to create VM")
	}
	return managed.ExternalCreation{}, errors.Wrap(err, "cannot create VM")
}

// Helper function to convert boolean to "0" or "1" for Proxmox
func boolToProxmoxString(val bool) string {
	if val {
		return "1"
	}
	return "0"
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	e.log.Info("Updating VirtualMachine resource in Proxmox")

	vm, ok := mg.(*proxmoxv1alpha1.VirtualMachine)
	if !ok {
		return managed.ExternalUpdate{}, errors.New("managed resource is not a VirtualMachine")
	}

	e.log.Info("Preparing VM update payload", "VMID", vm.Spec.VMID)
	payload := map[string]interface{}{
		"name":    vm.Spec.Name,
		"memory":  vm.Spec.Memory,
		"cores":   vm.Spec.Cores,
		"sockets": vm.Spec.Sockets,
	}
	err := e.client.Update(vm.Spec.VMID, payload)
	if err != nil {
		e.log.Error(err, "Failed to update VM", "VMID", vm.Spec.VMID)
	}
	return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update VM")
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	e.log.Info("Deleting VirtualMachine resource in Proxmox")

	vm, ok := mg.(*proxmoxv1alpha1.VirtualMachine)
	if !ok {
		return managed.ExternalDelete{}, errors.New("managed resource is not a VirtualMachine")
	}

	e.log.Info("Initiating VM deletion", "VMID", vm.Spec.VMID)
	err := e.client.Delete(vm.Spec.VMID)
	if proxmoxclient.IsNotFound(err) {
		e.log.Info("VM already deleted in Proxmox", "VMID", vm.Spec.VMID)
		// Rimuovi il finalizer se la VM è già cancellata
		RemoveFinalizer(vm, finalizerName)
		if err := e.kube.Update(ctx, vm); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, "failed to remove finalizer after VM deletion")
		}
		return managed.ExternalDelete{}, nil
	} else if err != nil {
		e.log.Error(err, "Failed to delete VM", "VMID", vm.Spec.VMID)
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete VM")
	}
	// Rimuovi il finalizer dopo cancellazione con successo
	e.log.Info("VM deletion successfully initiated", "VMID", vm.Spec.VMID)
	RemoveFinalizer(vm, finalizerName)
	if err := e.kube.Update(ctx, vm); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "failed to remove finalizer after deletion")
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(ctx context.Context) error {
	e.log.Info("Disconnecting from Proxmox API")
	return nil
}

// HasFinalizer checks if the given finalizer is present in the object's metadata.
func HasFinalizer(obj client.Object, finalizer string) bool {
	for _, f := range obj.GetFinalizers() {
		if f == finalizer {
			return true
		}
	}
	return false
}

// AddFinalizer adds a finalizer to the object's metadata if it doesn't already exist.
func AddFinalizer(obj client.Object, finalizer string) {
	if !HasFinalizer(obj, finalizer) {
		obj.SetFinalizers(append(obj.GetFinalizers(), finalizer))
	}
}

// RemoveFinalizer removes a finalizer from the object's metadata.
func RemoveFinalizer(obj client.Object, finalizer string) {
	finalizers := obj.GetFinalizers()
	newFinalizers := []string{}
	for _, f := range finalizers {
		if f != finalizer {
			newFinalizers = append(newFinalizers, f)
		}
	}
	obj.SetFinalizers(newFinalizers)
}
