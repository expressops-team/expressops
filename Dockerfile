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

# Create a plugins directory with only .so files, preserving the structure
RUN mkdir -p /build/plugins_bin && \
    find plugins -name "*.so" -exec bash -c 'mkdir -p /build/plugins_bin/$(dirname {#} | sed "s|^plugins/||") && cp {#} /build/plugins_bin/$(dirname {#} | sed "s|^plugins/||")/' \; -exec echo "Copied {}" \;

# Build main app with optimizations
RUN go build -ldflags="-s -w" -o expressops ./cmd

# Stage 2: Final image using Distroless
FROM gcr.io/distroless/base-debian11

WORKDIR /app

# Copy only the expressops binary
COPY --from=builder /build/expressops .

# Copy the directory structure with .so files
COPY --from=builder /build/plugins_bin /app/plugins

# Copy required config
COPY docs/samples/config.yaml /app/config.yaml

ENV PLUGINS_PATH=plugins

# Expose port 8080 for server
EXPOSE 8080

# User is already non-root in distroless (not like in alpine)

ENTRYPOINT ["/app/expressops", "-config", "/app/config.yaml"]
# Optimized image with only .so files and preserved directory structure
