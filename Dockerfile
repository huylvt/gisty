# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies for cgo (required by go-enry/go-oniguruma)
RUN apk add --no-cache \
    gcc \
    musl-dev \
    oniguruma-dev \
    git

WORKDIR /app

# Copy go mod files first for better cache
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/gisty ./cmd/server

# Final stage
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    oniguruma \
    tzdata

# Create non-root user
RUN addgroup -g 1000 gisty && \
    adduser -u 1000 -G gisty -s /bin/sh -D gisty

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gisty /app/gisty

# Copy docs directory for Swagger (if exists)
COPY --from=builder /app/docs /app/docs

# Change ownership
RUN chown -R gisty:gisty /app

# Switch to non-root user
USER gisty

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/gisty"]
