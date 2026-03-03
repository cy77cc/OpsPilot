# deployment-release-management Specification (Delta)

## ADDED Requirements

### Requirement: Deploy Target Resolution Fallback
The system SHALL resolve deployment target using a unified fallback chain when explicit target input is incomplete.

#### Scenario: Resolve target by fallback chain
- **WHEN** deployment request omits explicit cluster target
- **THEN** the system MUST resolve target in order: request override -> service default target -> scoped active target fallback
- **AND** the scoped fallback MUST filter by service project, team, environment, and target type

#### Scenario: Persist resolved fallback target
- **WHEN** deployment succeeds through scoped fallback
- **THEN** the system MUST persist the resolved target as service default target for subsequent deployments

## MODIFIED Requirements

### Requirement: Enhanced Release Creation Wizard
The system SHALL provide a 5-step wizard for creating releases with service selection, configuration, preview, strategy selection, and confirmation.

#### Scenario: Step 1 - Select service and target
- **WHEN** user starts release creation
- **THEN** system displays service selection with search, current deployment info, and target selection filtered by environment

#### Scenario: Service search and selection
- **WHEN** user searches for a service
- **THEN** system filters services by name and displays version, last update time, and current deployments

#### Scenario: Target selection with environment filter
- **WHEN** user selects an environment filter
- **THEN** system displays only deployment targets for that environment with readiness status

#### Scenario: Production environment warning
- **WHEN** user selects a Production target
- **THEN** system displays warning that approval will be required

#### Scenario: Step 2 - Configure variables
- **WHEN** user proceeds to configuration step
- **THEN** system displays manifest template variables that need to be filled

#### Scenario: Variable validation
- **WHEN** user fills in variables
- **THEN** system validates no template placeholders ({{variable}}) remain unresolved

#### Scenario: Step 3 - Preview manifest
- **WHEN** user proceeds to preview step
- **THEN** system calls preview API and displays resolved manifest, checks, and warnings

#### Scenario: Preview token generation
- **WHEN** preview is generated
- **THEN** system receives a preview token valid for 30 minutes

#### Scenario: Preview checks display
- **WHEN** preview completes
- **THEN** system displays validation checks (target type, service name) and any warnings

#### Scenario: Step 4 - Select deployment strategy
- **WHEN** user proceeds to strategy step
- **THEN** system displays strategy options (Rolling Update, Blue-Green, Canary) with descriptions

#### Scenario: Strategy selection
- **WHEN** user selects a deployment strategy
- **THEN** system stores the strategy choice for release creation

#### Scenario: Step 5 - Confirm and submit
- **WHEN** user proceeds to confirmation step
- **THEN** system displays summary of service, target, strategy, and variables

#### Scenario: Submit release
- **WHEN** user clicks "Create Release" with valid preview token
- **THEN** system creates release record and initiates approval workflow if required

#### Scenario: Submit release with unresolved target selection
- **WHEN** user submits release without explicit target and no service default target exists
- **THEN** system MUST attempt scoped fallback target resolution before failing
- **AND** failure response MUST include actionable resolution hints

### Requirement: Release Diagnostics
The system SHALL capture and display diagnostic information for failed releases.

#### Scenario: Capture deployment errors
- **WHEN** deployment fails
- **THEN** system stores diagnostics with runtime, stage, error code, message, and summary

#### Scenario: Display diagnostics
- **WHEN** user views failed release
- **THEN** system displays diagnostic information with error details and suggested actions

#### Scenario: Truncate long error messages
- **WHEN** error output exceeds 800 characters
- **THEN** system truncates and stores first 800 characters

#### Scenario: Display actionable deploy target diagnostics
- **WHEN** deployment fails due to missing deploy target
- **THEN** diagnostics MUST include evaluated scope dimensions (project, team, env, target type)
- **AND** diagnostics MUST include at least one actionable fix suggestion

## REMOVED Requirements

None.
