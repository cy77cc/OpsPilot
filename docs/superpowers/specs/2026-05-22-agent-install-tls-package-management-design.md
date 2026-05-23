# Agent Install Management, TLS, and Package Management Design

## Problem

The OpsPilot platform and opsagent have a working gRPC protocol, plugin catalog, and SSH-based install orchestration. However, three critical gaps prevent a usable end-to-end agent deployment:

1. **No TLS certificate generation**: The platform gRPC server runs insecure (`opsagent.insecure: true`), but the agent refuses non-TLS connections. There is no code to generate per-host mTLS certificates during installation.

2. **No install/uninstall on existing hosts**: Agent installation is only available during host creation (`PluginInstalls` in the create request). There is no way to install or uninstall an agent on an already-added host from the UI or API.

3. **No package management**: The `host_plugin_versions` table has a `package_path` column, but there is no API to upload, list, or manage agent packages. The path must be manually set in the database.

## Goals

- Platform acts as a private CA for opsagent mTLS, issuing per-host client certificates during installation.
- Users can install and uninstall the agent at any time from the host detail page.
- Users can upload and manage agent packages (tar.gz) through the platform UI.
- The platform gRPC server supports mTLS with an optional insecure mode for development.

## Non-Goals

- Certificate revocation list (CRL) distribution to agents (future work)
- Automatic agent self-upgrade
- Plugin marketplace or third-party plugin upload

## Architecture

### New Components

```
internal/pki/opsagent/        -- OpsAgent CA manager
internal/modules/hostplugin/  -- Extended with package mgmt + install/uninstall APIs
```

### Modified Components

```
internal/modules/opsagent/logic/server.go  -- mTLS support
internal/modules/hostplugin/logic/         -- TLS-aware install flow
configs/config.yaml                        -- TLS config section
web/src/pages/Hosts/Detail/tabs/PluginTab  -- Install/Uninstall UI
```

## Domain Model Changes

### New Table: `opsagent_ca`

Stores the platform's OpsAgent root CA (one row, singleton).

| Column | Type | Description |
|--------|------|-------------|
| id | bigint PK | |
| ca_cert_pem | text | CA certificate (PEM) |
| ca_key_pem | text | CA private key (PEM, encrypted at rest) |
| created_at | timestamp | |

### New Table: `opsagent_host_certificates`

Per-host client certificates issued by the CA.

| Column | Type | Description |
|--------|------|-------------|
| id | bigint PK | |
| host_id | bigint FK | |
| instance_id | bigint FK | |
| serial_number | text | Certificate serial |
| cert_pem | text | Client certificate (PEM) |
| key_pem | text | Client private key (PEM, encrypted at rest) |
| not_after | timestamp | Certificate expiry |
| revoked | boolean | |
| created_at | timestamp | |

### New Table: `host_plugin_packages`

Uploaded agent packages.

| Column | Type | Description |
|--------|------|-------------|
| id | bigint PK | |
| plugin_key | text | e.g. "opsagent" |
| version | text | Semver string |
| arch | text | "amd64" or "arm64" |
| filename | text | Original filename |
| storage_path | text | Local disk path |
| checksum | text | SHA256 |
| size_bytes | bigint | |
| uploaded_by | bigint | User ID |
| created_at | timestamp | |

## Key Flows

### 1. CA Initialization

On first agent install request, if no CA exists:

1. Generate RSA 4096 self-signed CA certificate (reuse `internal/pki.GenerateCA` logic).
2. Encrypt CA private key with platform encryption key.
3. Persist to `opsagent_ca` table.
4. Cache in memory for cert signing performance.

Subsequent installs reuse the existing CA.

### 2. Install Agent on Existing Host

```
POST /api/v1/hosts/:id/plugins/install
Body: { "plugin_key": "opsagent", "version": "v1.0.0" }
```

Flow:

1. Validate host exists and is reachable (SSH credentials present).
2. Ensure OpsAgent CA is initialized.
3. Look up `host_plugin_versions` for matching version + host arch.
4. Generate mTLS client certificate:
   - CN = agent_id (e.g. `opsagent-host-1-instance-5`)
   - SAN = host IP
   - Validity = 1 year
   - Sign with OpsAgent CA
