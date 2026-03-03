# Capability: deployment-infrastructure-management

## Purpose
This capability covers the management of infrastructure resources required for deployment operations, including Kubernetes clusters, credentials, and host pools.
## Requirements
### Requirement: Cluster List and Overview
The system SHALL display a list of all Kubernetes clusters with their current status, resource utilization, and management mode.

#### Scenario: View cluster list
- **WHEN** user navigates to the cluster management page
- **THEN** system displays all clusters with name, status, node count, and management mode

#### Scenario: Filter clusters by status
- **WHEN** user selects a status filter (active, inactive, error)
- **THEN** system displays only clusters matching the selected status

### Requirement: Cluster Creation Wizard
The system SHALL provide a multi-step wizard for creating new Kubernetes clusters through automated bootstrap.

#### Scenario: Start cluster creation
- **WHEN** user clicks "Create Cluster" button
- **THEN** system displays a wizard with steps: basic info, control plane selection, worker node selection, CNI configuration

#### Scenario: Select control plane host
- **WHEN** user is on the control plane selection step
- **THEN** system displays available hosts with resource information and allows selection of one host

#### Scenario: Select worker nodes
- **WHEN** user is on the worker node selection step
- **THEN** system displays available hosts and allows selection of multiple worker nodes

#### Scenario: Configure CNI plugin
- **WHEN** user is on the CNI configuration step
- **THEN** system provides options for CNI plugins (Calico, Flannel, etc.) with descriptions

#### Scenario: Submit cluster creation
- **WHEN** user completes all wizard steps and clicks "Create"
- **THEN** system initiates cluster bootstrap process and displays progress

### Requirement: Cluster Bootstrap Progress Tracking
The system SHALL display real-time progress of cluster bootstrap operations with detailed phase information.

#### Scenario: View bootstrap progress
- **WHEN** cluster bootstrap is in progress
- **THEN** system displays current phase (preflight, install, verify) with progress percentage

#### Scenario: View bootstrap logs
- **WHEN** user views bootstrap progress
- **THEN** system displays real-time logs from the bootstrap process

#### Scenario: Bootstrap completion
- **WHEN** cluster bootstrap completes successfully
- **THEN** system updates cluster status to "active" and displays success message

#### Scenario: Bootstrap failure
- **WHEN** cluster bootstrap fails
- **THEN** system displays error details and provides rollback option

### Requirement: Cluster Detail View
The system SHALL display detailed information about a specific cluster including nodes, resources, and deployment history.

#### Scenario: View cluster details
- **WHEN** user clicks on a cluster from the list
- **THEN** system displays cluster details including endpoint, nodes, resource usage, and recent deployments

#### Scenario: View cluster nodes
- **WHEN** user views cluster details
- **THEN** system displays all nodes with their roles (control-plane, worker), status, and resource allocation

### Requirement: Cluster Creation with Kubeadm Automation
The system MUST provide fully automated Kubernetes cluster bootstrap using kubeadm on selected hosts via SSH.

#### Scenario: Bootstrap cluster on selected hosts
- **WHEN** an authorized operator submits a cluster bootstrap request with control plane host, worker hosts, K8s version, and CNI selection
- **THEN** the system MUST execute preflight checks, install containerd, install kubeadm/kubelet/kubectl, initialize control plane, install CNI, join workers, and store kubeconfig
- **AND** the system MUST report step-by-step progress and allow cancellation

#### Scenario: Bootstrap fails mid-process
- **WHEN** a bootstrap step fails after partial installation
- **THEN** the system MUST execute rollback scripts for completed steps and mark the task as failed with diagnostic output
- **AND** the system MUST preserve step-level logs for troubleshooting

### Requirement: External Cluster Import via Kubeconfig
The system MUST allow importing existing Kubernetes clusters by providing kubeconfig credentials.

#### Scenario: Import cluster with valid kubeconfig
- **WHEN** an authorized user submits a kubeconfig with cluster admin access
- **THEN** the system MUST validate connectivity, extract cluster metadata, create Cluster and Credential records, and sync initial node information
- **AND** the system MUST mark the cluster source as `external_managed`

#### Scenario: Import with invalid kubeconfig
- **WHEN** a user submits a malformed or unauthorized kubeconfig
- **THEN** the system MUST reject the import with specific validation errors
- **AND** the system MUST NOT create any cluster records

### Requirement: Cluster Source Distinction
The system MUST distinguish between platform-managed and externally-managed clusters with appropriate metadata and lifecycle controls.

#### Scenario: Platform-managed cluster lifecycle
- **WHEN** a cluster is created via bootstrap workflow
- **THEN** the system MUST set `source` to `platform_managed` and enable full lifecycle operations (upgrade, scale, delete)

#### Scenario: External-managed cluster limitations
- **WHEN** a cluster is imported from external kubeconfig
- **THEN** the system MUST set `source` to `external_managed` and disable destructive operations that require infrastructure access

