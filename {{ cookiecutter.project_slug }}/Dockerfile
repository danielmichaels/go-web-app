FROM debian:bookworm-slim AS litestream

ARG TARGETPLATFORM
ARG litestream_version="v0.3.9"

WORKDIR /litestream

RUN set -x && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      ca-certificates \
      wget

RUN set -x && \
    if [ "$TARGETPLATFORM" = "linux/arm/v7" ]; then \
      ARCH="arm7" ; \
    elif [ "$TARGETPLATFORM" = "linux/arm64" ]; then \
      ARCH="arm64" ; \
    else \
      ARCH="amd64" ; \
    fi && \
    set -u && \
    litestream_binary_tgz_filename="litestream-${litestream_version}-linux-${ARCH}-static.tar.gz" && \
    wget "https://github.com/benbjohnson/litestream/releases/download/${litestream_version}/${litestream_binary_tgz_filename}" && \
    mv "${litestream_binary_tgz_filename}" litestream.tgz
RUN tar -xvzf litestream.tgz

FROM node:lts-bookworm-slim as node
RUN mkdir -p /build
WORKDIR /build
COPY . .
RUN yarn &&\
    mkdir -p /build/assets/static/css/ &&\
    mkdir -p /build/assets/static/js/ &&\
    yarn tailwind &&\
    yarn alpine

FROM golang:1.20-bookworm AS builder

WORKDIR /build

# only copy mod file for better caching
COPY go.mod go.mod
RUN go mod download

COPY --from=node ["/build/assets/static/css/theme.css", "/build/assets/static/css/theme.css"]
COPY --from=node ["/build/assets/static/js/bundle.js", "/build/assets/static/js/bundle.js"]

ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64

COPY . .

RUN apt-get install git &&\
    go build  \
    -ldflags="-s -w" \
    -o app github.com/danielmichaels/familyhub/cmd/app

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder ["/build/litestream.yml", "/app/litestream.yml"]
COPY --from=builder ["/build/entrypoint", "/app/entrypoint"]
#COPY --from=builder ["/build/database/data.db", "/app/database/"]
COPY --from=builder ["/build/app", "/usr/bin/app"]
COPY --from=litestream ["/litestream/litestream", "/usr/bin/litestream"]

# ensures that migrations are run using the embedded files
ENV DOCKER=1
ENTRYPOINT ["/app/entrypoint"]
CMD ["app", "serve"]
