# provider-proxmox
// TODO(user): Add simple overview of use/purpose

## Description
// TODO(user): An in-depth paragraph about your project and overview of use

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- Accesso a un cluster Kubernetes versione `v1.11.3+`
- Ambiente Proxmox con API abilitate e accesso a rete dal cluster Kubernetes

## Installazione e Configurazione

### 1. Creazione del cluster Kubernetes
Per creare un cluster Kubernetes locale, usa Kind (Kubernetes in Docker). Questo ambiente isolato facilita il testing dell’operatore.

kind create cluster --name crossplane
kind create namespace provider

### 2. Installazione di Crossplane
Installare Crossplane nel namespace provider del cluster Kubernetes usando Helm:

helm repo add crossplane https://charts.crossplane.io/stable
helm install crossplane crossplane/crossplane --namespace provider

### 3. Configurazione delle credenziali di Proxmox

Configura un Secret Kubernetes che memorizzi le credenziali necessarie per accedere a Proxmox, come l’username, password e endpoint API

kubectl create secret generic proxmox-credentials -n provider \
  --from-literal=username="username@pve" \
  --from-literal=password="password" \
  --from-literal=endpoint=https://endpoint:8006

### 4. Generare i CRD

make generate
make manifests  

### 5. Creazione dell'Immagine Docker

make docker-build docker-push IMG=<registry>/provider-proxmox:<tag> .

### 6. Creazione delle risorse
kubectl apply -f config/crd/bases
kubectl apply -f config/rbac
kubectl apply -f config/service
kubectl apply -f config/manager
kubectl apply -f config/resources

### 7. Partire il controller e creare la VirtualMachine

go run ./cmd/main.go

kubectl apply -f config/samples/Virtualmachine.yaml



Questo `README.md` fornisce una guida completa per l'installazione, configurazione e utilizzo del provider Proxmox per Crossplane. Contiene tutte le istruzioni necessarie per avviare e gestire il provider in un ambiente Kubernetes.