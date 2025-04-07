FROM golang:1.24.2-alpine3.21 AS builder
RUN apk add --no-cache git upx build-base
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY . .
# find every plugin and build it
RUN for dir in $(find plugins -type f -name "*.go" -exec dirname {} \; | sort -u); do \ 
      for gofile in $dir/*.go; do \
        if [ -f "$gofile" ]; then \
          plugin_name=$(basename "$gofile" .go); \
          CGO_ENABLED=1 go build -buildmode=plugin -o "$dir/$plugin_name.so" "$gofile"; \
        fi \
      done \
    done


RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o expressops ./cmd

RUN upx expressops || echo "UPX compression failed, continuing anyway"

FROM alpine:3.21
LABEL Name=expressops
RUN apk --no-cache add ca-certificates tzdata && mkdir -p /app/logs
WORKDIR /app
COPY --from=builder /app/expressops .
COPY --from=builder /app/plugins /app/plugins

# create a minimal config file, to not put it manually
RUN echo 'logging:' > /app/config.yaml && \
    echo '  level: info' >> /app/config.yaml && \
    echo '  format: text' >> /app/config.yaml && \
    echo '' >> /app/config.yaml && \
    echo 'server:' >> /app/config.yaml && \
    echo '  port: 8080' >> /app/config.yaml && \
    echo '  address: 0.0.0.0' >> /app/config.yaml && \
    echo '  timeoutSeconds: 4' >> /app/config.yaml && \
    echo '' >> /app/config.yaml && \
    echo '  http:' >> /app/config.yaml && \
    echo '    protocolVersion: 2' >> /app/config.yaml && \
    echo '' >> /app/config.yaml && \
    echo 'plugins: []' >> /app/config.yaml && \
    echo '' >> /app/config.yaml && \
    echo 'flows:' >> /app/config.yaml && \
    echo '  - name: healthz' >> /app/config.yaml && \
    echo '    description: "Health check"' >> /app/config.yaml && \
    echo '    pipeline: []' >> /app/config.yaml

VOLUME ["/app/logs"]
EXPOSE 8080
ENTRYPOINT ["./expressops"]

CMD ["-config", "/app/config.yaml"]

