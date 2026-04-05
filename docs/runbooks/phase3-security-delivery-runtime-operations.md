# Phase 3 Security Delivery Runtime Operations Runbook

## Prerequisites
- Phase 3 services deployed: admission, gitops, runtime handlers.
- Governance tables available: `operation_approvals`, `operation_audits`.
- Phase 3 tables migrated: `admission_policies`, `admission_exemptions`, `image_scan_reports`, `gitops_app_releases`, `runtime_security_events`, `runtime_disposal_actions`.
- Monitoring baseline connected: Prometheus scrape for `phase3_*` metrics.

## Release Flow
1. Register or update admission policy via `POST /api/v1/clusters/:id/admission/policies`.
2. Trigger app sync via `POST /api/v1/clusters/:id/apps/:name/sync`.
3. Verify release record persisted in `gitops_app_releases` with `sync_result=succeeded`.
4. Confirm operation audit entry exists for `gitops.sync` action.

## Rollback Flow
1. Trigger rollback via `POST /api/v1/clusters/:id/apps/:name/rollback` with `rollback_ref`.
2. Verify rollback release record persisted with `sync_result=rolled_back`.
3. Confirm operation audit entry exists and links rollback action/result.
4. If rollback fails, open incident and freeze further sync until breaker is reset.

## Runtime Containment
1. Ingest event via `POST /api/v1/clusters/:id/security/events/ingest`.
2. Inspect alerts via `GET /api/v1/clusters/:id/security/alerts`.
3. Execute containment via `POST /api/v1/clusters/:id/security/alerts/:alert_id/contain`.
4. Validate `runtime_disposal_actions` contains mode/result/audit_id.
5. Resolve alerts via `POST /api/v1/clusters/:id/security/alerts/:alert_id/resolve`.

## external_managed Downgrade
- `platform_managed`: containment runs in `auto` mode with approval-gated execution.
- `external_managed`: containment is downgraded to `suggest_only` and still records `OperationAudit` + disposal action.
- No direct workload mutation is attempted on external-managed clusters.

## SLO/SLA Verification
- Admission latency p95 target: `< 800ms`.
- Runtime detect-to-alert p95 target: `< 30s`.
- Containment success rate target (`platform_managed`): `>= 98%`.
- Suggest-only issuance rate (`external_managed`): `100%` for containment requests.

## Incident Checklist
- Capture cluster source (`platform_managed` vs `external_managed`).
- Capture approval ticket and audit id for failed or rejected operations.
- Attach latest drift result and circuit breaker status for GitOps incidents.
- Attach runtime event payload + disposal action row for containment incidents.