5. Create `host_plugin_instance` (install_status=pending).
6. Generate full agent config YAML:
   - `grpc.server_addr` resolved from `opsagent.host` + `opsagent.port` in platform config. If host is `0.0.0.0`, use the platform's external IP (from `server.host` config or auto-detect via hostname).
   - `auth.bearer_token` generated as 32-char random alphanumeric string via `crypto/rand`.
   ```yaml
   agent:
     id: "opsagent-host-1-instance-5"
     name: "opsagent-host-1-instance-5"
     interval_seconds: 15

   grpc:
     server_addr: "{resolved_platform_addr}:9090"
     enroll_token: "{agent_id}"
     mtls:
       cert_file: "/etc/opsagent/certs/client.crt"
       key_file: "/etc/opsagent/certs/client.key"
       ca_file: "/etc/opsagent/certs/ca.crt"

   auth:
     enabled: true
     bearer_token: "{random_32char_alphanumeric}"
   ```
7. Create `host_plugin_config_revision` with config YAML.
8. Create `host_plugin_task` (operation=install).
9. Persist certificate to `opsagent_host_certificates`.
10. Return task_id to caller.
11. Async: execute install task via SSH:
    - Upload tarball to `/tmp/opspilot/plugins/{instanceID}/`
    - Upload client cert + key + CA cert to `/etc/opsagent/certs/`
    - Upload config to `/etc/opsagent/config.yaml`
    - Extract tarball, run `install.sh`
    - `systemctl start opsagent`
    - Update instance: install_status=succeeded, runtime_status=pending_online

### 3. Uninstall Agent

```
POST /api/v1/hosts/:id/plugins/uninstall
Body: { "instance_id": 5 }
```

Flow:

1. Validate instance exists and belongs to host.
2. Update instance: runtime_status=draining (reject new AI executions).
3. Create `host_plugin_task` (operation=uninstall).
4. Async: SSH to host:
   - `systemctl stop opsagent`
   - Run `uninstall.sh` (from the original package)
   - Remove `/etc/opsagent/` directory
   - Remove `/tmp/opspilot/plugins/{instanceID}/`
5. Mark certificate as revoked.
6. Update instance: install_status=uninstalled, runtime_status=uninstalled.
7. Close any active gRPC session for this agent.

### 4. Package Upload

```
POST /api/v1/host-plugins/packages/upload
Content-Type: multipart/form-data
Fields: plugin_key, version, arch, file (tar.gz)
```

Flow:

1. Validate plugin_key exists in catalog.
2. Validate arch is amd64 or arm64.
3. Compute SHA256 checksum.
4. Save file to storage path: `storage/packages/{plugin_key}/{version}/{arch}/{filename}`.
5. Create `host_plugin_packages` record.
6. Auto-create `host_plugin_versions` record linking to the package.

### 5. Platform gRPC Server mTLS

When `opsagent.tls.enabled: true`:

1. Load OpsAgent CA from database.
2. Generate server certificate signed by the same CA.
3. Configure gRPC server with:
   - TLS certificate (server.crt + server.key)
   - Client CA pool (OpsAgent CA cert)
   - RequireClientCert: true
4. During `Connect` registration, validate client cert CN matches agent_id.

When `opsagent.tls.enabled: false` (development):
- Run gRPC server in insecure mode (current behavior).
- Agent must also be configured with `grpc.insecure: true` or skip TLS verification.

## API Specification

### Package Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/host-plugins/packages/upload` | Upload package |
| GET | `/host-plugins/packages` | List packages |
| DELETE | `/host-plugins/packages/:id` | Delete package |

