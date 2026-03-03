## MODIFIED Requirements

### Requirement: Environment runtime bootstrap via SSH
The system MUST provide environment bootstrap workflows that execute runtime installation through remote SSH for both Kubernetes and Compose targets, and MUST support mirror repository mode for weak-offline enterprise environments.

#### Scenario: Bootstrap Kubernetes runtime on new environment
- **WHEN** an authorized operator submits an environment bootstrap request with runtime `k8s`, SSH connection info, and a valid runtime package version
- **THEN** the system MUST create an installation job, execute remote install steps, and persist step-level status and logs

#### Scenario: Bootstrap Compose runtime on new environment
- **WHEN** an authorized operator submits an environment bootstrap request with runtime `compose`, SSH connection info, and a valid runtime package version
- **THEN** the system MUST execute Compose installation and post-install verification and return environment readiness status

#### Scenario: Bootstrap in mirror mode
- **WHEN** an authorized operator submits bootstrap with `repo_mode=mirror` and internal repository settings
- **THEN** the system MUST install runtime dependencies from configured mirror repositories
- **AND** the system MUST avoid falling back to public repositories unless explicitly allowed by policy

### Requirement: Runtime package manifest and integrity validation
The system MUST install runtime binaries only from approved package manifests and MUST verify package integrity before execution.

#### Scenario: Reject package with checksum mismatch
- **WHEN** the installation job resolves a runtime package whose checksum does not match the manifest
- **THEN** the system MUST fail the job before installation and record an integrity error in diagnostics

#### Scenario: Reject unsupported runtime package version
- **WHEN** the requested runtime version is not present in the approved package catalog
- **THEN** the system MUST reject the bootstrap request with a version-not-allowed error

#### Scenario: Reject mirror mode without required repository configuration
- **WHEN** bootstrap is requested with `repo_mode=mirror` but required mirror repository or image repository settings are missing
- **THEN** the system MUST reject the request with validation diagnostics and MUST NOT start remote execution

### Requirement: Installation rollback and diagnostics
The system MUST provide rollback hooks for failed bootstrap jobs and MUST persist structured diagnostics for remediation.

#### Scenario: Rollback after partial install failure
- **WHEN** a bootstrap job fails after changing remote runtime state
- **THEN** the system MUST execute runtime-specific rollback hooks and mark the job as `failed` with rollback outcome

#### Scenario: Query bootstrap diagnostics
- **WHEN** an authorized user opens the bootstrap job detail
- **THEN** the system MUST return ordered step diagnostics with timestamp, host, exit code, and remediation hint

#### Scenario: Return categorized diagnostics for mirror and endpoint failures
- **WHEN** bootstrap fails due to mirror repository access, registry access, endpoint reachability, or etcd connectivity
- **THEN** the system MUST categorize diagnostics by failure domain and return remediation-oriented messages
