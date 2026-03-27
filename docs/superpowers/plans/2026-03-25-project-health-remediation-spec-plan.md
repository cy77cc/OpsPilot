# Project Health Remediation Governance Spec Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a pure-English superpowers remediation spec that converts the project health assessment into enforceable, testable, and auditable requirements across backend, frontend, and operations.

**Architecture:** Keep one normative spec document as the source of truth for remediation governance. Build requirement groups around runtime safety, security hygiene, quality gates, maintainability controls, frontend health visibility, and auditability. Preserve a direct trace from assessment findings to policy-enforced scenarios and release-blocking behavior.

**Tech Stack:** Markdown, superpowers spec conventions (`ADDED Requirements` / `Requirement` / `Scenario`), Mermaid, git

---

## File Structure

- Create: `docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`
  - Final normative spec with enforceable requirement/scenario language.
- Modify: `docs/superpowers/specs/2026-03-25-project-health-remediation-governance-design.md`
  - Keep design/spec boundary explicit and add pointer to final spec.
- Optional Modify: `openspec/specs/project-health-assessment/SUMMARY.md`
  - Add one link line to the new superpowers spec for discoverability (only if repository convention allows cross-linking).

## Chunk 1: Draft Normative Spec Skeleton

### Task 1: Create the superpowers spec skeleton with strict section contracts

**Files:**
- Create: `docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`

- [ ] **Step 1: Write the file header and purpose block**

```markdown
# Project Health Remediation Governance Spec

## Purpose
<one-paragraph normative purpose>

## ADDED Requirements
```

- [ ] **Step 2: Add requirement group placeholders**

```markdown
### Requirement: Runtime safety and error containment
### Requirement: Security baseline and configuration hygiene
### Requirement: Risk-tiered quality and test gates
### Requirement: Maintainability control via structural metrics
### Requirement: Frontend health visibility and operator experience
### Requirement: Compliance evidence, timeline, and retention
### Requirement: Policy-as-code enforcement and release protection
```

- [ ] **Step 3: Verify skeleton completeness**

Run: `rg -n "^## ADDED Requirements|^### Requirement:" docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`
Expected: at least 7 requirement headings found.

- [ ] **Step 4: Commit skeleton**

```bash
git add docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md
git commit -m "docs(spec): scaffold health remediation governance spec"
```

## Chunk 2: Encode Enforceable Requirements and Scenarios

### Task 2: Implement runtime and security requirement scenarios

**Files:**
- Modify: `docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`

- [ ] **Step 1: Add runtime safety scenarios with strict fail behavior**

```markdown
#### Scenario: Reject crash-oriented runtime path
- **WHEN** a protected runtime path includes unrecovered panic-on-error behavior
- **THEN** policy evaluation MUST mark violation as blocking for protected release
- **AND** remediation status MUST include machine-readable reason code
```

- [ ] **Step 2: Add security baseline scenarios with strict-origin and secret hygiene controls**

```markdown
#### Scenario: Enforce strict cross-origin boundary
- **WHEN** protected-environment configuration uses wildcard origin policy
- **THEN** startup validation MUST fail or release gate MUST block deployment
```

- [ ] **Step 3: Validate scenario format**

Run: `rg -n "^#### Scenario:|\*\*WHEN\*\*|\*\*THEN\*\*" docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`
Expected: every requirement has at least one scenario with `WHEN/THEN`.

- [ ] **Step 4: Commit runtime/security scenarios**

```bash
git add docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md
git commit -m "docs(spec): add runtime and security governance scenarios"
```

### Task 3: Implement risk-tiered quality and maintainability scenarios

**Files:**
- Modify: `docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`

- [ ] **Step 1: Add Criticality taxonomy section in normative terms**

```markdown
## Criticality and Risk Tiers
- Tier 0 (Critical)
- Tier 1 (High)
- Tier 2 (Moderate)
<criteria and governance mapping>
```

