# -*- coding: utf-8 -*-
# vim: ft=Dockerfile

# container - builder
FROM golang:1.19.1-alpine AS build
LABEL maintainer="mindhunter86 <mindhunter86@vkom.cc>"

ARG GOAPP_MAIN_VERSION="devel"
ARG GOAPP_MAIN_BUILDTIME="never"

ENV MAIN_VERSION=$GOAPP_MAIN_VERSION
ENV MAIN_BUILDTIME=$GOAPP_MAIN_BUILDTIME

# hadolint/hadolint - DL4006
SHELL ["/bin/ash", "-eo", "pipefail", "-c"]

WORKDIR /usr/sources/addie
COPY . .

ENV CGO_ENABLED=0

# skipcq: DOK-DL3018 i'm a badboy, disable this shit
RUN echo "ready" \
  && go build -mod=vendor -trimpath -ldflags="-s -w -X 'main.version=$MAIN_VERSION' -X 'main.buildtime=$MAIN_BUILDTIME'" -o addie \
  && apk add --no-cache upx \
  && upx -9 -k addie


# container - production
FROM alpine:3
LABEL maintainer="mindhunter86 <mindhunter86@vkom.cc>"

# hadolint/hadolint - DL4006
SHELL ["/bin/ash", "-eo", "pipefail", "-c"]

WORKDIR /usr/local/bin/
COPY --from=build --chown=root:nobody --chmod=0550 /usr/sources/addie/addie addie

USER nobody:nobody
ENTRYPOINT ["/usr/local/bin/addie"]
