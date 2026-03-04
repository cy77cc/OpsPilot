# service-configuration-management Specification

## Purpose
TBD - created by archiving change refine-service-management-scope. Update Purpose after archive.
## Requirements
### Requirement: Service Configuration Inline Editing
The system SHALL provide inline editing for service configuration without requiring a modal dialog.

#### Scenario: View service configuration
- **WHEN** user navigates to service detail page and selects the "配置" tab
- **THEN** the system MUST display configuration cards in read-only mode
- **AND** each card MUST have an "编辑" button to enable editing

#### Scenario: Edit service configuration inline
- **WHEN** user clicks the "编辑" button on a configuration card
- **THEN** the system MUST switch that card to edit mode
- **AND** display form fields for the configuration content
- **AND** display "保存" and "取消" buttons

#### Scenario: Save inline configuration changes
- **WHEN** user modifies configuration and clicks "保存"
- **THEN** the system MUST validate the configuration
- **AND** persist changes to the database
- **AND** switch the card back to read-only mode
- **AND** display a success message

#### Scenario: Cancel inline configuration changes
- **WHEN** user clicks "取消" after modifying configuration
- **THEN** the system MUST discard unsaved changes
- **AND** restore the original values
- **AND** switch the card back to read-only mode

### Requirement: Service Configuration Tab Structure
The system SHALL organize service configuration into logical sections without embedding deployment target selection, and SHALL provide manual deployment actions by creating unified release requests instead of running an independent service-side deployment state machine.

#### Scenario: Configuration tab content
- **WHEN** user views the "配置" tab of a service
- **THEN** the system MUST display:
  - 服务配置卡片 (basic info, runtime, labels, config content)
  - 环境变量集卡片 (environment variables)
  - 渲染输出预览 (rendered YAML preview)
- **AND** the system MUST NOT display any "部署目标" or cluster selection UI in this tab

#### Scenario: Manual deploy action from service context
- **WHEN** user triggers manual deployment from service-related entry
- **THEN** the system MUST create a unified release request and return a release identifier for lifecycle tracking
- **AND** the system MUST NOT create a separate service-only deployment lifecycle record as the primary execution state

### Requirement: Service List Edit Configuration Navigation
The system SHALL navigate to service detail configuration tab when editing from the list page.

#### Scenario: Click edit configuration from list
- **WHEN** user clicks "编辑配置" from the service list dropdown menu
- **THEN** the system MUST navigate to `/services/:id?tab=config`
- **AND** activate the "配置" tab on the detail page

#### Scenario: Direct URL access to configuration tab
- **WHEN** user navigates directly to `/services/:id?tab=config`
- **THEN** the system MUST load the service detail page
- **AND** automatically activate the "配置" tab

### Requirement: Service Kind And Visibility Fields
The system SHALL support service kind and visibility metadata in service create/edit flows.

#### Scenario: Set service kind and visibility on create
- **WHEN** user creates a service in the provision page
- **THEN** the system MUST allow selecting `service_kind` (`middleware` or `business`)
- **AND** allow selecting `visibility` (`private`, `team`, `team-granted`, `public`)
- **AND** persist icon, tags, and granted teams metadata

#### Scenario: Default visibility by service kind
- **WHEN** user does not explicitly provide visibility
- **THEN** the system MUST default to `public` for `middleware`
- **AND** default to `team` for `business`

### Requirement: Service Visibility Management API
The system SHALL provide APIs to manage visibility and granted teams from service domain.

#### Scenario: Update visibility via service API
- **WHEN** user calls `PUT /api/v1/services/:id/visibility`
- **THEN** the system MUST update visibility for that service

#### Scenario: Update granted teams via service API
- **WHEN** user calls `PUT /api/v1/services/:id/grant-teams`
- **THEN** the system MUST update granted team IDs for that service
