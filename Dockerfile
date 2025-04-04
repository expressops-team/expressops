# Build image
FROM golang:1.24.2-alpine3.21 AS builder
RUN apk add --no-cache git upx build-base
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY ["go.mod", "go.sum", "./"]
RUN go mod download -x
COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o expressops ./cmd/expressops.go
RUN upx expressops

# Production image
FROM alpine:3.21
LABEL Name=expressops
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/expressops .
COPY docs/samples/config.yaml ./config.yaml
COPY --from=builder /app/plugins/ plugins/
ENTRYPOINT [ "./expressops" ]
CMD [ "-config", "/app/config.yaml" ]
EXPOSE 8080

