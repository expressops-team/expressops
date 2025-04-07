FROM golang:1.24.2-alpine3.21 
#NOT FROM alpine:3.21 directly because it's not supported by the plugin

# Build arguments
ARG SERVER_PORT=8080
ARG SERVER_ADDRESS=0.0.0.0
ARG TIMEOUT_SECONDS=4
ARG LOG_LEVEL=info
ARG LOG_FORMAT=text
ARG CONFIG_PATH=/app/config.yaml

# Install necessary dependencies
RUN apk add --no-cache git build-base ca-certificates tzdata curl

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
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
          CGO_ENABLED=1 go build -buildmode=plugin -o "$dir/$plugin_name.so" "$gofile" || exit 1; \
        fi \
      done \
    done

# Verify plugins were built
RUN find plugins -name "*.so" | sort

# Compile main application
RUN go build -o expressops ./cmd

# Set environment variables from build args
ENV SERVER_PORT=${SERVER_PORT}
ENV SERVER_ADDRESS=${SERVER_ADDRESS}
ENV TIMEOUT_SECONDS=${TIMEOUT_SECONDS}
ENV LOG_LEVEL=${LOG_LEVEL}
ENV LOG_FORMAT=${LOG_FORMAT}
ENV CONFIG_PATH=${CONFIG_PATH}

# Expose the server port
EXPOSE ${SERVER_PORT}

# Command to run the application
ENTRYPOINT ["sh", "-c", "./expressops -config ${CONFIG_PATH}"]
