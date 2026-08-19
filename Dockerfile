FROM golang:1.26.6-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS builder

COPY . /go/src/github.com/Luzifer/automail
WORKDIR /go/src/github.com/Luzifer/automail

RUN set -ex \
 && apk add --update git \
 && go install \
      -ldflags "-X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly


FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

ENV CONFIG=/data/config.yml \
    STORAGE_FILE=/data/automail_store.yml

LABEL maintainer="Knut Ahlers <knut@ahlers.me>"

RUN set -ex \
 && apk --no-cache add \
      ca-certificates

COPY --from=builder /go/bin/automail /usr/local/bin/automail

VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/automail"]
CMD ["--"]

# vim: set ft=Dockerfile:
