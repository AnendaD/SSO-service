# Build stage
# Using latest Go version to ensure compatibility with go.mod requirements
FROM golang:alpine AS builder

WORKDIR /app

# Install dependencies for building
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application (static binary)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -ldflags '-extldflags "-static"' \
    -o /app/grpc-auth ./cmd/sso

# Build the migrator (static binary)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -ldflags '-extldflags "-static"' \
    -o /app/migrator ./cmd/migrator

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

# Copy binaries from builder
COPY --from=builder /app/grpc-auth .
COPY --from=builder /app/migrator .

# Copy migrations
COPY migrations ./migrations

# Copy config file
COPY config/prod.yaml ./config/prod.yaml

# Copy entrypoint script
COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

# Expose port (will be overridden by Koyeb's PORT env var)
EXPOSE 8080

# Run entrypoint script
ENTRYPOINT ["./docker-entrypoint.sh"]