- [ ] **Step 2: Add quality-gate scenarios tied to tiers**

```markdown
#### Scenario: Enforce tier-aware test gate
- **WHEN** Tier 0 module coverage or mandatory suites fail
- **THEN** CI MUST fail and protected release MUST remain blocked
```

- [ ] **Step 3: Add multi-metric maintainability scenarios**

```markdown
#### Scenario: Block structural regression beyond threshold
- **WHEN** complexity or coupling metrics exceed policy limits for protected modules
- **THEN** policy evaluation MUST mark violation and require remediation plan reference
```

- [ ] **Step 4: Commit quality/maintainability section**

```bash
git add docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md
git commit -m "docs(spec): add tiered quality and structural maintainability controls"
```

## Chunk 3: Frontend, Policy Engine, and Evidence Loop

### Task 4: Add frontend visibility, policy-as-code, and audit loop scenarios

**Files:**
- Modify: `docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`

- [ ] **Step 1: Add frontend remediation visibility scenarios**

```markdown
#### Scenario: Render blocker state from policy result
- **WHEN** backend compliance status is blocking
- **THEN** frontend MUST render blocking state and reason code mapping
```

- [ ] **Step 2: Add Policy-as-Code requirement and local DX shift-left scenarios**

```markdown
#### Scenario: Local policy check mirrors CI decision
- **WHEN** developer runs local policy check command
- **THEN** decision class and reason codes MUST match CI policy engine output for same input snapshot
```

- [ ] **Step 3: Add observability fallback and evidence retention scenarios**

```markdown
#### Scenario: Preserve compliance event on primary pipeline failure
- **WHEN** compliance event sink is unavailable
- **THEN** system MUST route event to durable fallback queue
- **AND** system MUST emit operator alert
```

- [ ] **Step 4: Add Mermaid flow to visualize policy feedback loop**

Run: embed a `flowchart LR` showing Local DX checks -> Policy engine -> CI/Release gate -> Frontend/Audit.

- [ ] **Step 5: Commit frontend/policy/evidence section**

```bash
git add docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md
git commit -m "docs(spec): add frontend policy loop and compliance evidence scenarios"
```

## Chunk 4: Consistency, Cross-Linking, and Final Validation

### Task 5: Align design doc and final spec references

**Files:**
- Modify: `docs/superpowers/specs/2026-03-25-project-health-remediation-governance-design.md`
- Optional Modify: `openspec/specs/project-health-assessment/SUMMARY.md`

- [ ] **Step 1: Add “Normative Spec” pointer in design doc**

```markdown
## Normative Spec
See: ./2026-03-25-project-health-remediation-governance-spec.md
```

- [ ] **Step 2: (Optional) Add summary cross-link**

```markdown
Governance remediation spec: <relative path>
```

- [ ] **Step 3: Commit cross-link updates**

```bash
git add docs/superpowers/specs/2026-03-25-project-health-remediation-governance-design.md openspec/specs/project-health-assessment/SUMMARY.md
git commit -m "docs: link health assessment artifacts to normative governance spec"
```

### Task 6: Run final doc validation and package handoff

**Files:**
- Verify: `docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`

- [ ] **Step 1: Validate requirement/scenario coverage count**

Run: `rg -n "^### Requirement:|^#### Scenario:" docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`
Expected: each requirement has at least 2 scenarios.

- [ ] **Step 2: Validate required policy terms are present**

Run: `rg -n "Tier 0|Policy-as-Code|durable fallback queue|local policy check|release gate" docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`
Expected: all required governance terms found.

- [ ] **Step 3: Validate no non-English leakage**

Run: `rg -n "[\p{Han}]" docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md`
Expected: no matches.

- [ ] **Step 4: Commit final validation pass**

```bash
git add docs/superpowers/specs/2026-03-25-project-health-remediation-governance-spec.md
git commit -m "docs(spec): finalize project health remediation governance spec"
```

