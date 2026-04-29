# OpsAgent Host Plugin Integration Design

## Problem

The platform already has three relevant capability paths, but they are not unified:

- Host onboarding and remote execution are SSH-centric.
- AI host tools execute through SSH or local shell paths.
- `opsagent` exists only as external documentation, a `proto` definition, and packaged artifacts under `resource/opsagent/`.

The required outcome is larger than "connect one gRPC service". The platform needs a general host plugin framework, with `opsagent` as the first plugin, so that:

- users can choose the plugin while adding a host,
- the platform installs the plugin onto the host through the existing SSH control path,
- the plugin reports host metrics back to the platform,
- AI assistant shell commands, shell scripts, and python scripts run through the plugin-provided sandbox instead of falling back to SSH.

## Confirmed Scope And Decisions

The design is based on the following confirmed decisions:

- Build a complete general-purpose host plugin framework, not an `opsagent` one-off integration.
- `opsagent` is the first plugin implementation in that framework.
- Plugin install, upgrade, reinstall, and uninstall continue to use the platform's existing SSH-based control path.
- After installation, metrics collection, config delivery, and remote command/script execution run through the plugin's gRPC connection.
- AI host execution must be plugin-first and plugin-required. If the required plugin instance is not installed, online, and capability-compatible, execution is denied. There is no SSH fallback for AI execution in this design.

Out of scope for this phase:

- Transparent migration of in-flight AI execution when an agent disconnects mid-run
- Plugin self-upgrade driven by the plugin process itself
- Support for multiple plugin implementations beyond the framework hooks required for future plugins

## Goals

- Introduce a reusable host plugin domain that supports multiple plugins over time.
- Integrate `opsagent` as the first plugin, using the existing packaged artifacts in `resource/opsagent/`.
- Extend host creation and host detail flows so plugin installation and state are first-class platform concepts.
- Implement the platform side of `proto/agent.proto` for registration, heartbeat, metric ingestion, config update, and remote execution.
- Route AI host execution to `opsagent` sandbox execution only when the target host has a qualified online plugin instance.
- Preserve SSH as the installation control plane, while removing it from the AI execution data plane.

## Non-Goals

- Designing a generic plugin marketplace or external plugin upload workflow
- Replacing the existing SSH host management endpoints for human operators
- Defining new observability UI paradigms specific to `opsagent`
- Building a second plugin implementation in the same change

## Architecture

The recommended architecture introduces a dedicated host plugin domain beside the existing host and AI domains.

### High-Level Layers

1. Plugin catalog layer
- Stores plugin definitions, published versions, supported architectures, package references, config schema, and declared capabilities.
- Registers `opsagent` as the first catalog entry.

2. Host plugin instance layer
- Represents the lifecycle of a specific plugin installed on a specific host.
- Tracks desired version, installed version, install status, runtime status, health, heartbeat, capabilities, and last error.

3. Install orchestration layer
- Reuses the platform's SSH execution path to identify host architecture, upload package and config, run install or upgrade scripts, and persist task logs.

4. Agent connectivity layer
- Implements `AgentService.Connect` from `proto/agent.proto`.
- Manages registration, heartbeat, config updates, metric batches, execution output, and execution results.

5. AI execution routing layer
- Resolves the target host, checks plugin instance availability and capability, and dispatches `ExecuteCommand` or `ExecuteScript` messages through the active agent stream.
- Rejects execution when the plugin instance is unavailable or insufficient.

### Control Plane And Data Plane Split

The system explicitly separates installation control from runtime operations:

- SSH control plane: install, upgrade, reinstall, uninstall
- gRPC runtime data plane: registration, heartbeat, metrics, config updates, shell execution, shell script execution, python script execution

This keeps bootstrap and recovery operationally simple while making AI execution semantics consistent and sandbox-enforced.

## Domain Model

### 1. Plugin Catalog

Create a dedicated catalog for supported host plugins.

`host_plugins`
- `id`
- `plugin_key`
- `name`
- `category`
- `description`
- `default_version`
- `status`
- `created_at`
- `updated_at`

`host_plugin_versions`
- `id`
- `plugin_id`
- `version`
- `arch`
- `package_path`
- `install_entry`
- `upgrade_entry`
- `uninstall_entry`
- `checksum`
- `capabilities_json`
- `config_schema_json`
- `created_at`

Notes:

- `package_path` initially points at packaged artifacts already present in `resource/opsagent/`.
- `capabilities_json` for `opsagent` should include at least `metrics.collect`, `exec.shell`, `exec.script.shell`, and `exec.script.python`.

### 2. Host Plugin Instances

Represent plugin lifecycle and availability per host.

`host_plugin_instances`
- `id`
- `host_id`
- `plugin_id`
- `desired_version`
- `installed_version`
- `install_status`
- `runtime_status`
- `health_status`
- `agent_id`
- `last_seen_at`
- `capabilities_json`
- `last_error`
- `created_at`
- `updated_at`

`host_plugin_config_revisions`
- `id`
- `instance_id`
- `version`
- `config_yaml`
- `checksum`
- `delivery_status`
- `created_by`
- `created_at`

