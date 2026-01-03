# Build stage
FROM golang:1.21-alpine3.20 AS builder

# Version build args
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

# Build the application
COPY . ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s -X 'kiwi/internal/config.Version=${VERSION}' -X 'kiwi/internal/config.GitCommit=${GIT_COMMIT}' -X 'kiwi/internal/config.BuildTime=${BUILD_TIME}'" \
    -o kiwi cmd/main.go

# Final stage
FROM alpine:3.20

# Copy binary from builder and create appuser
RUN apk --no-cache add ca-certificates
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser
WORKDIR /app
COPY --from=builder /build/kiwi .
RUN mkdir -p /app/data && chown -R appuser:appuser /app
USER appuser

EXPOSE 3300 50051

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3300/health || exit 1

CMD ["./kiwi"]
