# OpsPilot - Kubernetes Deployment Management Platform

OpsPilot 是一个基于 Kubernetes 的部署管理平台，提供完整的 PaaS 能力。

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Frontend (React)                            │
├─────────────────────────────────────────────────────────────────────────┤
│  App.tsx (路由配置)                                                      │
│  ├── AppLayout (侧边栏 + Header)                                        │
│  ├── Dashboard (仪表板)                                                  │
│  ├── Services (服务管理)                                                 │
│  ├── Deployment (部署管理)
│  │   ├── Infrastructure (基础设施: 集群/凭证/主机)
│  │   ├── Targets (部署目标)
│  │   ├── Releases (发布管理)
│  │   └── Observability (可观测性)
│  ├── Hosts (主机管理 → 已迁移到 Deployment/Infrastructure)
│  ├── Tasks (任务中心)                                                    │
│  └── Settings (系统设置)
├─────────────────────────────────────────────────────────────────────────┤
│                              Backend (Go/Gin)                            │
├─────────────────────────────────────────────────────────────────────────┤
│  internal/                                                               │
│  ├── server/         # HTTP 服务入口                                     │
│  ├── service/        # 业务服务模块                                       │
│  │   ├── host/       # 主机管理                                          │
│  │   ├── deployment/ # 部署管理                                          │
│  │   ├── jobs/       # 任务调度                                          │
│  │   ├── cluster/    # 集群管理                                          │
│  │   └── ...                                                            │
│  ├── model/          # 数据模型                                          │
│  ├── websocket/      # WebSocket 推送                                    │
│  └── middleware/     # 中间件 (JWT, 日志, 权限)                           │
├─────────────────────────────────────────────────────────────────────────┤
│                              Database (MySQL/PostgreSQL)                 │
└─────────────────────────────────────────────────────────────────────────┘
```

## Key Directories

```
web/
├── src/
│   ├── App.tsx                    # 主应用入口和路由配置
│   ├── components/
│   │   └── Layout/
│   │       └── AppLayout.tsx      # 侧边栏菜单配置
│   ├── pages/
│   │   ├── Deployment/            # 部署管理页面
│   │   │   ├── Infrastructure/    # 基础设施 (集群/凭证/主机)
│   │   │   ├── Targets/           # 部署目标
│   │   │   └── Observability/     # 可观测性
│   │   ├── Hosts/                 # 主机管理
│   │   ├── Tasks/                 # 任务中心
│   │   └── ...
│   ├── api/
│   │   ├── api.ts                 # Axios 实例和拦截器
│   │   └── modules/               # API 模块
│   ├── hooks/
│   │   └── useNotificationWebSocket.ts  # WebSocket 连接
│   └── contexts/
│       └── NotificationContext.tsx      # 通知状态管理

internal/
├── server/
│   └── server.go                  # HTTP 服务启动
├── service/
│   ├── service.go                 # 服务注册入口
│   ├── host/                      # 主机管理服务
│   ├── deployment/                # 部署管理服务
│   └── jobs/                      # 任务调度服务 (新增)
├── model/
│   ├── host.go
│   ├── deployment.go
│   └── job.go                     # 任务模型 (新增)
└── websocket/
    ├── handler.go                 # WebSocket 处理
    └── hub.go                     # 连接管理
```

## Specs

| 规范 | 描述 |
|------|------|
| [navigation](./specs/navigation/spec.md) | 前端导航和路由系统 |
| [websocket](./specs/websocket/spec.md) | WebSocket 实时通信 |
| [api-routing](./specs/api-routing/spec.md) | 后端 API 路由规范 |
| [notification-center](./specs/notification-center/spec.md) | 通知中心功能 |
| [notification-realtime](./specs/notification-realtime/spec.md) | 实时通知推送 |

## Recent Changes

### 2026-03-02

1. **侧边栏导航重构**
   - 删除重复菜单项
   - 修正菜单路径
   - 添加缺失路由

2. **主机路由迁移**
   - `/hosts/*` 迁移到 `/deployment/infrastructure/hosts/*`
   - 保留旧路由重定向

3. **部署管理路由完善**
   - 所有子路由使用正确的页面组件
   - 基础设施、部署目标、可观测性路由完整

4. **WebSocket 连接优化**
   - 修复频繁重连问题
   - 添加 Vite proxy 配置
   - 防止 React 重渲染导致的重复连接

5. **Jobs API 新增**
   - 创建 `internal/service/jobs/` 模块
   - 实现任务 CRUD 和执行管理

## Development Setup

### 前端

```bash
cd web
npm install
npm run dev
```

### 后端

```bash
go run cmd/server/main.go
```

### 环境变量

```bash
# .env
VITE_API_BASE=/api/v1
VITE_WS_URL=         # 可选，自定义 WebSocket URL
```
