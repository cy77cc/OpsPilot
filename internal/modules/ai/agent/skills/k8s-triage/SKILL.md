---
name: k8s-triage
description: Use when diagnosing Kubernetes issues including pod failures, deployment problems, service connectivity, or cluster resource issues. Guides structured K8s investigation workflow.
---

# Kubernetes Triage Skill

Use this skill to systematically investigate Kubernetes cluster issues. Follow the decision tree for common failure modes.

## Diagnostic Decision Tree

```
                    What's the symptom?
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   Pod failing      Service down     Cluster issue
        │                 │                 │
        ▼                 ▼                 ▼
  Check pod status   Check endpoints   Check nodes
        │                 │                 │
   ┌────┼────┐      ┌────┼────┐      ┌────┼────┐
   │    │    │      │    │    │      │    │    │
  CP  OOM  IMG   DNS  SEL  NET   DISK MEM  CORD
```

## Pod Failure Investigation

### CrashLoopBackOff
1. Get pod logs: `k8s_logs` or `k8s_get_pod_logs`
2. Check events: `k8s_events` filtered by pod name
3. Common causes and next checks:

| Log Pattern | Cause | Next Check |
|-------------|-------|-----------|
| `panic:` / `fatal:` | Application bug | Review recent deployment |
| `connection refused` | Dependency unavailable | Check downstream service |
| `OOMKilled` (in events) | Memory limit too low | Check resource usage trends |
| `Back-off restarting` | Config error / missing env | Check configmap/secret |
| `image pull failed` | Registry/auth issue | Check image credentials |

### OOMKilled
1. Check current resource limits: `k8s_query` with filter for the deployment
2. Check memory usage trend: correlate with monitoring metrics
3. Actions: increase `resources.limits.memory`, fix memory leak, or adjust JVM heap

### Pending
1. Check node resources: `k8s_query` for nodes with `kubectl describe node`
2. Check taints/tolerations: `k8s_query` for pod spec
3. Check PVC bindings: `k8s_list_resources` in namespace for `PersistentVolumeClaim`
4. Common causes:
   - Insufficient CPU/memory on all nodes
   - Missing node selector match
   - PVC not bound (no matching PV)
   - Taint without toleration

### ImagePullBackOff
1. Check image name and tag in deployment spec
2. Check image pull secret: `k8s_query` for `imagePullSecrets`
3. Check registry connectivity (may need host_exec on nodes)

## Service Connectivity Investigation

### Service Not Reachable
1. Check endpoints: `k8s_query` for service with detail view
2. Check pod readiness: pods must be `Ready` to receive traffic
3. Check selector match: service selector must match pod labels
4. Check port mapping: service port → targetPort → containerPort

### DNS Resolution Failure
1. Check CoreDNS pods: `k8s_list_resources` in `kube-system`
2. Check pod DNS config: `k8s_query` for pod with detail
3. Test from within cluster: `nslookup <service>.<namespace>.svc.cluster.local`

## Deployment Investigation

### Deployment Not Rolling Out
1. Check rollout status: `k8s_query` for deployment detail
2. Check events: `k8s_events` for deployment
3. Check new pod status: `k8s_list_resources` in namespace for pods
4. Common blockers:
   - `minReadySeconds` not met (readiness probe failing)
   - Resource quota exceeded
   - PDB (PodDisruptionBudget) blocking
   - Image not available

### Deployment Rollback Needed
1. Check rollout history: `k8s_query` for deployment detail
2. Use `k8s_rollback_deployment` to revert
3. Monitor rollback success: check new pod status

## Cluster Health Check

```
Cluster Health Checklist
├── Node status: all nodes Ready?
├── Core components: kube-apiserver, etcd, controller-manager, scheduler healthy?
├── Resource utilization: any nodes > 80% CPU/Mem?
├── Pending pods: any stuck in Pending for > 5 min?
├── Storage: any PVCs in Pending state?
└── Events: any Warning events in last hour?
```

## K8s Diagnostic Commands via Tools

| Investigation | Tool | Parameters |
|--------------|------|-----------|
| Pod status | `k8s_query` | kind=Pod, name, namespace |
| Pod logs | `k8s_get_pod_logs` | podName, namespace, container |
| Events | `k8s_events` | namespace, kind, name |
| Deployment status | `k8s_query` | kind=Deployment, name, namespace |
| Service endpoints | `k8s_query` | kind=Service, name, namespace |
| Node status | `k8s_query` | kind=Node |
| Resource listing | `k8s_list_resources` | namespace, kind |

## Output Format

```
## K8s Issue: <brief description>
Namespace: <namespace>
Resource: <kind/name>

## Diagnosis
Status: <current state>
Events: <key events with timestamps>
Logs: <relevant log excerpts>
Root Cause: <identified or suspected cause>

## Recommended Action
Action: <specific k8s operation>
Command: <tool + parameters to use>
Risk: <low/medium/high>
```