### Requirement: Cluster Node Tracking
The system MUST maintain a separate `cluster_nodes` table tracking node status synced from Kubernetes API.

#### Scenario: Sync nodes after cluster creation
- **WHEN** a cluster bootstrap completes successfully
- **THEN** the system MUST query the Kubernetes API for node information and populate `cluster_nodes` records
- **AND** each node MUST be linked to its host record if applicable

#### Scenario: Node status refresh
- **WHEN** an authorized user requests node list for a cluster
- **THEN** the system MUST query the Kubernetes API and update `cluster_nodes` with current status

### Requirement: Node Lifecycle Management
The system MUST support adding and removing nodes from platform-managed clusters.

#### Scenario: Add worker node to cluster
- **WHEN** an authorized operator requests adding a new worker node to a platform-managed cluster
- **THEN** the system MUST execute kubeadm join on the target host
- **AND** update cluster_nodes and sync node status

#### Scenario: Remove node from cluster
- **WHEN** an authorized operator requests removing a node from a cluster
- **THEN** the system MUST cordon and drain the node, execute kubeadm reset, and delete the node record
- **AND** update host record to remove cluster association

#### Scenario: Node operation on external cluster
- **WHEN** an operator attempts node lifecycle operations on an externally-managed cluster
- **THEN** the system MUST reject the operation with an appropriate error message

### Requirement: Cluster Connectivity Testing
The system MUST provide cluster connectivity testing with detailed diagnostics.

#### Scenario: Test cluster connection
- **WHEN** an authorized user requests connectivity test for a cluster
- **THEN** the system MUST attempt to connect using stored credentials
- **AND** return connection status, latency, and Kubernetes version

#### Scenario: Test with stale credentials
- **WHEN** connectivity test fails due to expired or invalid credentials
- **THEN** the system MUST update credential status and return specific error guidance

### Requirement: Cluster Update and Deletion
The system MUST support updating cluster metadata and safe deletion.

#### Scenario: Update cluster metadata
- **WHEN** an authorized user updates cluster name, description, or labels
- **THEN** the system MUST persist changes and update the audit log

#### Scenario: Delete platform-managed cluster
- **WHEN** an authorized user deletes a platform-managed cluster
- **THEN** the system MUST execute kubeadm reset on all nodes, delete credentials, and remove cluster records
- **AND** require confirmation for destructive operation

#### Scenario: Delete external cluster reference
- **WHEN** an authorized user deletes an externally-managed cluster
- **THEN** the system MUST only remove platform records without affecting the actual cluster
- **AND** preserve audit trail of the deletion

### Requirement: Cluster Event Visibility
The system MUST provide access to Kubernetes cluster events.

#### Scenario: Query cluster events
- **WHEN** an authorized user requests events for a cluster
- **THEN** the system MUST return Kubernetes events sorted by creation timestamp
- **AND** support filtering by namespace, resource type, and severity

### Requirement: Real-time Kubernetes Resource Query
The system MUST provide real-time resource visibility by querying the Kubernetes API on demand.

#### Scenario: List namespaces in cluster
- **WHEN** an authorized user requests namespace list for a cluster
- **THEN** the system MUST use the cluster's stored credentials to query the Kubernetes API and return all namespaces

#### Scenario: List workloads in namespace
- **WHEN** an authorized user requests workload list for a cluster and namespace
- **THEN** the system MUST return Deployments, StatefulSets, DaemonSets, Jobs, and CronJobs with status information
- **AND** each workload MUST include replica counts, health status, and age

#### Scenario: List pods with details
- **WHEN** an authorized user requests pod list for a cluster and namespace
- **THEN** the system MUST return pods with node assignment, status, IP, and container information

### Requirement: Service and Configuration Visibility
The system MUST allow viewing Kubernetes Services, Ingresses, ConfigMaps, and Secrets.

#### Scenario: List services and ingresses
- **WHEN** an authorized user requests service list for a cluster and namespace
- **THEN** the system MUST return Services with type, cluster IP, ports, and Ingresses with host and path rules

#### Scenario: List configuration resources
- **WHEN** an authorized user requests config list for a cluster and namespace
- **THEN** the system MUST return ConfigMaps and Secrets metadata (NOT secret values)
- **AND** Secrets MUST only display name, type, and creation timestamp

### Requirement: Storage Resource Visibility
The system MUST provide visibility into PersistentVolumes and PersistentVolumeClaims.

#### Scenario: List storage resources
- **WHEN** an authorized user requests storage list for a cluster
- **THEN** the system MUST return PVs with capacity, access modes, and status
- **AND** return PVCs with namespace, capacity, and bound PV reference

### Requirement: Deployment-centric Service View
The system MUST allow viewing deployed services from the cluster perspective.

