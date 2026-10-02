FROM golang:1.25.13-alpine AS build
WORKDIR /src
COPY collector/main.go .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /collector main.go

FROM alpine:3.22
RUN apk add --no-cache ca-certificates coreutils git \
    && addgroup -g 1000 sprintorio \
    && adduser -D -u 1000 -G sprintorio sprintorio
COPY --from=build /collector /usr/local/bin/sprintorio-collector
COPY common/sprintorio-service-entrypoint /usr/local/bin/sprintorio-service-entrypoint
RUN chmod 0755 /usr/local/bin/sprintorio-service-entrypoint /usr/local/bin/sprintorio-collector

USER 1000:1000
ENV HOME=/home/sprintorio
EXPOSE 8091
CMD ["/usr/local/bin/sprintorio-collector"]
