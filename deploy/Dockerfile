FROM golang:1-alpine3.21 AS builder

RUN apk add --no-cache git ca-certificates build-base su-exec olm-dev

COPY . /build
WORKDIR /build
RUN ls -la
RUN ./build.sh

FROM alpine:3.21

RUN apk add --no-cache ffmpeg su-exec ca-certificates olm bash jq yq curl

COPY --from=builder /build/whatsapp-cloud-bin /usr/bin/whatsapp-cloud
COPY --from=builder /build/docker-run.sh /docker-run.sh
VOLUME /data

CMD ["../docker-run.sh"]
