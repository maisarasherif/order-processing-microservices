# Multi-stage build for minimal image size

FROM golang:1.25-alpine AS builder


RUN apk add --no-cache git ca-certificates tzdata


WORKDIR /app


COPY go.mod go.sum ./


RUN go mod download


COPY . .


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -a -installsuffix cgo \
    -o payment-service \
    main.go

FROM alpine:3.19


RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser


WORKDIR /app


COPY --from=builder /app/payment-service .


COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo


RUN chown -R appuser:appgroup /app


USER appuser


EXPOSE 8082


HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1


CMD ["./payment-service"]

# Image size: ~15-20MB (vs 800MB+ with full golang image)