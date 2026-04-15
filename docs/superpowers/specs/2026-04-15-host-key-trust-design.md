# Host Key Trust Flow Design

Date: 2026-04-15

## Goal

Add a product-level SSH host key trust workflow that keeps strict host key verification enabled while making unknown or changed host keys actionable from the UI and API.

The target outcome is:

- Unknown host keys no longer fail as opaque SSH errors
- Users with permission can explicitly trust a host key and retry the original action
- Host key changes remain blocked until explicitly confirmed
- OpsPilot keeps runtime SSH verification strict
- Trusted host keys are auditable and visible in the product

## Scope

This design applies to all SSH-backed host flows:

- Host probe during onboarding
- Host credential update
- Host health check
- SSH connectivity check
- Terminal session creation
- File operations

This design does not weaken SSH verification, disable `known_hosts`, or introduce silent trust-on-first-use behavior.

## Recommended Approach

Use a dual-write bridge model:

- Database is the audit and product truth for trusted host keys
- `known_hosts` file remains the runtime verification source used by SSH
- Trust operations write to both DB and `known_hosts`

This keeps the current secure SSH validation path intact while adding structured product behavior, visibility, and rotation controls.

## Data Model

Add a new table, `host_trusted_keys`, with the following fields:

- `id`
- `host_id`
- `host`
- `port`
- `algorithm`
- `fingerprint_sha256`
- `public_key`
- `status`
- `created_by`
- `confirmed_at`
- `last_seen_at`
- `created_at`
- `updated_at`

Status values:

- `trusted`: active trusted key
- `rotated`: previous trusted key superseded by a newer one
- `revoked`: explicitly revoked key

Semantics:

- `host_id` anchors trust to the managed host record
- `host` and `port` preserve the exact SSH target that was trusted
- `public_key` stores the canonical SSH public key line used to write `known_hosts`
- `fingerprint_sha256` is the primary user-visible identifier
- only one `trusted` record should exist per `(host_id, host, port)` at a time

## Backend Architecture

### 1. Structured host key verification errors

Wrap SSH host key verification failures into a structured domain error rather than returning plain SSH strings.

Error classes:

- `ssh_host_key_unknown`
- `ssh_host_key_mismatch`
- `ssh_host_key_revoked`

Each error should carry:

- `host`
- `port`
- `algorithm`
- `fingerprint_sha256`
- `public_key`
- `known_hosts_path`
- `existing_fingerprints` when mismatch occurs

### 2. Shared SSH connection wrapper

Introduce a shared SSH connection path in the host domain that all host SSH entrypoints use.

Responsibilities:

- call SSH client creation
- normalize host key verification failures into structured domain errors
- keep non-host-key SSH failures unchanged

This wrapper is used by:

- probe
- update credentials
- health check
- SSH check
- terminal session
- file handlers

### 3. Trust host key service

Add a host trust service that:

- validates the presented host key payload
- writes trust state to `host_trusted_keys`
- syncs the corresponding `known_hosts` entry
- handles create vs rotate behavior
- emits audit events

### 4. `known_hosts` sync layer

Keep `known_hosts` as the runtime verification source, but make its maintenance controlled by OpsPilot.

Write path rules:

- read existing file
- update only the target `host:port` entry
- deduplicate duplicates
- replace current trusted line on rotate
- write back atomically

Success requires:

1. DB write success
2. `known_hosts` sync success

If either fails, the trust operation fails.

## API Design

### Error response shape

Host-key-related SSH failures should return a structured error payload.

Example:

```json
{
  "code": 2000,
  "msg": "ssh host key verification failed",
  "data": {
    "error_type": "ssh_host_key_unknown",
    "host_key": {
      "host": "118.193.38.89",
      "port": 13012,
      "algorithm": "ssh-ed25519",
      "fingerprint_sha256": "SHA256:7TrbCYGq2IUcTdS07k5rVjLC3xh12hAvkMlmMNElefA",
      "public_key": "ssh-ed25519 AAAA..."
    },
    "trust_action": {
      "action": "/api/v1/hosts/10/trust-host-key",
      "mode": "create"
    }
  }
}
```