### Install/Uninstall on Existing Host

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/hosts/:id/plugins/install` | Install agent |
| POST | `/hosts/:id/plugins/uninstall` | Uninstall agent |

### Existing Stubs to Implement

| Method | Endpoint | Current State | Change |
|--------|----------|---------------|--------|
| GET | `/hosts/:id/plugins/instances` | Empty stub | Query `host_plugin_instances` |
| GET | `/host-plugins/tasks/:task_id` | Stub | Query `host_plugin_tasks` |
| GET | `/host-plugins/tasks/:task_id/logs` | Empty stub | Query `host_plugin_task_logs` |

## Config Changes

### Platform config.yaml additions

```yaml
opsagent:
  enable: true
  host: 0.0.0.0
  port: 9090
  tls:
    enabled: false          # true in production
    server_cert: ""         # auto-generated if empty
    server_key: ""          # auto-generated if empty
    ca_cert: ""             # loaded from DB if empty
```

### Agent config template (generated during install)

```yaml
agent:
  id: "{agent_id}"
  name: "{agent_id}"
  interval_seconds: 15
  shutdown_timeout_seconds: 30

server:
  listen_addr: "127.0.0.1:18080"

grpc:
  server_addr: "{platform_ip}:9090"
  enroll_token: "{agent_id}"
  heartbeat_interval: 15
  mtls:
    cert_file: "/etc/opsagent/certs/client.crt"
    key_file: "/etc/opsagent/certs/client.key"
    ca_file: "/etc/opsagent/certs/ca.crt"

auth:
  enabled: true
  bearer_token: "{random_token}"

collector:
  inputs:
    - type: "cpu"
    - type: "memory"
    - type: "disk"
    - type: "net"

sandbox:
  enabled: true
  nsjail_path: "/usr/bin/nsjail"
  base_workdir: "/tmp/opsagent/sandbox"
```

## Frontend Changes

### PluginTab Enhancement

Replace the current read-only table with interactive controls:

- **Install button**: Visible when no opsagent instance exists on the host.
  - Opens modal with version selector (from plugin catalog).
  - Shows install progress (poll task status).
  - Shows task logs on demand.

- **Uninstall button**: Visible when opsagent instance exists.
  - Confirmation dialog warning about agent removal.
  - Shows uninstall progress.

- **Instance list**: Show install_status, runtime_status, health_status, last_heartbeat, version.

### Package Management Page

New page at `/settings/plugin-packages`:

- Upload form: select plugin, version, arch, file.
- Table: list uploaded packages with delete action.

## Security Considerations

1. **CA private key**: Encrypted at rest using platform encryption key (`config.yaml security.encryption_key`). Same pattern as SSH key storage.

2. **Client certificate private key**: Encrypted at rest. Decrypted only during install task execution (SSH upload).

3. **Certificate validity**: 1 year per host cert. Expiry monitoring future work.

4. **Agent identity binding**: Client cert CN = agent_id. Platform validates CN matches registered agent_id during gRPC handshake.

5. **Package storage**: Files stored outside web root. Access only through authenticated API endpoints.

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Host SSH unreachable | Fail install task, log error |
| Package not found for arch | Return 400 with supported arch list |
| CA not initialized | Auto-initialize on first install |
| Instance already exists | Return 409 conflict |
| Install script fails | Mark task failed, preserve logs |
| Agent cert expired | Agent reconnect fails, host shows offline |
| Uninstall on offline host | Best-effort SSH, mark uninstalled even if SSH fails |

## Migration

New SQL migration file:

```sql
-- 20260522_0001_create_opsagent_pki_tables.postgres.sql
CREATE TABLE opsagent_ca ( ... );
CREATE TABLE opsagent_host_certificates ( ... );
CREATE TABLE host_plugin_packages ( ... );
```

## Testing

1. **Unit tests**: CA generation, cert signing, config rendering.
2. **Integration tests**: Install task execution with mock SSH, package upload API.
3. **E2E tests**: Full flow from package upload -> host creation -> agent install -> gRPC registration.

## Implementation Order

1. Database migration (3 new tables)
2. `internal/pki/opsagent/` - CA manager + cert issuance
3. Package management API (upload/list/delete)
4. Install/uninstall API for existing hosts
5. Enhanced config rendering with TLS paths
6. Install task update to include cert upload
7. gRPC server mTLS support
8. Handler implementations (replace stubs)
9. Frontend: PluginTab install/uninstall buttons
10. Frontend: Package management page
