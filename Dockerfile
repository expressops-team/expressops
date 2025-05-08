# Stage 1: Build
FROM golang:1.24.2-alpine3.21 AS builder

# Install build dependencies
RUN apk add --no-cache git build-base ca-certificates gcc musl-dev

WORKDIR /build

# Copy and download dependencies first (leverage Docker cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Clean any existing .so files
RUN find plugins -name "*.so" -delete

# Build plugins
RUN mkdir -p plugins && \
    CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o plugins/slack/slack.so plugins/slack/slack.go && \
    CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o plugins/healthcheck/health_check.so plugins/healthcheck/health_check.go && \
    CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o plugins/formatters/health_alert_formatter.so plugins/formatters/health_alert_formatter.go && \
    CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o plugins/testprint/testprint.so plugins/testprint/testprint.go && \
    CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o plugins/flowlister/flow_lister.so plugins/flowlister/flow_lister.go && \
    CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o plugins/sleep/sleep_plugin.so plugins/sleep/sleep_plugin.go && \
    CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o plugins/usercreation/user_creation.so plugins/usercreation/user_creation.go && \
    CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o plugins/permissions/permissions.so plugins/permissions/permissions.go && \
    CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o plugins/opensearch/opensearch_logger.so plugins/opensearch/opensearch_logger.go

# Build main app with optimizations
RUN go build -ldflags="-s -w" -o expressops ./cmd

# Stage 2: Final tiny image
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binaries and plugins from build stage
COPY --from=builder /build/expressops .
COPY --from=builder /build/plugins ./plugins
RUN find plugins -name "*.go" -delete

# Copy required config
COPY docs/samples/config.yaml /app/config.yaml

# Environment variables with direct values <=== given in the config.yaml file


# Expose port 8080 for server
EXPOSE 8080

# Run as non-root for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

ENTRYPOINT ["./expressops", "-config", "/app/config.yaml"]

# Image size reduced from 1.59GB to 162MB :0