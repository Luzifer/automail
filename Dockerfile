FROM golang:1.27.3-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS builder

COPY . /go/src/github.com/Luzifer/automail
WORKDIR /go/src/github.com/Luzifer/automail

RUN set -ex \
 && apk add --update git \
 && go install \
      -ldflags "-X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly


FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

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
