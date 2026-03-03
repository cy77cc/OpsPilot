# service-configuration-management Specification (Delta)

## ADDED Requirements

### Requirement: Service Creation Scope Context Binding
The system SHALL bind service scope fields (project and team) from runtime context and MUST NOT rely on hard-coded default identifiers.

#### Scenario: Create service with context-derived scope
- **WHEN** user creates a service from the service studio
- **THEN** the system MUST persist `project_id` and `team_id` from current user/project context
- **AND** the system MUST NOT write hard-coded fallback values for scope fields

#### Scenario: Reject creation when required scope context is missing
- **WHEN** required scope context cannot be resolved during service creation
- **THEN** the system MUST reject the request with validation error details
- **AND** the error MUST indicate which scope field is missing

### Requirement: Service List Table Operation Completeness
The system SHALL provide complete row actions and sortable columns in list view.

#### Scenario: Display complete row actions in list view
- **WHEN** user switches service list to table mode
- **THEN** each row MUST display action buttons including `启动`, `停止`, and `删除`

#### Scenario: Sort table by core columns
- **WHEN** user clicks sortable headers in table mode
- **THEN** the system MUST support deterministic sorting for at least 服务名 and 状态

### Requirement: Service Creation Localization Consistency
The system SHALL keep service creation page labels and option text consistent with Chinese localization requirements.

#### Scenario: Show localized environment options
- **WHEN** user opens environment selector in service creation page
- **THEN** the system MUST display option labels in Chinese
- **AND** the option values MAY remain canonical enums for API compatibility

## MODIFIED Requirements

### Requirement: Service List Edit Configuration Navigation
The system SHALL navigate to service detail configuration tab when editing from the list page.

#### Scenario: Click edit configuration from list
- **WHEN** user clicks "编辑配置" from the service list dropdown menu
- **THEN** the system MUST navigate to `/services/:id?tab=config`
- **AND** activate the "配置" tab on the detail page

#### Scenario: Click edit action in table row
- **WHEN** user clicks row-level edit action in service list table mode
- **THEN** the system MUST navigate to `/services/:id?tab=config`
- **AND** activate the "配置" tab on the detail page

#### Scenario: Direct URL access to configuration tab
- **WHEN** user navigates directly to `/services/:id?tab=config`
- **THEN** the system MUST load the service detail page
- **AND** automatically activate the "配置" tab

## REMOVED Requirements

None.
