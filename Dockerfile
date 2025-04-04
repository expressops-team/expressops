
# Build image
FROM golang:1.24.2-alpine3.21 AS builder
RUN apk add --no-cache git upx
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY ["go.mod", "go.sum", "./"]
RUN go mod download -x
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o expressops ./cmd/expressops
RUN upx expressops

# Production image
FROM alpine:3.21
LABEL Name=dockerization
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/expressops .
ENTRYPOINT [ "./expressops" ]
EXPOSE 8080
