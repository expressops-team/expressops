FROM golang:1.24.2-alpine3.21 AS builder
RUN apk add --no-cache git upx build-base
WORKDIR /app
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY . .
# Compilar plugins
RUN for dir in $(find plugins -type f -name "*.go" -exec dirname {} \; | sort -u); do \
      for gofile in $dir/*.go; do \
        if [ -f "$gofile" ]; then \
          plugin_name=$(basename "$gofile" .go); \
          CGO_ENABLED=1 go build -buildmode=plugin -o "$dir/$plugin_name.so" "$gofile"; \
        fi \
      done \
    done
<<<<<<< HEAD


RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o expressops ./cmd

RUN upx expressops || echo "UPX compression failed, continuing anyway"
=======
# Compilar aplicación principal
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o expressops ./cmd
>>>>>>> db06aae (NO LOG PLUGIN // NOT WORKING)

FROM alpine:3.21

# Variables configurables mediante build args
ARG SERVER_PORT=8080
ARG SERVER_ADDRESS=0.0.0.0
ARG TIMEOUT_SECONDS=4
ARG LOG_LEVEL=info
ARG LOG_FORMAT=text

RUN apk --no-cache add ca-certificates tzdata curl
WORKDIR /app
COPY --from=builder /app/expressops .
COPY --from=builder /app/plugins /app/plugins
COPY docs/samples/config.yaml /app/config.yaml

# Configurar variables en tiempo de construcción
ENV SERVER_PORT=${SERVER_PORT}
ENV SERVER_ADDRESS=${SERVER_ADDRESS}
ENV TIMEOUT_SECONDS=${TIMEOUT_SECONDS}
ENV LOG_LEVEL=${LOG_LEVEL}
ENV LOG_FORMAT=${LOG_FORMAT}

EXPOSE ${SERVER_PORT}
ENTRYPOINT ["./expressops"]
<<<<<<< HEAD

CMD ["-config", "/app/config.yaml"]

=======
CMD ["-config", "/docs/samples/config.yaml"]
>>>>>>> db06aae (NO LOG PLUGIN // NOT WORKING)