For mismatch, include:

- `current_fingerprint_sha256`
- `trusted_fingerprints`
- `mode: rotate`

### Trust endpoint

Add:

- `POST /api/v1/hosts/:id/trust-host-key`

Request:

```json
{
  "host": "118.193.38.89",
  "port": 13012,
  "algorithm": "ssh-ed25519",
  "fingerprint_sha256": "SHA256:...",
  "public_key": "ssh-ed25519 AAAA...",
  "replace_existing": false
}
```

Behavior:

- verify host exists
- verify submitted host key matches the currently observed host key
- create trust entry for unknown hosts
- rotate existing trust entry for mismatches when explicitly allowed
- sync `known_hosts`
- return success metadata for retry

### Trusted key history endpoint

Add:

- `GET /api/v1/hosts/:id/trusted-host-keys`

Returns:

- current trusted key
- prior rotated keys
- who confirmed them
- when they were confirmed

## Frontend Design

### Shared UX rules

When a host-key-structured error is returned:

- show a dedicated trust modal, not a generic error toast
- display host, port, algorithm, and SHA256 fingerprint
- explain whether this is first trust or a changed fingerprint
- allow a single explicit confirm action
- on success, retry the original operation exactly once
- do not enter automatic retry loops

### Unknown host key flow

For `ssh_host_key_unknown`:

- show a confirmation modal
- explain that this is the first observed fingerprint for this SSH target
- require explicit confirmation
- after success, retry the original operation once

### Mismatch flow

For `ssh_host_key_mismatch`:

- show a stronger warning
- display old and new fingerprints
- default to blocked state
- allow explicit replacement only when permitted
- after trust rotation succeeds, retry once

### Coverage

Integrate the shared modal flow into:

- Host onboarding probe
- Host detail health check
- Host detail credential update
- SSH check
- Terminal startup
- File access/edit/delete flows

## Permissions

Recommended permission model:

- first trust: `host:write` or `host:*`
- fingerprint replacement: `host:trust_host_key` or admin-equivalent elevated path

Reasoning:

- first trust is operational onboarding
- replacement is materially riskier and should be more tightly controlled

If a separate `host:trust_host_key` permission does not exist yet, this change may introduce it as part of the implementation.

## Audit

Every trust state transition writes an audit event with:

- host id
- host and port
- old fingerprint
- new fingerprint
- actor id
- action
- timestamp

Audit actions:

- `trust_created`
- `trust_rotated`
- `trust_revoked`

The host detail page should surface host key trust history once the data endpoint exists.

## Failure Handling

### DB write fails

- trust API returns failure
- do not modify `known_hosts`

### `known_hosts` sync fails

- trust API returns failure
- DB write should be rolled back or compensated

### Original operation retry fails after trust succeeds

- return the new operation error
- do not roll back trust, because the trust action itself was valid

### Non-host-key SSH failures

- preserve current behavior
- do not show trust modal for auth, network, timeout, or command execution errors

## Testing Strategy

### Backend unit tests

- unknown host key maps to structured `ssh_host_key_unknown`
- mismatch maps to structured `ssh_host_key_mismatch`
- trust create writes DB + `known_hosts`
- trust rotate supersedes old trusted key
- `known_hosts` sync failure causes overall trust failure
- non-host-key SSH failures are left unchanged

### Backend integration tests

- probe -> unknown host key -> trust -> retry success
- health check -> unknown host key -> trust -> retry success
- credential update -> unknown host key -> trust -> retry success

### Frontend tests

- host-key error opens trust modal
- modal displays fingerprint details
- confirm calls trust API
- original operation retries exactly once
- mismatch shows stronger warning than unknown

## Rollout Notes

- Existing operators can continue using manually maintained `known_hosts`
- OpsPilot-managed trust writes should be compatible with that file
- This change is additive and does not require weakening existing SSH verification

## Out of Scope

- Automatic trust-on-first-use without confirmation
- Disabling host key verification for local or private networks
- Full replacement of `known_hosts` runtime verification with DB-only verification
