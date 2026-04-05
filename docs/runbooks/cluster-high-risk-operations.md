# Cluster High-Risk Operations Runbook

适用范围：集群详情页中的高风险动作失败处置，包括节点 `drain`、节点 `remove`、集群 `upgrade`、证书 `renew`。

## 使用原则

1. 先在集群详情页或操作中心记录 `audit_id`、报错信息、失败阶段，再开始人工处置。
2. 先止损，再修复。高风险动作失败时，优先冻结新的变更、暂停自动化扩缩容和批量发布。
3. 所有重试都应在明确失败原因后执行，不要重复点击相同操作。

## Drain 失败恢复

常见症状：
- `PodDisruptionBudget` 阻止驱逐。
- 节点上仍有非 `DaemonSet` Pod 未迁移。
- `emptyDir`、本地盘或长连接任务导致驱逐超时。

恢复步骤：
1. 在操作中心确认失败节点、失败时间、失败消息，记录对应 `audit_id`。
2. 执行 `kubectl get pod -A -o wide --field-selector spec.nodeName=<node>`，确认仍驻留在节点上的 Pod。
3. 执行 `kubectl get pdb -A`，确认是否有 `ALLOWED DISRUPTIONS=0` 的 PDB 阻塞驱逐。
4. 对业务 Pod 做以下二选一处置：
   - 临时扩容或放宽 PDB，确保可驱逐窗口大于 0。
   - 先手工摘流量并迁移有状态或长连接任务。
5. 若失败消息涉及 `emptyDir` 或本地临时数据，确认数据可丢弃后，改用允许删除临时目录数据的策略重试。
6. 若失败消息涉及超时，先检查节点 `NotReady`、网络或 kubelet 状态；必要时先修复节点心跳，再重新执行 drain。
7. 重试前执行 `kubectl describe node <node>`，确认没有新的 `Unschedulable=false`/调度回流异常。

完成判定：
- `kubectl get pod -A --field-selector spec.nodeName=<node>` 仅剩允许保留的 `DaemonSet` Pod 或结果为空。
- 集群详情页中的节点状态与操作中心审计记录一致。

## Remove 失败恢复

常见症状：
- 节点仍为 `Ready`，说明节点未完成摘除或仍被 kubelet 注册。
- 节点尚未完成 drain。
- 云主机、自动伸缩组或负载均衡仍引用该节点。

恢复步骤：
1. 在操作中心记录失败 `audit_id`，确认是否在 drain 之后立即触发 remove。
2. 先确认 drain 已完成；若未完成，回到 Drain 失败恢复章节处理。
3. 执行 `kubectl get node <node> -o wide`，确认节点是否仍处于 `Ready` 或频繁心跳。
4. 从外部依赖中摘除节点：
   - 从负载均衡、入口流量池中移除。
   - 从自动伸缩组或主机池中标记为待下线，避免再次注册。
5. 若节点仍被 kubelet 注册，先在主机侧停止 kubelet/container runtime，再执行 `kubectl delete node <node>` 或通过平台重试移除。
6. 若云主机已销毁但节点对象残留，确认没有控制器会重新创建后，再清理残留 Node 对象。
7. 重试 remove 前，确认监控、告警和 CMDB/主机库存已同步到下线状态。

完成判定：
- `kubectl get node <node>` 返回不存在。
- 节点不再出现在流量池、自动伸缩池、主机池中。
- 操作中心显示最终移除成功记录。

## Upgrade 失败恢复

常见症状：
- 升级前检查未通过。
- 控制平面组件未全部恢复。
- 某个节点池升级卡住或版本不一致。

恢复步骤：
1. 立即冻结新的集群变更、工作负载发布、节点扩缩容。
2. 在操作中心确认失败阶段与 `audit_id`，同时保留升级前目标版本与当前版本。
3. 确认恢复基础：
   - etcd/控制平面备份可用。
   - API Server 可访问。
   - `kubectl get nodes`、`kubectl get pods -n kube-system` 可返回结果。
4. 若失败发生在 preflight：
   - 处理版本偏差、弃用 API、磁盘空间或证书告警。
   - 重新生成升级计划，确认警告项已消除后再重试。
5. 若失败发生在控制平面升级：
   - 检查 `kube-apiserver`、`kube-controller-manager`、`kube-scheduler` 静态 Pod 是否全部恢复。
   - 若控制平面组件持续 CrashLoop，优先回退到升级前的静态 Pod 清单或镜像版本。
6. 若失败发生在工作节点升级：
   - 先停止继续滚动。
   - 逐节点检查 kubelet/container runtime 健康和版本一致性。
   - 对单个异常节点完成修复后，再恢复滚动或分批重试。
7. 若升级后出现 API 不可用或控制面持续不健康，按备份恢复流程执行回滚，不要继续推进后续节点。

完成判定：
- 控制平面关键 Pod 全部 Ready。
- `kubectl get nodes` 中节点版本符合目标计划，且没有新的 `NotReady`。
- 升级计划重新评估后无阻塞项。

## Renew 失败恢复

常见症状：
- 续期后控制平面组件未自动恢复。
- 新旧证书不一致，静态 Pod 未加载新证书。
- 节点或控制平面时间漂移导致证书校验失败。

恢复步骤：
1. 在操作中心记录失败 `audit_id`，确认失败组件与续期时间点。
2. 执行 `kubeadm certs check-expiration` 或等效平台命令，确认哪些证书已更新、哪些仍旧过期。
3. 逐项检查控制平面组件：
   - `kube-apiserver`
   - `kube-controller-manager`
   - `kube-scheduler`
   确认静态 Pod 已因证书更新而重启，并加载到新文件。
4. 若某个组件未重启，手工重启对应静态 Pod 或 kubelet，并重新确认健康探针通过。
5. 校验所有控制平面节点时间同步状态；若存在时钟漂移，先修复 NTP/chrony，再重新加载证书。
6. 若新证书异常或部分组件无法读取，回滚到续期前证书备份，再按组件顺序重新签发和重启。
7. 完成控制平面恢复后，再验证 kubelet 与聚合 API 是否能正常握手。

完成判定：
- `kubectl get pods -n kube-system` 中控制平面关键组件全部 Ready。
- `kubeadm certs check-expiration` 无意外过期或残留旧证书。
- 集群详情页中的证书状态恢复正常，且操作中心有成功记录。

## 升级与恢复后的统一核对

1. 在操作中心确认最后一次成功/失败记录链路完整，`audit_id` 可追溯。
2. 在集群详情页确认节点、版本、证书状态与实际集群一致。
3. 恢复被冻结的发布、扩缩容、自动化任务前，先完成一次健康巡检。
