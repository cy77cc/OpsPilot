# API 文档规范

## 1. 概述

本文档基于 OpenAPI 3.0 规范定义了平台的 API 接口标准，作为后端开发的权威依据。文档基于当前 Mock 数据结构，详细描述了所有 API 端点的规范，包括 HTTP 方法、URL 路径、请求参数、响应结构、状态码等信息。

## 2. 基本信息

### 2.1 版本信息

- **API 版本**: v1
- **文档版本**: 1.0.0
- **最后更新**: 2026-02-23

### 2.2 认证方式

所有 API 端点均采用 JWT (JSON Web Token) 认证方式，需要在请求头中包含 `Authorization` 字段：

```http
Authorization: Bearer <token>
```

### 2.3 基础路径

所有 API 端点的基础路径为：

```
/api/v1
```

### 2.4 响应格式

所有 API 响应均采用 JSON 格式，标准响应结构如下：

#### 2.4.1 成功响应

```json
{
  "success": true,
  "data": {...},  // 响应数据
  "message": "操作成功"  // 可选
}
```

#### 2.4.2 错误响应

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "错误描述"
  },
  "message": "操作失败"  // 可选
}
```

## 3. API 端点定义

### 3.1 主机管理 API

#### 3.1.1 获取主机列表

```http
GET /api/v1/hosts
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| status | string | query | 主机状态筛选 | 否 | online |
| region | string | query | 区域筛选 | 否 | us-west-1 |
| tags | string[] | query | 标签筛选 | 否 | ["api", "prod"] |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 4,
    "list": [
      {
        "id": "host-1",
        "name": "Web Server 1",
        "ip": "192.168.1.10",
        "status": "online",
        "cpu": 72,
        "memory": 58,
        "disk": 45,
        "network": 82,
        "tags": ["web", "production", "us-west"],
        "region": "us-west-1",
        "createdAt": "2023-01-15T08:30:00Z",
        "lastActive": "2023-11-20T14:20:00Z"
      }
    ]
  }
}
```

#### 3.1.2 获取主机详情

```http
GET /api/v1/hosts/{id}
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| id | string | path | 主机ID | 是 | host-1 |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "id": "host-1",
    "name": "Web Server 1",
    "ip": "192.168.1.10",
    "status": "online",
    "cpu": 72,
    "memory": 58,
    "disk": 45,
    "network": 82,
    "tags": ["web", "production", "us-west"],
    "region": "us-west-1",
    "createdAt": "2023-01-15T08:30:00Z",
    "lastActive": "2023-11-20T14:20:00Z"
  }
}
```

#### 3.1.3 创建主机

```http
POST /api/v1/hosts
```

**请求体**:

```json
{
  "name": "New Server",
  "ip": "192.168.1.30",
  "status": "online",
  "tags": ["new", "test"],
  "region": "us-west-1"
}
```

**响应结构**:

```json
{
  "success": true,
  "data": {
    "id": "host-new",
    "name": "New Server",
    "ip": "192.168.1.30",
    "status": "online",
    "cpu": 0,
    "memory": 0,
    "disk": 0,
    "network": 0,
    "tags": ["new", "test"],
    "region": "us-west-1",
    "createdAt": "2026-02-23T10:00:00Z",
    "lastActive": "2026-02-23T10:00:00Z"
  },
  "message": "主机创建成功"
}
```

#### 3.1.4 更新主机

```http
PUT /api/v1/hosts/{id}
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| id | string | path | 主机ID | 是 | host-1 |

**请求体**:

```json
{
  "name": "Updated Server",
  "status": "maintenance",
  "tags": ["web", "production", "us-west", "maintenance"]
}
```

**响应结构**:

```json
{
  "success": true,
  "data": {
    "id": "host-1",
    "name": "Updated Server",
    "ip": "192.168.1.10",
    "status": "maintenance",
    "cpu": 72,
    "memory": 58,
    "disk": 45,
    "network": 82,
    "tags": ["web", "production", "us-west", "maintenance"],
    "region": "us-west-1",
    "createdAt": "2023-01-15T08:30:00Z",
    "lastActive": "2023-11-20T14:20:00Z"
  },
  "message": "主机更新成功"
}
```

