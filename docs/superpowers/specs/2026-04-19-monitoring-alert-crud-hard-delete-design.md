# Monitoring Alerting Full CRUD (With Hard Delete) Design

- Date: 2026-04-19
- Owner: Codex + User
- Status: Draft Approved For Planning

## 1. Context

The monitoring alerting backend already supports most create/update/read operations for:

- Alert rules
- Alert channels
- Severity routes
- Rule-channel bindings

Frontend pages were partially read-oriented and lacked full edit flows. Deletion semantics were also incomplete on the backend for core resources.

This design upgrades the monitoring configuration experience to **full CRUD** across rule/channel/route/binding, with **hard delete** and **conflict protection** (no automatic cascade delete).

## 2. Goals

1. Deliver full CRUD for:
- `AlertRule`
- `AlertNotificationChannel`
- `AlertSeverityRoute`
- `AlertRuleChannelBinding`

2. Support both scopes in UI and API:
- Global (`project_id` absent)
- Project (`project_id` present)

3. Implement hard delete with safe conflict behavior:
- Reject deletes that would break references
- Return structured blockers so UI can guide user actions

4. Keep existing route structure and page IA (incremental enhancement).

## 3. Non-Goals

1. No automatic cascade delete for rule/channel removal.
2. No redesign of monitoring navigation or full-page state architecture.
3. No schema redesign beyond what is required for CRUD endpoints and checks.

## 4. Architecture Approach (Recommended A)

Use incremental enhancement in existing pages and handlers:

- Backend: add missing `DELETE` and single-item create/update endpoints where absent.
- Frontend: add create/edit/delete UI actions directly inside current monitor config pages.
- Keep list + modal/drawer interaction model.
- Keep existing API module (`web/src/api/modules/monitoring.ts`) as integration boundary.

## 5. Backend API Contract

## 5.1 AlertRule

- Existing:
1. `POST /alert-rules`
2. `PUT /alert-rules/:id`
3. `GET /alert-rules`
4. `GET /alert-rules/effective`
- Add:
1. `DELETE /alert-rules/:id`

Delete rules:
- If referenced by any `alert_rule_channel_bindings.rule_id`, return `409` with blockers.
- If not found, return `404`.

## 5.2 AlertNotificationChannel

- Existing:
1. `POST /alert-channels`
2. `PUT /alert-channels/:id`
3. `GET /alert-channels`
4. `POST /alert-channels/test`
- Add:
1. `DELETE /alert-channels/:id`

Delete channels:
- If referenced by `alert_rule_channel_bindings.channel_id`, return `409`.
- If referenced by `alert_severity_routes.channel_ids_json`, return `409`.
- If not found, return `404`.

## 5.3 AlertSeverityRoute

- Existing:
1. `GET /alert-routing/severity`
2. `PUT /alert-routing/severity` (bulk replace)
- Add:
1. `POST /alert-routing/severity` (single create)
2. `PUT /alert-routing/severity/:id` (single update)
3. `DELETE /alert-routing/severity/:id` (single delete)

## 5.4 AlertRuleChannelBinding

- Existing:
1. `GET /alert-rules/:id/channels`
2. `PUT /alert-rules/:id/channels` (bulk replace)
- Add:
1. `POST /alert-rules/:id/channels` (single create)
2. `PUT /alert-rules/:id/channels/:channel_id` (single update)
3. `DELETE /alert-rules/:id/channels/:channel_id` (single delete)

Binding operations are scoped by `(rule_id, channel_id, project_id)` where:
- `project_id` absent => global binding
- `project_id` present => project binding

## 5.5 Auth, Status Codes, Error Payload

Write operations require: `monitoring:write`.

Status handling:
1. `200`: success
2. `404`: target not found
3. `409`: conflict due to references
4. `500`: server/internal

`409` response includes:

```json
{
  "code": 409,
  "msg": "resource has references",
  "data": {
    "blockers": [
      { "type": "binding", "count": 2, "samples": ["rule_id=7,channel_id=1001"] },
      { "type": "severity_route", "count": 1, "samples": ["route_id=18"] }
    ]
  }
}
```

## 6. Backend Validation Rules

1. `scope=project` requires `project_id`.
2. `severity` limited to: `critical`, `warning`, `info`.
3. `priority` must be positive integer.
4. `channel_ids` deduplicated and filtered for valid positive IDs.
5. Binding delete updates exactly one scoped binding; zero affected rows => `404`.

## 7. Frontend UX Design

## 7.1 RulesConfigPage

Add operations:
1. Create rule
2. Edit rule
3. Delete rule
4. Manage bindings (add/edit/delete binding)

Interaction:
- Table row actions with modal/drawer forms.
- Delete via `Popconfirm`.
- On success: toast + reload list while preserving current filters/pagination.

## 7.2 ChannelsConfigPage

Add operations:
1. Create channel
2. Edit channel
3. Delete channel
4. Test send (existing, retained)

Delete conflict (`409`) shows blocker detail modal.

## 7.3 RoutingConfigPage

Add operations:
1. Create route
2. Edit route
3. Delete route

Use single-item CRUD endpoints for normal UI actions.
Keep bulk replace endpoint for optional future batch mode.

## 7.4 Binding Management

Bindings are managed in rules context (not separate nav page):

1. Create binding
2. Update binding (`priority`, `enabled`)
3. Delete binding (unbind)

## 7.5 Scope Selector

Each config page has scope switch:
1. Global
2. Project (`project_id` required)

All read/write requests include project scope accordingly.

## 8. Frontend Error/State Handling

1. Success => `message.success` + list refresh.
2. `404` => prompt "resource no longer exists" + refresh.
3. `409` => `Modal.error` with parsed blocker list.
4. Disable submit/delete buttons while request is in flight.
5. Preserve UI state (scope/filter/page) across refresh after mutation.

## 9. File Impact

Backend:
- `internal/modules/monitoring/api/routes.go`
- `internal/modules/monitoring/handler/handler.go`
- `internal/modules/monitoring/logic/logic.go`
- `internal/modules/monitoring/logic/routing_policy.go`
- related handler/logic tests

Frontend:
- `web/src/api/modules/monitoring.ts`
- `web/src/pages/Monitor/RulesConfigPage.tsx`
- `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- `web/src/pages/Monitor/RoutingConfigPage.tsx`
- related page/api tests

## 10. Test Matrix

Backend tests:
1. CRUD success for rule/channel/route/binding
2. `404` for missing targets
3. `409` blocker behavior for delete conflicts
4. Scope isolation for global vs project operations

Frontend tests:
1. API path/payload mapping for all CRUD methods
2. Create/edit/delete action flow in each page
3. `409` blocker rendering and guidance
4. Scope switch affecting request payload/query

Regression:
1. Existing monitor list/read behaviors remain stable
2. Existing alert receiver + AI fanout flows unaffected

## 11. Rollout Strategy

1. Add backend endpoints + tests first.
2. Wire frontend API methods.
3. Add page-level create/edit/delete UX incrementally (Rules -> Channels -> Routing -> Binding).
4. Run targeted monitor + backend regression suites.
5. Merge with no feature flag (scope is operational tooling and backward compatible).

## 12. Risks and Mitigations

1. Risk: false positives in channel-in-route reference checks (JSON parsing edge cases)
- Mitigation: centralized parser in logic + explicit test fixtures for mixed/empty JSON

2. Risk: scope confusion (global vs project)
- Mitigation: visible scope control + clear payload/query handling + scope-specific tests

3. Risk: UX friction on delete conflicts
- Mitigation: structured `409.blockers` with actionable dependency hints
