## ADDED Requirements

### Requirement: Bootstrap preflight diagnostics for mirror, VIP, and external etcd
The system MUST perform preflight checks for package mirror accessibility, image repository accessibility, control-plane endpoint reachability, and external etcd connectivity before executing bootstrap steps.

#### Scenario: Preflight fails for mirror package source
- **WHEN** bootstrap is requested with `repo_mode=mirror` and the configured package repository is unreachable or missing required packages
- **THEN** the system MUST fail before installation starts
- **AND** the system MUST return structured diagnostics with remediation hints

#### Scenario: Preflight fails for external etcd TLS
- **WHEN** bootstrap is requested with `etcd_mode=external` and etcd endpoints fail TLS verification
- **THEN** the system MUST fail before kubeadm init
- **AND** the diagnostics MUST identify the failing endpoint and certificate error category

## MODIFIED Requirements

### Requirement: Cluster Creation Wizard
The system SHALL provide a multi-step wizard for creating new Kubernetes clusters through automated bootstrap, including advanced options for version source, repository mode, control-plane endpoint mode, VIP provider, and etcd mode.

#### Scenario: Start cluster creation
- **WHEN** user clicks "Create Cluster" button
- **THEN** system displays a wizard with steps: basic info, control plane selection, worker node selection, network configuration, and advanced bootstrap settings

#### Scenario: Select control plane host
- **WHEN** user is on the control plane selection step
- **THEN** system displays available hosts with resource information and allows selection of one host

#### Scenario: Select worker nodes
- **WHEN** user is on the worker node selection step
- **THEN** system displays available hosts and allows selection of multiple worker nodes

#### Scenario: Configure CNI plugin
- **WHEN** user is on the network configuration step
- **THEN** system provides options for CNI plugins (Calico, Flannel, etc.) with descriptions

#### Scenario: Configure advanced bootstrap options
- **WHEN** user expands advanced settings
- **THEN** system MUST allow configuring Kubernetes version selection mode, repository mode (`online|mirror`), `imageRepository`, endpoint mode (`nodeIP|vip|lbDNS`), VIP provider (`kube-vip|keepalived`), and etcd mode (`stacked|external`)

#### Scenario: Submit cluster creation
- **WHEN** user completes required wizard fields and clicks "Create"
- **THEN** system initiates cluster bootstrap process and displays progress

### Requirement: Cluster Creation with Kubeadm Automation
The system MUST provide fully automated Kubernetes cluster bootstrap using kubeadm on selected hosts via SSH with config-driven initialization and runtime-mode-specific validation.

#### Scenario: Bootstrap cluster on selected hosts
- **WHEN** an authorized operator submits a cluster bootstrap request with control plane host, worker hosts, K8s version, CNI selection, endpoint mode, repository mode, and etcd mode
- **THEN** the system MUST execute preflight checks, install containerd, install kubeadm/kubelet/kubectl, generate kubeadm config, initialize control plane, install CNI, join workers, and store kubeconfig
- **AND** the system MUST report step-by-step progress and allow cancellation

#### Scenario: Bootstrap with mirror repository mode
- **WHEN** bootstrap runs with `repo_mode=mirror`
- **THEN** the system MUST use configured internal package repositories and configured `imageRepository` for image pulls
- **AND** the system MUST fail fast with actionable diagnostics when mirror sources are unavailable

#### Scenario: Bootstrap with VIP endpoint mode
- **WHEN** bootstrap runs with endpoint mode `vip` or `lbDNS`
- **THEN** kubeadm init configuration MUST include `controlPlaneEndpoint`
- **AND** the system MUST execute selected VIP provider automation and verify API accessibility through the configured endpoint

#### Scenario: Bootstrap with external etcd mode
- **WHEN** bootstrap runs with `etcd_mode=external`
- **THEN** kubeadm config MUST reference external etcd endpoints and TLS materials
- **AND** the system MUST NOT initialize stacked etcd on control-plane node

#### Scenario: Bootstrap fails mid-process
- **WHEN** a bootstrap step fails after partial installation
- **THEN** the system MUST execute rollback scripts for completed steps and mark the task as failed with diagnostic output
- **AND** the system MUST preserve step-level logs for troubleshooting
