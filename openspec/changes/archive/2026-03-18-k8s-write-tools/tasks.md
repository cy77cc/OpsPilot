## 1. Platform Discovery Tool

- [x] 1.1 Create `internal/ai/tools/platform/tools.go` with `PlatformDiscoverResources` tool
- [x] 1.2 Implement resource type handlers: clusters, hosts, services, namespaces, metrics
- [x] 1.3 Add RBAC checks for resource visibility
- [x] 1.4 Register tool in `internal/ai/tools/tools.go` entry point

## 2. K8s Write Tools - P0 (Core Operations)

- [x] 2.1 Create `internal/ai/tools/kubernetes/write.go` with write operation tools
- [x] 2.2 Implement `k8s_scale_deployment` tool with replicas parameter
- [x] 2.3 Implement `k8s_restart_deployment` tool using annotation update
- [x] 2.4 Implement `k8s_delete_pod` tool with grace period support
- [x] 2.5 Add input validation for all P0 tools

## 3. K8s Write Tools - P1 (Advanced Operations)

- [x] 3.1 Implement `k8s_rollback_deployment` tool
- [x] 3.2 Implement `k8s_delete_deployment` tool with critical risk level

## 4. Approval Middleware Integration

- [x] 4.1 Add K8s write tool names to `DefaultNeedsApproval` map
- [x] 4.2 Update `DefaultPreviewGenerator` for K8s tools
- [x] 4.3 Add tool-specific preview generators to `DefaultToolConfigs`
- [x] 4.4 Verify approval flow works with new K8s tools

## 5. Tool Registration and Testing

- [x] 5.1 Update `NewKubernetesWriteTools` to return actual tools
- [x] 5.2 Update `NewChangeTools` to include write tools
- [x] 5.3 Write unit tests for all new tools
- [x] 5.4 Verify approval flow end-to-end

## 6. Documentation

- [x] 6.1 Update CLAUDE.md with new tool documentation
- [x] 6.2 Archive change after implementation
