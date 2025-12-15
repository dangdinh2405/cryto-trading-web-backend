FROM golang:1.25.3-alpine AS builder

WORKDIR /app

# Install ca-certificates for HTTPS calls
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./ 
RUN go mod download

COPY . .

# Build static binary (no CGO dependencies)
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o myapp ./cmd/api

# Production stage
FROM alpine:3.20

WORKDIR /app

# Add ca-certificates for HTTPS and create non-root user
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -g '' appuser

# Copy binary from builder
COPY --from=builder /app/myapp .

# Use non-root user for security
USER appuser

EXPOSE 8080

# Health check for ECS
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./myapp"]
