FROM danielmichaels/ci-toolkit AS toolkit
FROM golang:1.24-bookworm AS builder

WORKDIR /build
# only copy mod file for better caching
COPY ../../go.mod go.sum ./
RUN go mod download && go mod verify

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY ../.. .

RUN apt-get install git -y &&\
    go build  \
    -ldflags="-s -w" \
    -o gecko ./cmd/gecko

FROM debian:bookworm-slim
WORKDIR /app

COPY --from=toolkit ["/usr/local/bin/goose", "/usr/local/bin/goose"]
COPY --from=builder ["/build/entrypoint", "/app/entrypoint"]
COPY --from=builder ["/build/assets/migrations", "/app/migrations"]
COPY --from=builder ["/build/gecko", "/usr/bin/gecko"]

RUN apt-get update && apt-get install ca-certificates curl -y &&\
    chmod +x /app/entrypoint

# ensures that migrations are run using the embedded files
ENV DOCKER=1
ENTRYPOINT ["/app/entrypoint"]