Recommended status values:

- `install_status`: `pending`, `running`, `succeeded`, `failed`
- `runtime_status`: `pending_online`, `online`, `offline`, `draining`, `uninstalled`
- `health_status`: `healthy`, `degraded`, `unhealthy`, `unknown`

### 3. Plugin Task And Audit Model

Track installation and lifecycle operations explicitly.

`host_plugin_tasks`
- `id`
- `instance_id`
- `operation`
- `status`
- `requested_by`
- `started_at`
- `finished_at`
- `error_message`
- `created_at`

`host_plugin_task_logs`
- `id`
- `task_id`
- `stream`
- `content`
- `created_at`

This supports UI progress visibility, retries, and auditability without hiding orchestration inside host creation handlers.

### 4. Agent Session And Metrics Intake

Use an in-memory connection registry for live streams, with light persistence of the latest observed state.

Suggested persistence options:

- `opsagent_connection_snapshots` for latest online and heartbeat metadata, or
- reuse `host_plugin_instances` fields for current runtime status and `last_seen_at`

For metrics:

- prefer mapping incoming batches into the platform's host observability model,
- if no suitable existing intake exists, persist normalized host metric samples first and adapt the monitor layer on top.

The agent session itself should remain a runtime concern, not a heavy relational object.

## Module Boundaries

Recommended backend split:

- `internal/modules/hostplugin/...`
  - plugin catalog
  - plugin instance CRUD and queries
  - install task orchestration
  - config revision management
  - host integration APIs

- `internal/modules/opsagent/...`
  - generated `proto` bindings usage
  - gRPC server implementation
  - session registry
  - registration and heartbeat handling
  - metric ingestion adapters
  - remote execution dispatcher

- `internal/modules/ai/...`
  - execution routing changes only
  - capability validation
  - run state and approval integration

- `internal/modules/host/...`
  - host create and host detail entry points
  - no ownership of plugin runtime lifecycle

This split avoids hard-coding `opsagent` behavior into host or AI models, while still allowing first-party integration.

## Key Flows

### 1. Host Creation With Optional Plugin Install

When a user creates a host and selects `opsagent`:

1. Persist the host normally.
2. Create a `host_plugin_instance` for the selected plugin and version.
3. Create an `install` task.
4. Use SSH to:
   - detect host architecture,
   - select the correct package,
   - upload package and generated config,
   - run the install script,
   - enable and start the systemd service.
5. Mark the instance `install_status=succeeded` and `runtime_status=pending_online` on success.
6. Mark the task and instance failed on any error, preserving stdout and stderr in task logs.

The install task owns orchestration. Host creation should not block on opaque inline shell logic.

### 2. Agent Registration And Heartbeat

After installation, the plugin starts and connects back through `AgentService.Connect`.

Registration flow:

1. Agent sends `AgentRegistration`.
2. Platform validates:
   - enrollment token,
   - `agent_id`,
   - plugin instance binding,
   - plugin and protocol compatibility.
3. Platform binds the live stream to the target plugin instance.
4. Platform updates instance metadata and runtime status.

Heartbeat flow:

1. Agent sends `Heartbeat` periodically.
2. Platform updates `last_seen_at`, `runtime_status`, `agent_info`, and capability snapshot.
3. Missed heartbeat transitions the instance to `offline`.
4. Reconnection restores `online` without reinstall.

### 3. Config Delivery

Configuration is generated and versioned by the platform.

Rules:

- Initial config is generated during SSH installation.
- Subsequent changes are versioned in `host_plugin_config_revisions`.
- Runtime config updates are sent through `ConfigUpdate`.
- The platform waits for `Ack` before marking the config revision delivered.
- Failed config delivery leaves the previous delivered config as authoritative.

### 4. Metric Ingestion

The plugin reports `MetricBatch` messages through the agent stream.

Platform responsibilities:

- resolve the batch to host, plugin instance, and session,
- validate metric schema and field types,
- normalize tags and fields,
- map accepted metrics into the platform's host observability pipeline.

Rejected metrics should not terminate the entire connection unless protocol invariants are broken.

### 5. AI Sandbox Execution

AI host execution no longer directly executes over SSH when the target is a managed host.

Execution flow:

1. AI tool resolves the target host.
2. The platform checks for an online plugin instance with the required execution capability.
3. The platform converts the request into `ExecuteCommand` or `ExecuteScript`.
4. The request includes:
   - task ID
   - timeout
   - environment
   - sandbox resource limits
   - output limit
5. Agent streams back `ExecOutput`.
6. Agent completes with `ExecResult`.
7. The platform persists audit data and returns the final execution result to the AI runtime.

If no qualified plugin instance is online, execution is denied. There is no SSH fallback in this design.

### 6. Upgrade, Reinstall, And Uninstall

All lifecycle changes reuse the plugin task framework.

Upgrade and reinstall:

- continue to use SSH orchestration,
- avoid relying on the plugin to replace its own running binary,
- preserve task logs and rollback points where practical.

