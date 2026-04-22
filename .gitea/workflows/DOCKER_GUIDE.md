# Docker 配置文档

## 📋 文件说明

### Dockerfile (后端)
- **位置**: 项目根目录
- **功能**: 构建后端API服务
- **特点**:
  - 四阶段构建：前端构建 → 后端构建 → 前端nginx runtime → 后端runtime
  - 前端编译并运行在单独的nginx容器中
  - 后端仅包含API服务
  - 镜像大小优化
  - 非root用户运行
  - 健康检查集成

### web/Dockerfile (前端)
- **位置**: `web/` 目录
- **功能**: 仅构建前端镜像
- **特点**:
  - 基于Nginx Alpine
  - React + Vite 编译构建
  - 包含API代理和WebSocket支持
  - 智能缓存策略（静态资源长期缓存，HTML不缓存）
  - SPA路由支持（try_files）
  - Gzip压缩启用
  - 安全头设置
  - 非root用户运行

### docker-compose.yml
- **位置**: 项目根目录
- **功能**: 完整的应用栈编排
- **包含服务**:
  - Frontend (Nginx - 端口80)
  - Backend (Go API - 端口8080)
  - PostgreSQL (数据库 - 端口5432)
  - Redis (缓存 - 端口6379)
  - Prometheus (监控，可选 - 端口9090)

### .dockerignore
- **目的**: 优化Docker构建上下文
- **效果**: 减小镜像大小，加快构建速度

---

## 🚀 使用方式

### 方案1: 仅构建前端镜像（Nginx）

```bash
# 构建前端镜像
docker build -t opspilot-frontend:latest ./web

# 运行前端容器
docker run -d \
  -p 80:80 \
  --name opspilot-frontend \
  opspilot-frontend:latest

# 访问: http://localhost
```

### 方案2: 仅构建后端镜像

```bash
# 构建后端镜像
docker build -t opspilot-backend:latest .

# 运行后端容器
docker run -d \
  -p 8080:8080 \
  -e DB_HOST=<db-host> \
  -e DB_PASSWORD=<password> \
  --name opspilot-backend \
  opspilot-backend:latest

# 访问: http://localhost:8080
```

### 方案3: Docker Compose (推荐)

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f frontend
docker-compose logs -f backend

# 停止所有服务
docker-compose down

# 删除所有数据和容器
docker-compose down -v
```

---

## 🔧 配置

### 环境变量 (后端)

```bash
# 数据库配置
DB_HOST=postgres
DB_PORT=5432
DB_USER=opspilot
DB_PASSWORD=your-password
DB_NAME=opspilot

# Redis配置
REDIS_HOST=redis
REDIS_PORT=6379

# 应用配置
APP_ENV=production
LOG_LEVEL=info
```

### Nginx 配置 (前端)

前端Dockerfile中内置的Nginx配置包括：

- **静态资源缓存**: 30天缓存 `*.js, *.css, 图片等`
- **HTML不缓存**: 确保总是加载最新版本
- **API代理**: `/api/*` 转发到后端 `http://backend:8080`
- **WebSocket代理**: `/ws` 支持长连接
- **SPA路由**: `try_files $uri $uri/ /index.html` 支持前端路由
- **Gzip压缩**: 启用gzip压缩
- **安全头**: X-Frame-Options, X-Content-Type-Options等

### Docker Compose 自定义

创建 `docker-compose.override.yml` 来覆盖默认配置：

```yaml
version: '3.8'

services:
  backend:
    environment:
      - DB_PASSWORD=your-secure-password
      - LOG_LEVEL=debug
      - APP_ENV=development

  postgres:
    environment:
      - POSTGRES_PASSWORD=your-secure-password

  redis:
    command: redis-server --requirepass your-secure-password
```

---

## 📊 镜像大小对比

| 镜像 | 大小 | 说明 |
|------|------|------|
| opspilot-frontend:latest | ~30MB | Nginx + 前端静态文件 |
| opspilot-backend:latest | ~80MB | Go二进制 + 运行时 |
| 合计 | ~110MB | 完整应用栈 |

相比使用Node.js serve (200MB+)，Nginx减小了70%的镜像大小！

---

## ✅ 最佳实践

### 1. 安全性
- ✅ 使用非root用户运行
- ✅ 使用Alpine基础镜像减少攻击面
- ✅ Nginx配置了安全头（X-Frame-Options, CSP等）
- ✅ 定期更新基础镜像

### 2. 性能
- ✅ Nginx相比Node.js serve性能更好
- ✅ Gzip压缩减少传输大小
- ✅ 智能缓存策略加快加载
- ✅ API代理支持负载均衡

### 3. 可维护性
- ✅ 清晰的多阶段构建
- ✅ 内置的健康检查
- ✅ 容器日志配置
- ✅ 完整的文档

### 4. 前端开发体验
- ✅ Nginx自动处理SPA路由
- ✅ API代理支持跨域请求
- ✅ WebSocket支持实时通信
- ✅ 热重载支持（开发模式）

---

## 🔗 Nginx 路由说明

前端Nginx会自动转发以下请求：

```
Request          Handled By          Purpose
────────────────────────────────────────────────
/                Nginx               主页 (index.html)
/app/*           Nginx               应用内路由
/static/*        Nginx               静态资源 (缓存30天)
/api/*           后端 (8080)         API请求
/ws              后端 (8080)         WebSocket
/health          Nginx               健康检查
```

这样前后端完全分离：
- **前端**: Nginx容器处理（快速、轻量）
- **后端**: Go容器处理（业务逻辑）

---

## 🐛 故障排查

### 问题1: 前端无法访问

```bash
# 查看前端日志
docker logs opspilot-frontend

# 检查nginx进程
docker exec opspilot-frontend ps aux | grep nginx

# 进入容器调试
docker exec -it opspilot-frontend sh
```

### 问题2: 前端无法连接后端

```bash
# 检查网络
docker network inspect opspilot-network

# 测试DNS解析
docker exec opspilot-frontend nslookup backend

# 测试API连接
docker exec opspilot-frontend curl -v http://backend:8080/health
```

### 问题3: 镜像构建失败

```bash
# 清理Docker资源
docker system prune -a

# 查看构建日志
docker build --progress=plain ./web

# 重新构建（不使用缓存）
docker build --no-cache -t opspilot-frontend:latest ./web
```

---

## 📝 CI/CD 集成

工作流中的Docker构建会自动：

1. 构建前端镜像 (Nginx)
2. 构建后端镜像 (Go)
3. 推送到Registry
4. 运行安全扫描

---

## 🚀 生产部署

### Kubernetes 部署示例

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: opspilot-frontend
spec:
  replicas: 2
  selector:
    matchLabels:
      app: opspilot-frontend
  template:
    metadata:
      labels:
        app: opspilot-frontend
    spec:
      containers:
      - name: nginx
        image: registry.example.com/opspilot-frontend:latest
        ports:
        - containerPort: 80
        livenessProbe:
          httpGet:
            path: /health
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: opspilot-frontend
spec:
  selector:
    app: opspilot-frontend
  ports:
  - port: 80
    targetPort: 80
  type: LoadBalancer
```

---

## 📖 更多资源

- [Nginx官方文档](https://nginx.org/en/docs/)
- [Docker最佳实践](https://docs.docker.com/develop/dev-best-practices/)
- [Alpine Linux](https://alpinelinux.org/)
- [React + Vite](https://vitejs.dev/)

---

## 📞 支持

如有问题，请查看容器日志或联系开发团队。
