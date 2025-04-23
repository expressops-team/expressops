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

RUN find plugins -name "*.so" | sort

# Compile main application
RUN go build -ldflags="-s -w" -o expressops ./cmd

# ============= Runtime stage - using distroless ================
FROM gcr.io/distroless/base-debian12

# Copy the compiled application and plugins from the builder stage
WORKDIR /app
COPY --from=builder /app/expressops /app/
COPY --from=builder /app/plugins /app/plugins/



# Expose port 8080 - this is just documentation, actual port is set via Kubernetes
EXPOSE 8080

# Run the application
ENTRYPOINT ["/app/expressops", "-config"]

# CMD will be overwritten by k3s
CMD ["/app/config.yaml"]
