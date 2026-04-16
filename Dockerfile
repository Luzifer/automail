FROM golang:1.26-alpine@sha256:27f829349da645e287cb195a9921c106fc224eeebbdc33aeb0f4fca2382befa6 AS builder

COPY . /go/src/github.com/Luzifer/automail
WORKDIR /go/src/github.com/Luzifer/automail

RUN set -ex \
 && apk add --update git \
 && go install \
      -ldflags "-X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly


FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

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
