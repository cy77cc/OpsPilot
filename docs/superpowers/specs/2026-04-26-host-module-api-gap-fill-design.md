# Host Module API Gap Fill Design

## Problem

The host detail page frontend was significantly refactored with multiple new tabs (Process, Service, Disk, Network, Package, Monitor, Alarm, OperationLog). While the frontend components are fully wired to call backend APIs, 7 backend handlers still return hardcoded mock data with `// TODO` comments. Additionally, the NetworkTab contains a hardcoded route table, the Metrics handler returns zero for disk I/O and network I/O fields, and credential usage stats are empty.

## Scope

### Backend Changes

| # | Endpoint | Current | Target |
|---|----------|---------|--------|
| 1 | `GET /hosts/:id/processes` | 3 mock processes | SSH `ps aux --no-headers`, parse output |
| 2 | `POST /hosts/:id/processes/:pid/kill` | mock string | SSH `kill -<signal> <pid>`, return result |
| 3 | `GET /hosts/:id/services` | 4 mock services | SSH `systemctl list-units --type=service --all --no-pager --no-legend`, parse |
| 4 | `POST /hosts/:id/services/:name/actions` | mock string | SSH `systemctl <action> <name>`, return result |
| 5 | `GET /hosts/:id/disks` | 2 mock partitions | SSH `df -BG --output=source,target,fstype,size,used,avail,pcent`, parse |
| 6 | `GET /hosts/:id/network-interfaces` | 2 mock interfaces | SSH `ip -o addr show` + `ip link show`, parse |
| 7 | `GET /hosts/:id/routes` (new) | none | SSH `ip route show`, parse |
| 8 | `GET /hosts/:id/packages` | 2 mock packages | Detect dpkg/rpm, query, parse |
| 9 | `GET /hosts/:id/metrics` | diskIo/netIn/netOut = 0 | Add `/proc/diskstats` and `/proc/net/dev` sampling |

### Frontend Changes

| # | File | Change |
|---|------|--------|
| 1 | `NetworkTab.tsx` | Wire route table to new `getHostNetworkRoutes` API |
| 2 | `hosts.ts` | Add `getHostNetworkRoutes` method + `hostAction` signal param |

### Credential Stats (deferred)

The credential usage-records, permissions, and usage-stats endpoints return empty because they require an audit/tracking subsystem that doesn't exist yet. Implementing this properly means:
- Creating a credential_usage_records table
- Instrumenting all credential-consuming operations to log usage
- Building aggregation queries

This is a separate concern that should be its own change. The Credentials page already handles empty data gracefully. **Out of scope for this change.**

## Architecture

### SSH Command Pattern

All new handlers follow the established pattern from `host_exec.go`:

1. Parse host ID from URL param
2. Fetch node from DB via `hostService.Get()`
3. Load SSH private key via `loadNodePrivateKey()`
4. Resolve password via `hostService.ResolveNodeSSHPassword()`
5. Create SSH client via `sshclient.NewSSHClient()`
6. Run command via `sshclient.RunCommand()`
7. Parse stdout, map to response struct, return JSON
8. Handle SSH host key trust errors via `writeHostKeyPayloadIfNeeded()`

All commands run with a 10-second context timeout to prevent hanging.

### Command-to-Struct Mapping

Each handler runs a specific system command and parses the output:

**Processes** — `ps aux --no-headers`
- Split each line by whitespace (first 10 fields are fixed)
- Remaining content is the command string

**Kill Process** — `kill -<signal> <pid>`
- Default signal: 15 (SIGTERM)
- Optional `signal` query param for custom signal

**Services** — `systemctl list-units --type=service --all --no-pager --no-legend`
- 5 columns: UNIT, LOAD, ACTIVE, SUB, DESCRIPTION
- Description may contain spaces, so split on first 4 fields

**Service Action** — `systemctl <action> <name>` where action is start/stop/restart/reload/status

**Disks** — `df -BG --output=source,target,fstype,size,used,avail,pcent --total`
- Parse `G` suffix from numeric fields, convert to string representation
- Skip the header line and the total line

**Network Interfaces** — Two commands:
- `ip -o addr show` → name, IP, CIDR
- `ip link show` → name, MAC, state, MTU
- Merge by interface name
- For RX/TX: read `/sys/class/net/<name>/statistics/rx_bytes` and `tx_bytes`

**Routes** — `ip route show`
- Parse `default via <gw> dev <iface>` and `<subnet> via <gw> dev <iface>` patterns

**Packages** — OS detection:
- Debian/Ubuntu: `dpkg-query -W --showformat='${Package}\t${Version}\t${Architecture}\t${Status}\n'`
- RHEL/CentOS/Rocky: `rpm -qa --queryformat '%{NAME}\t%{VERSION}\t%{ARCH}\n'`
- Use OS info already stored in the node record

**Metrics Enhancement** — Two additional data points per snapshot:
- Disk I/O: `cat /proc/diskstats` — sum reads/writes across all non-loop devices
- Network I/O: `cat /proc/net/dev` — sum RX/TX bytes across all non-lo interfaces

### New API Types

Add to `api/host/v1/host.go`:

```go
// RouteItem represents a routing table entry.
type RouteItem struct {
    Destination string `json:"destination"`
    Gateway     string `json:"gateway"`
    Mask        string `json:"mask"`
    Flags       string `json:"flags"`
    Interface   string `json:"iface"`
    Metric      int    `json:"metric,omitempty"`
}
```

### Error Handling

- SSH connection errors: return `httpx.OK` with `reachable: false` payload (same as `SSHCheck`)
- SSH host key not trusted: return via `writeHostKeyPayloadIfNeeded()`
- Command execution failures: return error with the specific command that failed
- Parse failures: return error with the raw output for debugging

### Route Registration

Add the new route in `internal/modules/host/api/routes.go`:

```go
hostGroup.GET("/:id/routes", h.ListNetworkRoutes)
```

## Testing Strategy

- Manual testing via frontend UI — visit host detail page, verify each tab shows real data
- Verify error cases: offline host, invalid credentials, untrusted SSH key
- Commands should work on both Debian and RHEL family systems

## Files Changed

### Backend
- `internal/modules/host/handler/host_resource_details.go` — Rewrite 7 handlers with real SSH implementations
- `internal/modules/host/handler/host_query.go` — Enhance `Metrics()` with disk I/O and network I/O
- `internal/modules/host/api/routes.go` — Add `GET /:id/routes` route
- `api/host/v1/host.go` — Add `RouteItem` struct

### Frontend
- `web/src/pages/Hosts/Detail/tabs/NetworkTab.tsx` — Wire route table to API
- `web/src/api/modules/hosts.ts` — Add `getHostNetworkRoutes` method
