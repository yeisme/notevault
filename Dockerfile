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

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/notevault .

# Copy configs
COPY --from=builder /app/configs ./configs

# Create logs directory
RUN mkdir -p logs

# Expose port
EXPOSE 8080
EXPOSE 9090

# Command to run
CMD ["./notevault"]