#### 3.1.5 删除主机

```http
DELETE /api/v1/hosts/{id}
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| id | string | path | 主机ID | 是 | host-1 |

**响应结构**:

```json
{
  "success": true,
  "message": "主机删除成功"
}
```

### 3.2 任务管理 API

#### 3.2.1 获取任务列表

```http
GET /api/v1/tasks
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| status | string | query | 任务状态筛选 | 否 | running |
| type | string | query | 任务类型筛选 | 否 | scheduled |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 4,
    "list": [
      {
        "id": "task-1",
        "name": "Data Backup",
        "type": "scheduled",
        "status": "success",
        "schedule": "0 2 * * *",
        "lastRun": "2023-11-19T02:00:00Z",
        "nextRun": "2023-11-20T02:00:00Z",
        "duration": 125,
        "createdAt": "2023-01-15T08:30:00Z"
      }
    ]
  }
}
```

#### 3.2.2 获取任务详情

```http
GET /api/v1/tasks/{id}
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| id | string | path | 任务ID | 是 | task-1 |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "id": "task-1",
    "name": "Data Backup",
    "type": "scheduled",
    "status": "success",
    "schedule": "0 2 * * *",
    "lastRun": "2023-11-19T02:00:00Z",
    "nextRun": "2023-11-20T02:00:00Z",
    "duration": 125,
    "createdAt": "2023-01-15T08:30:00Z"
  }
}
```

#### 3.2.3 创建任务

```http
POST /api/v1/tasks
```

**请求体**:

```json
{
  "name": "New Backup Task",
  "type": "scheduled",
  "schedule": "0 3 * * *",
  "nextRun": "2026-02-24T03:00:00Z"
}
```

**响应结构**:

```json
{
  "success": true,
  "data": {
    "id": "task-new",
    "name": "New Backup Task",
    "type": "scheduled",
    "status": "pending",
    "schedule": "0 3 * * *",
    "nextRun": "2026-02-24T03:00:00Z",
    "createdAt": "2026-02-23T10:00:00Z"
  },
  "message": "任务创建成功"
}
```

#### 3.2.4 更新任务

```http
PUT /api/v1/tasks/{id}
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| id | string | path | 任务ID | 是 | task-1 |

**请求体**:

```json
{
  "name": "Updated Backup Task",
  "schedule": "0 4 * * *",
  "status": "pending"
}
```

**响应结构**:

```json
{
  "success": true,
  "data": {
    "id": "task-1",
    "name": "Updated Backup Task",
    "type": "scheduled",
    "status": "pending",
    "schedule": "0 4 * * *",
    "lastRun": "2023-11-19T02:00:00Z",
    "nextRun": "2023-11-20T04:00:00Z",
    "duration": 125,
    "createdAt": "2023-01-15T08:30:00Z"
  },
  "message": "任务更新成功"
}
```

#### 3.2.5 删除任务

