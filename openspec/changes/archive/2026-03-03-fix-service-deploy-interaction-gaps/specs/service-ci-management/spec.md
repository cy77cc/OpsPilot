# service-ci-management Specification (Delta)

## ADDED Requirements

### Requirement: CI and Manual Deployment Target Contract Consistency
The system SHALL enforce the same deploy target resolution contract for CI-triggered deployments and manual deployments.

#### Scenario: CI trigger uses shared target resolution chain
- **WHEN** CI pipeline triggers a deployment for a service
- **THEN** the system MUST resolve deploy target using the same order as manual deployment
- **AND** the system MUST apply identical scope filters and fallback behavior

## MODIFIED Requirements

### Requirement: Service CI pipeline configuration management
The system SHALL provide APIs to create, read, update, and delete CI pipeline configurations for each service, including repository source, branch strategy, build steps, artifact target, and trigger mode.

#### Scenario: Create CI configuration for a service
- **WHEN** an authorized user submits a valid CI configuration for an existing service
- **THEN** the system MUST persist the configuration and return a unique pipeline configuration identifier

#### Scenario: Reject invalid CI configuration
- **WHEN** an authorized user submits a CI configuration with missing required fields (repository, artifact target, or trigger mode)
- **THEN** the system MUST reject the request with validation errors and MUST NOT persist the configuration

#### Scenario: Validate deploy target policy in CI configuration
- **WHEN** CI configuration defines deploy target constraints or overrides
- **THEN** the system MUST validate they are compatible with service runtime and scope policy
- **AND** the system MUST reject incompatible target policy definitions

### Requirement: CI trigger policy enforcement
The system SHALL enforce CI trigger policy per service and MUST support at least manual trigger and source-event trigger.

#### Scenario: Manual trigger execution
- **WHEN** an authorized user manually triggers a CI run for a service with manual trigger enabled
- **THEN** the system MUST enqueue a build execution request and record the trigger actor and timestamp

#### Scenario: Block unsupported trigger mode
- **WHEN** a trigger event is received for a service whose configured trigger mode does not allow that event type
- **THEN** the system MUST reject execution and record the reason in the CI run history

#### Scenario: CI deployment target resolution failure diagnostics
- **WHEN** a CI-triggered deployment cannot resolve deploy target after fallback resolution
- **THEN** the system MUST mark run result as failed with explicit reason
- **AND** the system MUST include actionable hints for target configuration remediation

## REMOVED Requirements

None.
