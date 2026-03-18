# k8s-workload-operations Specification

## Purpose
TBD - created by archiving change k8s-write-tools. Update Purpose after archive.
## Requirements
### Requirement: AI can scale K8s Deployment

The system SHALL provide a `k8s_scale_deployment` tool that allows AI to adjust the replica count of a Deployment.

#### Scenario: Scale up deployment
- **WHEN** AI calls `k8s_scale_deployment` with `cluster_id`, `namespace`, `name`, and `replicas=5`
- **THEN** system updates the Deployment's replica count to 5
- **AND** system returns the new replica count and previous replica count

#### Scenario: Scale down deployment
- **WHEN** AI calls `k8s_scale_deployment` with `replicas=1`
- **THEN** system scales down the Deployment to 1 replica

#### Scenario: Scale to zero requires approval
- **WHEN** AI calls `k8s_scale_deployment` with `replicas=0`
- **THEN** system triggers approval flow before execution
- **AND** approval preview shows "Deployment will have 0 replicas (effectively stopped)"

### Requirement: AI can restart K8s Deployment

The system SHALL provide a `k8s_restart_deployment` tool that triggers a rolling restart of all Pods in a Deployment.

#### Scenario: Restart deployment
- **WHEN** AI calls `k8s_restart_deployment` with `cluster_id`, `namespace`, and `name`
- **THEN** system triggers rolling restart by updating `spec.template.metadata.annotations["kubectl.kubernetes.io/restartedAt"]`
- **AND** system returns confirmation with timestamp

#### Scenario: Restart requires approval
- **WHEN** AI calls `k8s_restart_deployment`
- **THEN** system triggers approval flow with risk level "medium"
- **AND** approval preview shows "Deployment will rolling restart, may cause brief service instability"

### Requirement: AI can delete K8s Pod

The system SHALL provide a `k8s_delete_pod` tool that allows AI to delete a specific Pod.

#### Scenario: Delete pod
- **WHEN** AI calls `k8s_delete_pod` with `cluster_id`, `namespace`, and `name`
- **THEN** system deletes the specified Pod
- **AND** system returns confirmation with Pod name

#### Scenario: Delete pod requires approval
- **WHEN** AI calls `k8s_delete_pod`
- **THEN** system triggers approval flow with risk level "high"
- **AND** approval preview shows "Pod will be deleted, controller may recreate a new Pod"

#### Scenario: Delete pod with grace period
- **WHEN** AI calls `k8s_delete_pod` with optional `grace_period_seconds=30`
- **THEN** system uses the specified grace period for graceful termination

### Requirement: AI can rollback K8s Deployment

The system SHALL provide a `k8s_rollback_deployment` tool that allows AI to rollback a Deployment to a previous revision.

#### Scenario: Rollback to previous revision
- **WHEN** AI calls `k8s_rollback_deployment` with `cluster_id`, `namespace`, and `name`
- **THEN** system rolls back the Deployment to the previous revision
- **AND** system returns the revision number rolled back to

#### Scenario: Rollback requires approval
- **WHEN** AI calls `k8s_rollback_deployment`
- **THEN** system triggers approval flow with risk level "medium"
- **AND** approval preview shows "Deployment will rollback to previous version"

### Requirement: AI can delete K8s Deployment

The system SHALL provide a `k8s_delete_deployment` tool that allows AI to delete a Deployment.

#### Scenario: Delete deployment
- **WHEN** AI calls `k8s_delete_deployment` with `cluster_id`, `namespace`, and `name`
- **THEN** system deletes the specified Deployment
- **AND** system returns confirmation with Deployment name

#### Scenario: Delete deployment requires approval with high risk
- **WHEN** AI calls `k8s_delete_deployment`
- **THEN** system triggers approval flow with risk level "critical"
- **AND** approval preview shows "Deployment will be permanently deleted, service will stop"
- **AND** approval preview includes warning "This action is irreversible"

### Requirement: All K8s write operations require valid cluster access

The system SHALL validate that the `cluster_id` parameter corresponds to an existing cluster with accessible kubeconfig before executing any write operation.

#### Scenario: Invalid cluster ID
- **WHEN** AI calls any K8s write tool with non-existent `cluster_id`
- **THEN** system returns error "cluster not found or no kubeconfig available"

### Requirement: K8s write operations integrate with approval middleware

The system SHALL automatically trigger the approval middleware for all K8s write operations based on the tool name registration in `DefaultNeedsApproval`.

#### Scenario: Approval flow triggered
- **WHEN** AI calls any K8s write tool
- **THEN** system interrupts execution with `tool.StatefulInterrupt`
- **AND** sends `ApprovalInfo` via SSE to frontend
- **AND** waits for `ApprovalResult` via `ResumeWithParams`

#### Scenario: Approval granted
- **WHEN** user approves the operation via frontend
- **THEN** system resumes execution and performs the operation
- **AND** returns the operation result

#### Scenario: Approval rejected
- **WHEN** user rejects the operation
- **THEN** system returns "tool '<name>' disapproved by user" message
- **AND** does not perform the operation

