# Stage 1: Build
FROM golang:1.24.2-alpine3.21 AS builder

# Install build dependencies
RUN apk add --no-cache git build-base ca-certificates

WORKDIR /build

# Copy and download dependencies first (leverage Docker cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Clean any existing .so files
RUN find plugins -name "*.so" -delete

# Build plugins
RUN for dir in $(find plugins -type f -name "*.go" -exec dirname {} \; | sort -u); do \
      for gofile in $dir/*.go; do \
        if [ -f "$gofile" ]; then \
          plugin_name=$(basename "$gofile" .go); \
          echo "Building plugin $plugin_name.so from $gofile"; \
          CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o "$dir/$plugin_name.so" "$gofile" || exit 1; \
        fi \
      done \
    done

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