# Multi-stage build for OpsPilot backend
# Stage 1: Frontend build
FROM node:18-alpine AS frontend-builder

WORKDIR /app/web

# Copy web package files
COPY web/package*.json ./

# Install dependencies
RUN npm ci

# Copy web source code
COPY web/src ./src
COPY web/public ./public
COPY web/*.js ./
COPY web/*.json ./
COPY web/*.config.js ./
COPY web/tsconfig.*.json ./

# Build frontend
RUN npm run build


# Stage 2: Go backend build
FROM golang:1.26.1-alpine3.23 AS backend-builder

WORKDIR /app

# Set build environment
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO_PROXY=https://goproxy.cn,direct

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary with optimizations
RUN go build \
    -ldflags="-s -w -X main.Version=$(git describe --tags --always 2>/dev/null || echo 'unknown')" \
    -o bin/opspilot \
    ./cmd/opspilot


# Stage 3: Frontend nginx runtime
FROM nginx:1.27-alpine AS frontend-runtime

# Install curl for health checks
RUN apk add --no-cache curl

# Remove default nginx config
RUN rm /etc/nginx/conf.d/default.conf

# Create nginx configuration
RUN cat > /etc/nginx/conf.d/default.conf << 'EOF'
server {
    listen 80;
    server_name _;
    client_max_body_size 100M;

    gzip on;
    gzip_types text/plain text/css text/javascript application/javascript application/json;
    gzip_min_length 1000;

    location ~ \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        root /usr/share/nginx/html;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }

    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
EOF

# Copy built frontend from builder
COPY --from=frontend-builder /app/web/dist /usr/share/nginx/html

# Create non-root user
RUN addgroup -g 65534 -S appgroup && \
    adduser -u 65534 -S appuser -G appgroup && \
    chown -R appuser:appgroup /usr/share/nginx/html /var/cache/nginx /var/log/nginx /run/nginx.pid 2>/dev/null || true

USER appuser
EXPOSE 80

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

CMD ["nginx", "-g", "daemon off;"]


# Stage 4: Final backend runtime
FROM alpine:3.23.3

LABEL maintainer="OpsPilot Team"
LABEL description="OpsPilot - Kubernetes Management Platform with Nginx"

WORKDIR /app

# Install ca-certificates for HTTPS and runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Copy built backend binary
COPY --from=backend-builder /app/bin/opspilot .

# Copy configs if needed
COPY --chown=65534:65534 configs ./configs 2>/dev/null || true

# Create non-root user for security
RUN addgroup -g 65534 -S appgroup && \
    adduser -u 65534 -S appuser -G appgroup && \
    chown -R appuser:appgroup /app

USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Expose port
EXPOSE 8080

# Run application
CMD ["./opspilot"]