```http
DELETE /api/v1/tasks/{id}
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| id | string | path | 任务ID | 是 | task-1 |

**响应结构**:

```json
{
  "success": true,
  "message": "任务删除成功"
}
```

#### 3.2.6 获取任务日志

```http
GET /api/v1/tasks/{id}/logs
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| id | string | path | 任务ID | 是 | task-1 |
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 20 |
| level | string | query | 日志级别筛选 | 否 | error |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 2,
    "list": [
      {
        "id": "log-1",
        "taskId": "task-1",
        "timestamp": "2023-11-19T02:00:00Z",
        "level": "info",
        "message": "Backup started"
      },
      {
        "id": "log-2",
        "taskId": "task-1",
        "timestamp": "2023-11-19T02:02:05Z",
        "level": "info",
        "message": "Backup completed successfully"
      }
    ]
  }
}
```

### 3.3 Kubernetes 集群管理 API

#### 3.3.1 获取集群列表

```http
GET /api/v1/k8s/clusters
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| status | string | query | 集群状态筛选 | 否 | healthy |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 2,
    "list": [
      {
        "id": "cluster-1",
        "name": "Production Cluster",
        "version": "v1.25.3",
        "status": "healthy",
        "nodes": 10,
        "pods": 98,
        "cpu": 75,
        "memory": 65,
        "createdAt": "2023-01-15T08:30:00Z"
      }
    ]
  }
}
```

#### 3.3.2 获取集群详情

```http
GET /api/v1/k8s/clusters/{id}
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| id | string | path | 集群ID | 是 | cluster-1 |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "id": "cluster-1",
    "name": "Production Cluster",
    "version": "v1.25.3",
    "status": "healthy",
    "nodes": 10,
    "pods": 98,
    "cpu": 75,
    "memory": 65,
    "createdAt": "2023-01-15T08:30:00Z"
  }
}
```

#### 3.3.3 获取集群节点列表

```http
GET /api/v1/k8s/clusters/{clusterId}/nodes
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| clusterId | string | path | 集群ID | 是 | cluster-1 |
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| status | string | query | 节点状态筛选 | 否 | ready |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 2,
    "list": [
      {
        "id": "node-1",
        "name": "prod-worker-1",
        "role": "worker",
        "status": "ready",
        "cpu": 72,
        "memory": 64,
        "pods": 15,
        "labels": {
          "zone": "us-west-1a",
          "node-type": "general-purpose"
        }
      }
    ]
  }
}
```

#### 3.3.4 获取集群 Pod 列表

```http
GET /api/v1/k8s/clusters/{clusterId}/pods
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| clusterId | string | path | 集群ID | 是 | cluster-1 |
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| namespace | string | query | 命名空间筛选 | 否 | frontend |
| status | string | query | Pod 状态筛选 | 否 | running |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 7,
    "list": [
      {
        "id": "pod-1",
        "name": "web-app-1",
        "namespace": "frontend",
        "status": "running",
        "phase": "Running",
        "node": "prod-worker-1",
        "cpu": 25,
        "memory": 15,
        "restarts": 0,
        "age": "2d 4h",
        "labels": {
          "app": "web-app",
          "tier": "frontend"
        },
        "containers": [
          {
            "name": "main",
            "image": "nginx:1.21",
            "status": "running",
            "cpu": 20,
            "memory": 12,
            "restarts": 0
          }
        ],
        "qosClass": "Burstable",
        "createdBy": "devops-team",
        "startTime": "2024-02-20T08:30:00Z"
      }
    ]
  }
}
```

#### 3.3.5 获取集群服务列表

```http
GET /api/v1/k8s/clusters/{clusterId}/services
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| clusterId | string | path | 集群ID | 是 | cluster-1 |
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| namespace | string | query | 命名空间筛选 | 否 | frontend |
| type | string | query | 服务类型筛选 | 否 | LoadBalancer |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 2,
    "list": [
      {
        "id": "svc-1",
        "name": "web-svc",
        "namespace": "frontend",
        "type": "LoadBalancer",
        "clusterIP": "10.96.5.10",
        "externalIP": "203.0.113.25",
        "ports": [
          {
            "port": 80,
            "targetPort": 8080,
            "protocol": "TCP"
          }
        ],
        "selector": {
          "app": "web-app"
        },
        "age": "2d 10h"
      }
    ]
  }
}
```

#### 3.3.6 获取集群 Ingress 列表

```http
GET /api/v1/k8s/clusters/{clusterId}/ingresses
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| clusterId | string | path | 集群ID | 是 | cluster-1 |
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| namespace | string | query | 命名空间筛选 | 否 | frontend |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 2,
    "list": [
      {
        "id": "ing-1",
        "name": "web-ingress",
        "namespace": "frontend",
        "host": "example.com",
        "path": "/",
        "service": "web-svc",
        "port": 80,
        "tls": true
      }
    ]
  }
}
```

### 3.4 监控告警 API

#### 3.4.1 获取告警列表

```http
GET /api/v1/alerts
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| severity | string | query | 告警级别筛选 | 否 | critical |
| status | string | query | 告警状态筛选 | 否 | firing |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 3,
    "list": [
      {
        "id": "alert-1",
        "title": "High CPU Usage on Web Server",
        "severity": "warning",
        "source": "Prometheus",
        "status": "firing",
        "createdAt": "2023-11-20T09:45:00Z"
      }
    ]
  }
}
```

#### 3.4.2 获取告警规则列表

```http
GET /api/v1/alert-rules
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| severity | string | query | 规则级别筛选 | 否 | critical |
| enabled | boolean | query | 是否启用筛选 | 否 | true |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 3,
    "list": [
      {
        "id": "rule-1",
        "name": "CPU Threshold",
        "condition": "cpu_usage > 80%",
        "severity": "warning",
        "enabled": true,
        "channels": ["email", "slack"],
        "createdAt": "2023-01-15T08:30:00Z"
      }
    ]
  }
}
```

#### 3.4.3 获取监控指标

```http
GET /api/v1/monitor/metrics
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| metric | string | query | 指标名称 | 是 | cpu_usage |
| startTime | string | query | 开始时间 (ISO 格式) | 是 | 2023-11-20T00:00:00Z |
| endTime | string | query | 结束时间 (ISO 格式) | 是 | 2023-11-20T10:00:00Z |
| interval | string | query | 时间间隔 | 否 | 1h |