Uninstall:

1. Mark instance `runtime_status=draining`.
2. Reject new AI execution requests.
3. Stop the service and remove installed artifacts through SSH.
4. Mark the instance `runtime_status=uninstalled`.

This avoids removing a plugin while it is still serving active execution requests.

## API And UI Implications

### Backend API Additions

Expected additions include:

- plugin catalog queries
- host create or host update support for selected plugin installs
- host plugin instance list and detail queries
- install, upgrade, reinstall, and uninstall task endpoints
- plugin task log queries
- opsagent connectivity and metrics intake endpoints internal to the gRPC service

### Frontend Changes

Minimum required UI changes:

- host creation flow includes plugin selection, version selection, and install intent
- host detail shows installed plugin instances, version, runtime state, heartbeat, and recent task status
- plugin task views expose install and upgrade progress logs
- AI or host execution surfaces explain "plugin required" failures clearly

The first UI iteration can remain focused on `opsagent`, while the backend model stays general.

## Security And Permission Model

This integration changes AI execution semantics and therefore needs explicit security boundaries.

### Platform Authorization

Users must satisfy both:

- host execution or relevant host operation permissions
- AI runtime or tool invocation permissions already enforced by the platform

### Capability Enforcement

The platform must not treat the plugin as a blind remote shell.

Before dispatching execution, it must verify:

- target host is known,
- target plugin instance is installed and online,
- requested execution type is allowed by the plugin capability set,
- timeout, CPU, memory, PID, network, and output constraints are attached.

### Sandbox Contract

The platform is responsible for policy and request shaping.
The plugin is responsible for runtime sandbox enforcement.

This keeps the approval and security model consistent:

- platform decides whether the action may run,
- plugin ensures the action runs inside a bounded environment.

## Error Handling

Errors should be grouped by domain so UI, AI, and operations flows can reason about them predictably.

### Install Errors

- package missing
- architecture mismatch
- SSH unreachable
- config upload failure
- install script failure

Result:

- plugin task fails
- instance keeps a failed install state
- task logs retain execution evidence

### Registration Errors

- invalid token
- unknown `agent_id`
- host or instance binding mismatch
- protocol or version incompatibility

Result:

- connection rejected
- instance remains `pending_online` or `offline`

### Runtime Errors

- missed heartbeat
- stream disconnect
- config update `Ack` failure

Result:

- instance transitions to `offline` or `degraded`
- no implicit reinstall

### Execution Errors

- capability missing
- sandbox refusal
- timeout
- truncated output
- process killed
- stream interruption during run

Result:

- AI execution returns a structured failure
- audit records include reason and runtime metadata

### Metric Intake Errors

- malformed metric payload
- unsupported type
- invalid field value

Result:

- reject the offending batch or sample
- preserve connection health unless protocol safety requires disconnect

## Audit Requirements

The platform should record:

- plugin install, upgrade, reinstall, and uninstall requests
- task status transitions and task logs
- agent registration accept or reject decisions
- heartbeat status transitions
- config update versions and acknowledgements
- AI execution request metadata
- sandbox execution result metadata

For AI execution, record at least:

- requester
- host ID
- plugin instance ID
- agent task ID
- execution type
- sandbox limits
- exit code
- timeout or kill flags
- decision reason when denied

## Testing Strategy

### 1. Model And State Tests

Cover:

- plugin instance lifecycle transitions
- task retries
- config revision delivery state changes
- runtime status transitions caused by heartbeat loss and reconnection

### 2. gRPC Contract Tests

Cover:

- registration success and rejection paths
- heartbeat updates
- metric batch ingestion
- command execution dispatch
- script execution dispatch
- config update acknowledgements

### 3. Install Orchestration Tests

Cover:

- architecture-based package selection
- config generation
- SSH upload and install command execution
- upgrade and uninstall task behavior
- task log persistence

### 4. AI Routing Integration Tests

Cover:

- online plugin instance causes execution to route through agent
- missing plugin instance denies execution
- offline instance denies execution
- capability mismatch denies execution
- no SSH fallback occurs in AI execution code paths

## Tradeoffs And Rationale

Why this design is preferred:

- It matches the requested product model: a real plugin framework, not an `opsagent` exception path.
- It preserves operational pragmatism by continuing to use SSH for bootstrap and recovery.
- It improves security by making AI execution consistently sandbox-bound and plugin-required.
- It keeps the door open for future plugins without forcing a second refactor of host or AI models.

Main cost:

- more up-front modeling and orchestration work than a direct `opsagent` embedding into host records.

This cost is intentional. The alternative would optimize for the first plugin and make the second plugin expensive.

## Open Implementation Notes

- Existing untracked `docs/platform-integration-guide.md` and `proto/agent.proto` are treated as source inputs for this design, not as implementation targets in the spec itself.
- The platform should generate Go bindings from the checked-in `proto` source at implementation time and keep protocol ownership explicit.
- The first implementation may expose `opsagent`-specific UI labels while keeping backend entities plugin-generic.
