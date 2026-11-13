FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS builder

COPY . /go/src/github.com/Luzifer/automail
WORKDIR /go/src/github.com/Luzifer/automail

RUN set -ex \
 && apk add --update git \
 && go install \
      -ldflags "-X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly


FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

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
