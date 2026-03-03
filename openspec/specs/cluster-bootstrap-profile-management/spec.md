# Capability: cluster-bootstrap-profile-management

## Purpose
Define, validate, and resolve reusable cluster bootstrap profiles with governed Kubernetes version selection.

## Requirements
### Requirement: Bootstrap profile definition and validation
The system MUST provide a bootstrap profile model that captures version channel, package repository mode, image repository, control-plane endpoint mode, VIP provider, and etcd mode with strict cross-field validation.

#### Scenario: Create valid bootstrap profile
- **WHEN** an authorized operator submits a profile with `repo_mode`, `image_repository`, `endpoint_mode`, `vip_provider`, and `etcd_mode` fields that satisfy cross-field rules
- **THEN** the system MUST persist the profile and return a profile identifier for cluster bootstrap selection

#### Scenario: Reject invalid profile combination
- **WHEN** a profile uses an invalid combination (for example `etcd_mode=external` without external endpoints and cert material)
- **THEN** the system MUST reject the request with field-level validation errors and MUST NOT persist the profile

### Requirement: Kubernetes version catalog with support status
The system MUST maintain a Kubernetes version catalog sourced from upstream channels and enriched with platform support status for bootstrap.

#### Scenario: Query version catalog
- **WHEN** an authorized user opens cluster creation version selector
- **THEN** the system MUST return available versions with channel metadata and support status (`supported`, `preview`, `blocked`)

#### Scenario: Reject blocked bootstrap version
- **WHEN** a bootstrap request selects a version marked as `blocked`
- **THEN** the system MUST reject the request and provide at least one supported alternative version

### Requirement: Profile-driven bootstrap request resolution
The system MUST resolve effective bootstrap parameters from selected profile and explicit request overrides using deterministic precedence.

#### Scenario: Resolve effective parameters
- **WHEN** a bootstrap request references a profile and supplies partial overrides
- **THEN** the system MUST resolve parameters in precedence order: explicit request overrides -> profile values -> platform defaults
- **AND** the system MUST record the resolved parameters in task metadata for audit and diagnostics

#### Scenario: Enforce RBAC on profile mutation
- **WHEN** a user without cluster-write permission attempts to create or update bootstrap profiles
- **THEN** the system MUST reject the operation with authorization failure and MUST NOT mutate profile state