**响应结构**:

```json
{
  "success": true,
  "data": [
    {
      "timestamp": "2023-11-20T00:00:00Z",
      "value": 72.1
    },
    {
      "timestamp": "2023-11-20T01:00:00Z",
      "value": 74.3
    }
  ]
}
```

### 3.5 集成工具 API

#### 3.5.1 获取集成工具列表

```http
GET /api/v1/integrations
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| status | string | query | 状态筛选 | 否 | connected |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 3,
    "list": [
      {
        "id": "tool-1",
        "name": "Grafana",
        "icon": "📊",
        "url": "https://grafana.example.com",
        "status": "connected",
        "description": "Visualization and analytics platform"
      }
    ]
  }
}
```

### 3.6 配置管理 API

#### 3.6.1 获取配置列表

```http
GET /api/v1/configs
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| status | string | query | 配置状态筛选 | 否 | active |
| env | string | query | 环境筛选 | 否 | production |
| type | string | query | 配置类型筛选 | 否 | json |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 8,
    "list": [
      {
        "id": "1",
        "name": "数据库连接池配置",
        "key": "db.pool.size",
        "value": "{\"min\": 5, \"max\": 50}",
        "version": 3,
        "status": "active",
        "type": "json",
        "env": "production",
        "updatedAt": "2024-02-15 14:30:00",
        "updatedBy": "admin"
      }
    ]
  }
}
```

#### 3.6.2 获取配置版本历史

```http
GET /api/v1/configs/{id}/versions
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| id | string | path | 配置ID | 是 | 1 |

**响应结构**:

```json
{
  "success": true,
  "data": [
    {
      "id": "1-v1",
      "configId": "1",
      "version": 1,
      "value": "{\"min\": 5, \"max\": 10}",
      "createdAt": "2024-02-15 14:30:00",
      "createdBy": "admin",
      "comment": "Version 1"
    },
    {
      "id": "1-v2",
      "configId": "1",
      "version": 2,
      "value": "{\"min\": 5, \"max\": 30}",
      "createdAt": "2024-02-15 14:30:00",
      "createdBy": "admin",
      "comment": "Version 2"
    }
  ]
}
```

### 3.7 服务管理 API

#### 3.7.1 获取服务列表

```http
GET /api/v1/services
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| status | string | query | 服务状态筛选 | 否 | running |
| environment | string | query | 环境筛选 | 否 | production |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 3,
    "list": [
      {
        "id": "svc-app-1",
        "name": "Frontend App",
        "status": "running",
        "owner": "frontteam",
        "environment": "production",
        "tags": ["web", "ui", "http"],
        "cpu": 60,
        "memory": 2048,
        "replicas": 3,
        "lastDeployTime": "2023-11-18T14:30:00Z",
        "createdAt": "2023-01-15T08:30:00Z",
        "k8sResources": {
          "pods": [],
          "services": [],
          "ingresses": []
        },
        "config": "{\n  \"port\": 80,\n  \"healthCheck\": \"/health\"\n}",
        "metrics": {
          "cpu": [60, 65, 70, 72, 65],
          "memory": [2048, 2100, 2200, 2090, 2110]
        }
      }
    ]
  }
}
```

#### 3.7.2 获取服务配额

```http
GET /api/v1/services/quota
```

**响应结构**:

```json
{
  "success": true,
  "data": {
    "cpuLimit": 100,
    "memoryLimit": 16384,
    "cpuUsed": 190,
    "memoryUsed": 14336
  }
}
```

### 3.8 配置中心 API

#### 3.8.1 获取配置应用列表

```http
GET /api/v1/config-center/apps
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 3,
    "list": [
      {
        "id": "app-1",
        "name": "user-service",
        "serviceId": "svc-app-1",
        "description": "用户服务配置",
        "namespaces": ["development", "staging", "production"],
        "createdAt": "2026-01-10T09:15:00Z",
        "updatedAt": "2026-01-20T09:15:00Z"
      }
    ]
  }
}
```

#### 3.8.2 获取配置项列表

```http
GET /api/v1/config-center/apps/{appId}/items
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| appId | string | path | 应用ID | 是 | app-1 |
| namespace | string | query | 命名空间筛选 | 否 | production |
| env | string | query | 环境筛选 | 否 | prod |
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 3,
    "list": [
      {
        "id": "config-1",
        "appId": "app-1",
        "namespace": "production",
        "env": "prod",
        "key": "database.url",
        "value": "jdbc:mysql://prod-db.example.com:3306/users",
        "format": "text",
        "isSecret": false,
        "createdAt": "2026-01-10T10:00:00Z",
        "updatedAt": "2026-01-10T10:00:00Z",
        "updatedBy": "admin"
      }
    ]
  }
}
```

#### 3.8.3 获取配置项版本历史

```http
GET /api/v1/config-center/items/{itemId}/versions
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| itemId | string | path | 配置项ID | 是 | config-1 |

