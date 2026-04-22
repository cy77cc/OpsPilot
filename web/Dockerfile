# Multi-stage build for OpsPilot frontend
# Stage 1: Build
FROM node:18-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY src ./src
COPY public ./public
COPY *.js .
COPY *.json .
COPY *.config.js .
COPY tsconfig.*.json ./

# Build application
RUN npm run build


# Stage 2: Runtime
FROM node:18-alpine

LABEL maintainer="OpsPilot Team"
LABEL description="OpsPilot Frontend - React + Vite"

WORKDIR /app

# Install serve to run the application
RUN npm install -g serve

# Create non-root user for security
RUN addgroup -g 65534 -S appgroup && \
    adduser -u 65534 -S appuser -G appgroup

# Copy built frontend from builder
COPY --from=builder /app/dist ./dist

# Copy public assets if needed
COPY --from=builder /app/public ./public

# Change ownership
RUN chown -R appuser:appgroup /app

USER appuser

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000 || exit 1

# Run the application
CMD ["serve", "-s", "dist", "-l", "3000"]