#### Scenario: List deployed services in cluster
- **WHEN** an authorized user requests service deployments for a cluster
- **THEN** the system MUST return services from the deployment table that are deployed to this cluster
- **AND** each service MUST include environment, last deployment time, and current status

### Requirement: Cluster Credential Caching
The system MUST cache Kubernetes client connections to reduce latency.

#### Scenario: Cache client connection
- **WHEN** the system establishes a Kubernetes client for a cluster
- **THEN** it MUST cache the client for subsequent requests
- **AND** invalidate cache on credential update or connection failure

### Requirement: Credential Management
The system SHALL allow users to manage cluster credentials for both platform-managed and external clusters.

#### Scenario: List credentials
- **WHEN** user navigates to credential management page
- **THEN** system displays all credentials with name, type, source, and last test status

#### Scenario: Register platform credential
- **WHEN** user selects a platform-managed cluster and clicks "Register Credential"
- **THEN** system automatically extracts kubeconfig from the cluster and creates a credential record

#### Scenario: Import external credential via kubeconfig
- **WHEN** user clicks "Import External Credential" and provides kubeconfig file
- **THEN** system validates the kubeconfig and creates a credential record with source "external_managed"

#### Scenario: Import external credential via certificates
- **WHEN** user clicks "Import External Credential" and provides endpoint, CA cert, client cert, and client key
- **THEN** system validates the certificates and creates a credential record

#### Scenario: Test credential connectivity
- **WHEN** user clicks "Test Connection" on a credential
- **THEN** system attempts to connect to the cluster and displays result with latency

#### Scenario: Successful connectivity test
- **WHEN** credential connectivity test succeeds
- **THEN** system updates last_test_status to "ok" and displays success message with latency

#### Scenario: Failed connectivity test
- **WHEN** credential connectivity test fails
- **THEN** system updates last_test_status to "failed" and displays error message

### Requirement: Credential Security
The system SHALL encrypt all sensitive credential data (kubeconfig, certificates, tokens) before storage.

#### Scenario: Store encrypted credentials
- **WHEN** user creates or imports a credential
- **THEN** system encrypts all sensitive fields using the configured encryption key before database storage

#### Scenario: Retrieve credentials for use
- **WHEN** system needs to use a credential for cluster operations
- **THEN** system decrypts the credential data using the encryption key

### Requirement: Host Pool Management
The system SHALL display and manage the pool of available hosts for cluster and deployment target operations.

#### Scenario: View host pool
- **WHEN** user navigates to host management page
- **THEN** system displays all hosts with IP, status, resource capacity, and current assignments

#### Scenario: Filter available hosts
- **WHEN** user filters hosts by status "available"
- **THEN** system displays only hosts that are not currently assigned to any cluster or target

#### Scenario: View host assignments
- **WHEN** user views a host's details
- **THEN** system displays which clusters or deployment targets the host is assigned to

### Requirement: RBAC Protection
The system SHALL require appropriate permissions for all infrastructure management operations.

#### Scenario: Unauthorized cluster creation
- **WHEN** user without cluster management permission attempts to create a cluster
- **THEN** system returns 403 Forbidden error

#### Scenario: Unauthorized credential access
- **WHEN** user without credential management permission attempts to view credentials
- **THEN** system returns 403 Forbidden error

#### Scenario: Authorized operations
- **WHEN** user with appropriate permissions performs infrastructure operations
- **THEN** system allows the operation and logs the action

### Requirement: Cluster Environment Type
The system SHALL maintain an environment type for each cluster to constrain deployment targets.

#### Scenario: Cluster with environment type
- **WHEN** a cluster is created or imported
- **THEN** the system MUST set `env_type` to one of: `development`, `staging`, `production`
- **AND** the default value MUST be `development`

#### Scenario: Update cluster environment type
- **WHEN** an authorized user updates a cluster's environment type
- **THEN** the system MUST persist the change
- **AND** the system MUST log the change in the audit trail

### Requirement: Deployment Environment Matching
The system SHALL validate environment matching when deploying a service to a cluster.

#### Scenario: Deploy to matching environment
- **WHEN** a deployment is requested with `service_id` and `cluster_id`
- **AND** the service's `env` matches the cluster's `env_type`
- **THEN** the system MUST allow the deployment to proceed

#### Scenario: Reject deployment to mismatched environment
- **WHEN** a deployment is requested with `service_id` and `cluster_id`
- **AND** the service's `env` does NOT match the cluster's `env_type`
- **THEN** the system MUST reject the deployment with error code `ENV_MISMATCH`
- **AND** the error message MUST indicate the expected and actual environment types

#### Scenario: Deployment without cluster_id
- **WHEN** a deployment is requested without `cluster_id`
- **THEN** the system MUST reject the request with validation error
- **AND** the error message MUST indicate `cluster_id` is required