**响应结构**:

```json
{
  "success": true,
  "data": [
    {
      "version": 1,
      "value": "jdbc:mysql://test-db.example.com:3306/users",
      "createdBy": "admin",
      "createdAt": "2026-01-10T10:00:00Z",
      "comment": "测试环境URL"
    },
    {
      "version": 2,
      "value": "jdbc:mysql://prod-db.example.com:3306/users",
      "createdBy": "admin",
      "createdAt": "2026-01-15T14:00:00Z",
      "comment": "生产环境URL"
    }
  ]
}
```

#### 3.8.4 获取审计日志

```http
GET /api/v1/config-center/audit-logs
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| appId | string | query | 应用ID筛选 | 否 | app-1 |
| action | string | query | 操作类型筛选 | 否 | update |
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 13,
    "list": [
      {
        "id": "log-1",
        "appId": "app-1",
        "appName": "user-service",
        "namespace": "database",
        "key": "db.pool.size",
        "action": "update",
        "operator": "admin",
        "timestamp": "2024-02-15 14:30:00",
        "details": "更新配置值: max: 30 -> max: 50",
        "oldValue": "{\"min\": 5, \"max\": 30}",
        "newValue": "{\"min\": 5, \"max\": 50}",
        "status": "success"
      }
    ]
  }
}
```

#### 3.8.5 获取发布记录

```http
GET /api/v1/config-center/releases
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| appId | string | query | 应用ID筛选 | 否 | app-1 |
| status | string | query | 发布状态筛选 | 否 | success |
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 2,
    "list": [
      {
        "id": "release-1",
        "appId": "app-1",
        "namespace": "production",
        "key": "database.url",
        "env": "prod",
        "fromVersion": 1,
        "toVersion": 2,
        "releasedBy": "admin",
        "releasedAt": "2026-01-15T14:30:00Z",
        "status": "success",
        "comment": "切换到生产数据库"
      }
    ]
  }
}
```

#### 3.8.6 获取配置模板

```http
GET /api/v1/config-center/templates
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| category | string | query | 模板分类筛选 | 否 | infrastructure |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 2,
    "list": [
      {
        "id": "tmpl-1",
        "name": "Database Configuration Template",
        "description": "标准数据库配置模板",
        "format": "json",
        "content": "{\n  \"database\": {\n    \"url\": \"{{ DB_HOST }}:{{ DB_PORT }}/{{ DB_NAME }}\",\n    \"username\": \"{{ DB_USER }}\",\n    \"password\": \"{{ DB_PASSWORD }}\",\n    \"options\": {\n      \"poolSize\": {{ DB_POOL_SIZE }}\n    }\n  }\n}",
        "category": "infrastructure"
      }
    ]
  }
}
```

