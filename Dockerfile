# ============= Stage 1: Build ================
FROM golang:1.24.2-alpine3.21 AS builder

# Install necessary build tools
RUN apk add --no-cache git build-base ca-certificates

WORKDIR /app

# Copy and download dependencies first (leverage Docker cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire codebase
COPY . .

# Clean any pre-existing .so files
RUN find plugins -name "*.so" -delete

# Compile plugins
RUN for dir in $(find plugins -type f -name "*.go" -exec dirname {} \; | sort -u); do \
      for gofile in $dir/*.go; do \
        if [ -f "$gofile" ]; then \
          plugin_name=$(basename "$gofile" .go); \
          echo "Building plugin $plugin_name.so from $gofile"; \
          CGO_ENABLED=1 go build -ldflags="-s -w" -buildmode=plugin -o "$dir/$plugin_name.so" "$gofile" || exit 1; \
        fi \
      done \
    done

# Create target directory for .so files only
RUN mkdir -p /app/plugins_bin && \
    find plugins -name "*.so" -exec bash -c 'mkdir -p /app/plugins_bin/$(dirname {} | sed "s|^plugins/||") && cp {} /app/plugins_bin/$(dirname {} | sed "s|^plugins/||")/' \; && \
    echo "Compiled plugin files:" && \
    find /app/plugins_bin -type f | sort

# Compile main application
RUN go build -ldflags="-s -w" -o expressops ./cmd

# ============= Runtime stage - using Alpine ================
FROM alpine:3.19

# Install minimal dependencies
RUN apk add --no-cache ca-certificates

# Copy the compiled application and plugins from the builder stage
WORKDIR /app
COPY --from=builder /app/expressops /app/
COPY --from=builder /app/plugins_bin /app/plugins/
COPY docs/samples/config.yaml /app/config.yaml

# Set environment variables
ENV PLUGINS_PATH=plugins

# Expose port 8080 - this is just documentation, actual port is set via Kubernetes
EXPOSE 8080

# Use non-root user for better security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Run the application
ENTRYPOINT ["/app/expressops", "-config", "/app/config.yaml"]
