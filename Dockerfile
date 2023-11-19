# -*- coding: utf-8 -*-
# vim: ft=Dockerfile

ARG GOAPP_MAIN_VERSION=""
ARG GOAPP_MAIN_BUILDTIME=""


# container - builder
FROM golang:1.19.1-alpine AS build
LABEL maintainer="mindhunter86 <mindhunter86@vkom.cc>"

# hadolint/hadolint - DL4006
SHELL ["/bin/ash", "-eo", "pipefail", "-c"]

WORKDIR /usr/sources/addie
COPY . .

ENV CGO_ENABLED=0
RUN echo "ready" \
  && go build -mod=vendor -trimpath -ldflags="-s -w -X main.version=$GOAPP_MAIN_VERSION -X main.buildtime=$GOAPP_MAIN_BUILDTIME" -o addie \
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
