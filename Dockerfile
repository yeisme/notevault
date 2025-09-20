# Build stage
FROM golang:1.25.1-alpine3.22 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -trimpath -ldflags "-s -w" -o notevault ./cmd/notevault

# Final stage
FROM alpine:latest

# Install ca-certificates, tzdata, and curl for health checks
RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/notevault .

# Create logs directory and set permissions
RUN mkdir -p logs && \
    chown -R appuser:appgroup /root/

# Switch to non-root user
USER appuser

# Expose ports
# 8080: Application port
# 9091: Pprof port And Prometheus metrics port
EXPOSE 8080
EXPOSE 9091

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Use ENTRYPOINT so runtime can pass flags, e.g. `-c /config/config.yaml`
ENTRYPOINT ["./notevault"]
