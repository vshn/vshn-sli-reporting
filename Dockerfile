FROM docker.io/library/alpine:3.23 as runtime

RUN \
  apk add --update --no-cache \
    bash \
    coreutils \
    curl \
    ca-certificates \
    tzdata

ENTRYPOINT ["vshn-sli-reporting"]
COPY vshn-sli-reporting /usr/bin/

USER 65536:0
