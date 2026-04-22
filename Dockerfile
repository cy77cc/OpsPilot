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


# Stage 3: Runtime image
FROM alpine:3.23.3

LABEL maintainer="OpsPilot Team"
LABEL description="OpsPilot - Kubernetes Management Platform"

WORKDIR /app

# Install ca-certificates for HTTPS and runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Copy built frontend from builder
COPY --from=frontend-builder /app/web/dist ./web/dist

# Copy built binary from builder
COPY --from=backend-builder /app/bin/opspilot .

# Copy configs and scripts if needed
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