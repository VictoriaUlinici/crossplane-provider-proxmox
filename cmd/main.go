package main

import (
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	proxmoxapis "provider-proxmox/api/v1alpha1"
	proxmoxcontroller "provider-proxmox/internal/controller"

	crossplaneapis "github.com/crossplane/crossplane/apis"
)

func main() {
	log.SetLogger(zap.New(zap.UseDevMode(true)))

	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		panic(err)
	}

	// Register Crossplane and Proxmox API schemes
	if err := crossplaneapis.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}
	if err := proxmoxapis.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	// Setup the Proxmox controller with the manager
	vmcontroller := &proxmoxcontroller.VirtualMachineController{}
	if err := vmcontroller.SetupWithManager(mgr); err != nil {
		panic(err)
	}

	// Start the manager, handling system signals for graceful shutdown
	panic(mgr.Start(signals.SetupSignalHandler()))
}
