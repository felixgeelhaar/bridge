# Build stage
FROM golang:1.26-alpine@sha256:30b1e546a9c7fa3b747c2bdba3a0c7bdf3a02ebb4c5b8405d2f948d7e8aac4ec AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata && apk upgrade --no-cache zlib

# Copy go mod files
COPY --chown=root:root go.mod go.sum ./
RUN go mod download

# Copy source code
COPY --chown=root:root . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bridge ./cmd/bridge

# Runtime stage
FROM alpine:3.23@sha256:469c9ea410b9215eaee7976ba6c0639039b444be8e5d638f7abbada189971eb2

LABEL org.opencontainers.image.authors="Felix Geelhaar" \
      org.opencontainers.image.source="https://github.com/felixgeelhaar/bridge"

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata && apk upgrade --no-cache zlib

# Create non-root user
RUN adduser -D -g '' bridge

# Copy binary from builder
COPY --from=builder --chown=bridge:bridge /bridge /usr/local/bin/bridge

# Copy default policies
COPY --chown=bridge:bridge policies/ /app/policies/

USER bridge

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["bridge", "--version"]

# Default command
ENTRYPOINT ["bridge"]
CMD ["--help"]
