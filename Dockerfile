FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o /akira .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata su-exec \
    && adduser -D -u 1000 akira

COPY --from=builder /akira /usr/local/bin/akira
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh \
    && mkdir -p /data

WORKDIR /data
VOLUME ["/data"]

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["akira", "daemon", "-f", "--pid-file", "/data/akira.pid"]
