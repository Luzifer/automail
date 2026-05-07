FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

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