### 3.9 任务调度 API

#### 3.9.1 获取作业列表

```http
GET /api/v1/jobs
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| type | string | query | 作业类型筛选 | 否 | shell |
| status | string | query | 作业状态筛选 | 否 | running |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 6,
    "list": [
      {
        "id": "job-001",
        "name": "数据库备份任务",
        "type": "shell",
        "command": "mysqldump -u root -p$DB_PASSWORD myapp_db > backup_$(date +%Y%m%d).sql && gzip backup_$(date +%Y%m%d).sql",
        "schedule": "0 2 * * *",
        "timeout": 3600,
        "strategy": "random",
        "retryCount": 2,
        "retryInterval": 60,
        "concurrencyPolicy": "Forbid",
        "description": "每日定时备份生产数据库",
        "enabled": true,
        "createdAt": "2026-01-15T08:30:00Z",
        "updatedAt": "2026-01-20T14:20:00Z"
      }
    ]
  }
}
```

#### 3.9.2 获取作业执行记录

```http
GET /api/v1/jobs/{jobId}/executions
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| jobId | string | path | 作业ID | 是 | job-001 |
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| status | string | query | 执行状态筛选 | 否 | success |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 2,
    "list": [
      {
        "id": "exec-001",
        "jobId": "job-001",
        "startTime": "2026-02-22T02:00:05Z",
        "endTime": "2026-02-22T02:15:22Z",
        "status": "success",
        "exitCode": 0,
        "stdout": "mysqldump: [Warning] Using a password on the command line interface can be insecure.\nDump completed on 2026-02-22 02:15:22\nCompressed file created successfully",
        "stderr": "",
        "retryCount": 0
      }
    ]
  }
}
```

#### 3.9.3 获取作业调度计划

```http
GET /api/v1/jobs/{jobId}/schedules
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| jobId | string | path | 作业ID | 是 | job-001 |
| page | integer | query | 页码 | 否 | 1 |
| pageSize | integer | query | 每页数量 | 否 | 10 |
| status | string | query | 调度状态筛选 | 否 | executed |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "total": 3,
    "list": [
      {
        "id": "sched-001",
        "jobId": "job-001",
        "scheduledTime": "2026-02-23T02:00:00Z",
        "actualStartTime": "2026-02-23T02:00:05Z",
        "actualEndTime": "2026-02-23T02:14:55Z",
        "status": "executed"
      }
    ]
  }
}
```

## 4. 状态码说明

| 状态码 | 描述 | 说明 |
|--------|------|------|
| 200 | OK | 请求成功 |
| 201 | Created | 资源创建成功 |
| 204 | No Content | 请求成功但无内容返回 |
| 400 | Bad Request | 请求参数错误 |
| 401 | Unauthorized | 未授权，缺少或无效的认证信息 |
| 403 | Forbidden | 禁止访问，权限不足 |
| 404 | Not Found | 资源不存在 |
| 405 | Method Not Allowed | 请求方法不支持 |
| 500 | Internal Server Error | 服务器内部错误 |
| 502 | Bad Gateway | 网关错误 |
| 503 | Service Unavailable | 服务不可用 |
| 504 | Gateway Timeout | 网关超时 |

## 5. 版本控制策略

### 5.1 API 版本控制

- **版本号位置**: API 路径中包含版本号，如 `/api/v1/hosts`
- **版本升级策略**: 当 API 发生不兼容变更时，增加主版本号
- **向后兼容**: 旧版本 API 在一段时间内保持可用（至少 6 个月）

### 5.2 变更管理流程

1. **变更申请**: 提交 API 变更申请，包括变更原因、影响范围、向后兼容性分析
2. **变更审核**: 由架构师和相关团队审核变更申请
3. **变更实施**: 按照审核通过的方案实施变更
4. **测试验证**: 进行充分的测试，确保变更不会影响现有功能
5. **发布通知**: 提前通知 API 消费者变更内容和生效时间
6. **灰度发布**: 采用灰度发布策略，逐步扩大变更影响范围
7. **回滚机制**: 建立变更回滚机制，出现问题时快速回滚

### 5.3 向后兼容性保障

