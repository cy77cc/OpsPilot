---
name: ops-triage-workflow
description: Use when investigating incidents, alerts, or system health issues across Kubernetes, host, CI/CD, or monitoring domains. Provides a structured diagnostic workflow.
---

# Ops Triage Workflow

A structured approach to investigating incidents and system health issues. Follow this workflow to systematically identify root causes and recommend actions.

## 5-Step Diagnostic Workflow

### 1. Assess Scope
- What domain is affected? (Kubernetes, host, CI/CD, monitoring, service)
- Which services/hosts are impacted?
- What is the severity? (single host vs multi-service vs cluster-wide)

### 2. Check Symptoms
- **Alerts**: Check active/firing alerts via `monitor_alert` or `monitor_alert_active`
- **Events**: Check Kubernetes events via `k8s_events` or `k8s_get_events`
- **Logs**: Check service/pod logs via `k8s_logs` or `os_get_journal_tail` (if using host skill)
- **Pipeline status**: Check CI/CD via `cicd_pipeline_status` for recent failures

### 3. Correlate Signals
Cross-reference data from multiple domains:

| Signal Combination | Likely Meaning |
|---|---|
| Pod crash + host resource saturation | Resource constraint on underlying node |
| Alert firing + recent deployment | Deployment-induced regression |
| Pipeline fail + deployment target down | Infrastructure blocking CI/CD |
| Multiple pods OOMKilled | Insufficient resource limits |
| Alert + no pod events | Application-level issue (not infra) |

### 4. Identify Root Cause
Classify into one of these categories:
- **Resource constraint**: CPU, memory, disk, network saturation
- **Configuration error**: bad config, missing env vars, wrong image tag
- **Dependency failure**: downstream service unavailable, DB connection lost
- **Deployment regression**: code bug, incompatible API change, missing migration
- **Infrastructure failure**: node crash, network partition, DNS resolution failure
- **Security incident**: unauthorized access, certificate expiry, credential rotation

### 5. Recommend Action
Based on root cause, recommend ONE of:
- **Scale**: increase replicas, adjust resource limits (via `k8s_scale_deployment`)
- **Restart**: rolling restart of affected deployment (via `k8s_restart_deployment`)
- **Rollback**: revert to previous deployment version (via `k8s_rollback_deployment`)
- **Escalate**: issue requires human intervention or deeper investigation

## Cross-Domain Correlation Rules

```
K8s Pod Failure
├── CrashLoopBackOff → check logs + resource limits + recent config changes
├── OOMKilled → check memory limits + heap profiles + memory trends
├── ImagePullBackOff → check image registry + credentials + network
└── Pending → check node resources + taints/tolerations + PVC bindings

Host Degradation
├── High CPU → identify top processes → correlate with scheduled jobs
├── High Memory → check swap → identify memory-heavy processes
├── Disk Full → identify large files → check log rotation
└── Network Issues → check DNS → check firewall rules → check MTU

CI/CD Failure
├── Build failed → check build logs → identify failing step
├── Test failed → check test output → differentiate code vs infra
├── Deploy failed → check target health → check permissions
└── Pipeline stalled → check runner status → check queue depth

Alert Firing
├── Metric threshold → check trend (sudden spike vs gradual drift)
├── Availability alert → check service status → check dependencies
└── Log pattern alert → check log volume → check error rate trend
```

## Output Format

```
## Incident Summary
Affected: <services/hosts>
Severity: <critical/high/medium/low>
Duration: <estimated time since onset>

## Root Cause Analysis
Category: <classification from Step 4>
Evidence: <key findings with data points>
Confidence: <high/medium/low>

## Recommended Action
Action: <scale/restart/rollback/escalate>
Target: <specific deployment/service/host>
Rationale: <why this action>
```
