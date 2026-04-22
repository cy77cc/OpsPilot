# Docker 配置文档

## 📋 文件说明

### Dockerfile (后端)
- **位置**: 项目根目录
- **功能**: 构建包含前端和后端的完整应用镜像
- **特点**:
  - 三阶段构建 (Multi-stage)
  - 前端编译 + 后端编译 + 运行时镜像
  - 镜像大小优化
  - 非root用户运行
  - 健康检查集成

### web/Dockerfile (前端)
- **位置**: `web/` 目录
- **功能**: 仅构建前端镜像
- **特点**:
  - 基于Node.js Alpine
  - 使用serve提供生产级服务
  - 安全的非root用户
  - 健康检查

### docker-compose.yml
- **位置**: 项目根目录
- **功能**: 完整的应用栈编排
- **包含服务**:
  - Backend (OpensPilot API)
  - Frontend (React应用)
  - PostgreSQL (数据库)
  - Redis (缓存)
  - Prometheus (监控，可选)

### .dockerignore
- **目的**: 优化Docker构建上下文
- **效果**: 减小镜像大小，加快构建速度

---

## 🚀 使用方式

### 方案1: 构建完整镜像（后端+前端）

```bash
# 构建镜像
docker build -t opspilot:latest .

# 运行容器（需要数据库）
docker run -d \
  -p 8080:8080 \
  -e DB_HOST=<db-host> \
  -e DB_PASSWORD=<password> \
  opspilot:latest
```

### 方案2: 仅构建前端镜像

```bash
# 构建前端镜像
docker build -t opspilot-frontend:latest ./web

# 运行前端容器
docker run -d \
  -p 3000:3000 \
  -e VITE_API_BASE_URL=http://backend:8080 \
  opspilot-frontend:latest
```

### 方案3: Docker Compose (推荐开发和演示)

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f backend
docker-compose logs -f frontend

# 停止所有服务
docker-compose down

# 删除所有数据
docker-compose down -v
```

---

## 🔧 配置

### 环境变量 (后端)

在运行容器时设置：

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=opspilot
DB_PASSWORD=your-password
DB_NAME=opspilot

# Redis配置
REDIS_HOST=localhost
REDIS_PORT=6379

# 应用配置
APP_ENV=production
LOG_LEVEL=info
```

### 环境变量 (前端)

在docker-compose或构建时设置：

```bash
# API配置
VITE_API_BASE_URL=http://backend:8080
VITE_WS_URL=ws://backend:8080
```

### Docker Compose 自定义

创建 `docker-compose.override.yml` 来覆盖默认配置：

```yaml
version: '3.8'

services:
  backend:
    environment:
      - DB_PASSWORD=your-secure-password
      - LOG_LEVEL=debug

  postgres:
    environment:
      - POSTGRES_PASSWORD=your-secure-password
```

---

## 📊 镜像大小对比

| 镜像 | 大小 | 说明 |
|------|------|------|
| opspilot:latest | ~150MB | 包含前后端完整应用 |
| opspilot-frontend:latest | ~200MB | 仅前端应用 |

---

## ✅ 最佳实践

### 1. 安全性
- ✅ 使用非root用户运行
- ✅ 使用Alpine基础镜像减少攻击面
- ✅ 定期更新基础镜像
- ✅ 扫描镜像漏洞: `trivy image opspilot:latest`

### 2. 性能
- ✅ 多阶段构建减小镜像大小
- ✅ 缓存优化 (Dockerfile中的COPY顺序)
- ✅ 去掉不必要的文件 (.dockerignore)

### 3. 可维护性
- ✅ 明确的构建阶段和标签
- ✅ 完整的注释和元数据
- ✅ 健康检查确保服务正常
- ✅ 一致的错误处理

### 4. 监控
- ✅ 集成Prometheus监控
- ✅ 健康检查端点
- ✅ 容器日志配置

---

## 🐛 故障排查

### 问题1: 容器启动失败

```bash
# 查看容器日志
docker logs opspilot-backend

# 检查容器状态
docker ps -a

# 进入容器调试
docker exec -it opspilot-backend sh
```

### 问题2: 前端无法连接后端

```bash
# 检查网络连接
docker network ls
docker network inspect opspilot-network

# 检查DNS解析
docker exec opspilot-frontend nslookup backend
```

### 问题3: 数据库连接失败

```bash
# 检查PostgreSQL状态
docker-compose logs postgres

# 验证连接
docker exec opspilot-postgres psql -U opspilot -d opspilot -c "SELECT 1"
```

### 问题4: 镜像构建失败

```bash
# 清理Docker资源
docker system prune -a

# 重新构建（不使用缓存）
docker build --no-cache -t opspilot:latest .
```

---

## 📝 CI/CD 集成

### GitHub Actions / Gitea Actions

在工作流中构建Docker镜像：

```yaml
- name: Build and push Docker image
  uses: docker/build-push-action@v5
  with:
    context: .
    push: true
    tags: registry.example.com/opspilot:latest
    cache-from: type=gha
    cache-to: type=gha,mode=max
```

### 推送到Registry

```bash
# 登录Registry
docker login registry.example.com

# 标记镜像
docker tag opspilot:latest registry.example.com/opspilot:1.0.0

# 推送镜像
docker push registry.example.com/opspilot:1.0.0
```

---

## 🔗 相关命令

```bash
# 查看镜像信息
docker image inspect opspilot:latest

# 查看镜像层
docker image history opspilot:latest

# 扫描漏洞
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image opspilot:latest

# 执行compose命令
docker-compose ps              # 查看服务状态
docker-compose up              # 启动服务
docker-compose down            # 停止服务
docker-compose logs -f         # 查看日志
docker-compose exec backend sh # 进入容器
```

---

## 📖 更多资源

- [Docker官方文档](https://docs.docker.com/)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)
- [Compose文件参考](https://docs.docker.com/compose/compose-file/)
- [Alpine Linux](https://alpinelinux.org/)

---

## 📞 支持

如有问题，请查看容器日志或联系开发团队。
