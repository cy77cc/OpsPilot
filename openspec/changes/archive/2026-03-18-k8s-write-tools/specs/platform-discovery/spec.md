## ADDED Requirements

### Requirement: AI can discover platform resources

The system SHALL provide a `platform_discover_resources` tool that allows AI to query available platform resources without prior knowledge of resource IDs.

#### Scenario: Discover all clusters
- **WHEN** AI calls `platform_discover_resources` with `resource_type="clusters"`
- **THEN** system returns list of all K8s clusters with `id`, `name`, `endpoint`, `status`

#### Scenario: Discover all hosts
- **WHEN** AI calls `platform_discover_resources` with `resource_type="hosts"`
- **THEN** system returns list of all hosts with `id`, `name`, `ip`, `status`

#### Scenario: Discover all services
- **WHEN** AI calls `platform_discover_resources` with `resource_type="services"`
- **THEN** system returns list of all services with `id`, `name`, `env`, `status`

#### Scenario: Discover namespaces in a cluster
- **WHEN** AI calls `platform_discover_resources` with `resource_type="namespaces"` and `cluster_id=<id>`
- **THEN** system returns list of namespaces in the specified cluster

#### Scenario: Discover available metrics
- **WHEN** AI calls `platform_discover_resources` with `resource_type="metrics"`
- **THEN** system returns list of available Prometheus metrics with `name`, `type`, `help`

#### Scenario: Discover all resource types
- **WHEN** AI calls `platform_discover_resources` without `resource_type` parameter
- **THEN** system returns overview of all resource types with counts

### Requirement: Resource discovery respects RBAC

The system SHALL apply RBAC checks when returning resource lists, ensuring the requesting user has permission to view the resources.

#### Scenario: User lacks cluster read permission
- **WHEN** AI calls `platform_discover_resources` with `resource_type="clusters"` on behalf of a user without `cluster:read` permission
- **THEN** system returns empty list or filtered results based on permissions
