---
name: host-diagnostic
description: Use when investigating host performance issues, resource saturation, service failures, or system health on a Linux host. Covers CPU, memory, disk, network, process, and container runtime diagnostics.
---

# Host Diagnostic Skill

Use this skill to systematically diagnose Linux host issues. Always start with the least invasive check first.

## Diagnostic Command Reference

| Symptom | Command | What to Look For |
|---------|---------|-----------------|
| CPU 饱和 / 负载高 | `cat /proc/loadavg && uptime` | load > nCPU, high user/sys time |
| 内存不足 | `cat /proc/meminfo \| head -20 && free -m` | Available < 10%, high swap usage |
| 磁盘满 | `df -h` | Usage > 90%, inode exhaustion |
| 网络异常 | `cat /proc/net/dev && ss -ltn` | Dropped packets, port binding issues |
| 进程异常 | `ps aux --sort=-%cpu \| head -11` | Zombie processes, run-away processes |
| 服务故障 | `journalctl -u <service> -n 200 --no-pager` | Error/fatal log patterns, restart loops |
| 容器状态 | `docker ps --format '{{.ID}} {{.Image}} {{.Status}}'` | Exited containers, unhealthy status |
| 容器运行时 (containerd) | `ctr -n k8s.io containers list` | Container state and image info |

## Workflow: Host Health Check

1. **Identify the host** — use `host_list_inventory` to find valid targets (by ID, IP, or hostname)
2. **Check symptoms** — call `host_exec` with the appropriate diagnostic command from the table above
3. **Analyze results** — look for resource saturation, failed services, error signatures
4. **Cross-reference** — if multiple hosts affected, check for common patterns (same service, same deployment)
5. **Summarize** — return structured findings: issue, evidence, impact, suggested next check

## Output Rules

- **Never dump raw output** — summarize into actionable findings
- **Use structured format**:
  ```
  Host <ID/IP> - <issue>
  Evidence: <key metrics>
  Impact: <affected services/users>
  Confidence: high/medium/low
  Next check: <recommended follow-up>
  ```
- **Flag critical thresholds**: CPU > 90%, Mem < 5%, Disk > 90%
- **Note recurring patterns**: restart loops, repeated errors, correlation with deployment time

## Common Failure Patterns

| Pattern | Likely Cause | Next Check |
|---------|-------------|------------|
| High load + low CPU% | I/O wait, disk bottleneck | `iostat -x 1 5` |
| OOM in logs | Memory leak, insufficient allocation | `dmesg \| grep -i oom` |
| Service restart loop | Config error, dependency unavailable | `journalctl -u <svc> --since "1 hour ago"` |
| Port not listening | Service crashed, firewall rule | `ss -ltnp \| grep <port>` |
| Container exited | Image pull failure, health check fail | `docker inspect <container>` |
