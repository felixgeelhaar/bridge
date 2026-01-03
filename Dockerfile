# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bridge ./cmd/bridge

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /bridge /usr/local/bin/bridge

# Copy default policies
COPY policies/ /app/policies/

# Create non-root user
RUN adduser -D -g '' bridge && \
    chown -R bridge:bridge /app

USER bridge

# Default command
ENTRYPOINT ["bridge"]
CMD ["--help"]
