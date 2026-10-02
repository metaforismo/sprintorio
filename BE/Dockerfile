FROM golang:1.25.13-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM alpine:3.21
ARG OPENSSL_MIN_VERSION=3.3.7-r2
RUN apk add --no-cache --upgrade "libcrypto3>=${OPENSSL_MIN_VERSION}" "libssl3>=${OPENSSL_MIN_VERSION}" ca-certificates && \
    addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
RUN mkdir -p /app/uploads && chown -R appuser:appgroup /app
USER appuser
EXPOSE 8080
CMD ["sh", "-c", "./server migrate up && exec ./server"]