- **新增字段**: 只在响应中新增字段，不修改现有字段结构
- **废弃字段**: 对于需要废弃的字段，先标记为废弃并在文档中说明，一段时间后再移除
- **参数兼容**: 对于新增的请求参数，设置合理的默认值，确保旧客户端仍然可以正常工作
- **错误处理**: 保持错误响应格式一致，确保客户端可以正确处理错误

## 6. 文档维护

### 6.1 文档更新流程

1. **变更触发**: 当 API 发生变更时，触发文档更新
2. **文档修改**: 根据变更内容更新 API 文档
3. **文档审核**: 由相关团队审核文档更新内容
4. **文档发布**: 将更新后的文档发布到文档平台

### 6.2 文档工具

- **文档格式**: OpenAPI 3.0
- **文档生成工具**: Swagger Editor
- **文档展示工具**: Swagger UI
- **文档版本控制**: 与代码仓库保持一致，使用 Git 进行版本控制

## 7. 安全注意事项

1. **认证与授权**: 所有 API 端点必须进行认证和授权检查
2. **输入验证**: 对所有输入参数进行严格验证，防止注入攻击
3. **输出脱敏**: 对敏感信息（如密码、密钥）进行脱敏处理
4. **HTTPS**: 所有 API 通信必须使用 HTTPS
5. **Rate Limiting**: 实施 API 速率限制，防止滥用
6. **CORS**: 合理配置 CORS 策略，只允许受信任的域名访问
7. **审计日志**: 记录所有 API 操作的审计日志，便于安全分析

## 8. 附录

### 8.1 示例请求

#### 8.1.1 获取主机列表

```http
GET /api/v1/hosts?page=1&pageSize=10&status=online
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### 8.1.2 创建主机

```http
POST /api/v1/hosts
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "name": "New Server",
  "ip": "192.168.1.30",
  "status": "online",
  "tags": ["new", "test"],
  "region": "us-west-1"
}
```

### 8.2 示例响应

#### 8.2.1 成功响应

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": true,
  "data": {
    "total": 4,
    "list": [
      {
        "id": "host-1",
        "name": "Web Server 1",
        "ip": "192.168.1.10",
        "status": "online",
        "cpu": 72,
        "memory": 58,
        "disk": 45,
        "network": 82,
        "tags": ["web", "production", "us-west"],
        "region": "us-west-1",
        "createdAt": "2023-01-15T08:30:00Z",
        "lastActive": "2023-11-20T14:20:00Z"
      }
    ]
  }
}
```

#### 8.2.2 错误响应

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "success": false,
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "IP 地址格式错误"
  },
  "message": "操作失败"
}
```
### 3.20 主控台概览 API

#### 3.20.1 获取主控台概览

```http
GET /api/v1/dashboard/overview
```

**请求参数**:

| 参数名 | 类型 | 位置 | 描述 | 必填 | 示例 |
|--------|------|------|------|------|------|
| time_range | string | query | 时间范围，可选 `1h` / `6h` / `24h`，默认 `1h` | 否 | 6h |

**响应结构**:

```json
{
  "success": true,
  "data": {
    "hosts": {
      "total": 15,
      "healthy": 12,
      "degraded": 2,
      "offline": 1
    },
    "clusters": {
      "total": 5,
      "healthy": 4,
      "unhealthy": 1
    },
    "services": {
      "total": 32,
      "healthy": 28,
      "degraded": 2,
      "unhealthy": 2
    },
    "alerts": {
      "firing": 3,
      "recent": [
        {
          "id": "101",
          "title": "CPU 超过 85%",
          "severity": "warning",
          "source": "node-01",
          "createdAt": "2026-03-04T02:10:00Z"
        }
      ]
    },
    "events": [
      {
        "id": "node-201",
        "type": "host_online",
        "message": "node-05 上线",
        "createdAt": "2026-03-04T02:08:00Z"
      }
    ],
    "metrics": {
      "cpu_usage": [
        {
          "timestamp": "2026-03-04T01:10:00Z",
          "value": 42.3
        }
      ],
      "memory_usage": [
        {
          "timestamp": "2026-03-04T01:10:00Z",
          "value": 63.8
        }
      ]
    }
  }
}
```
