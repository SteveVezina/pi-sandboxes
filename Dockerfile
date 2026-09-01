FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o /out/pi-sandboxd ./cmd/pi-sandboxd/main.go
RUN go build -ldflags="-s -w" -o /out/pi-box ./cmd/pi-box/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    curl \
    ripgrep \
    jq \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r pi && useradd -r -g pi -d /home/pi -s /bin/bash pi
RUN mkdir -p /home/pi/.pi-box/sandboxes /home/pi/.pi-box/templates /home/pi/.pi-box/caches
RUN chown -R pi:pi /home/pi

COPY --from=builder /out/pi-sandboxd /usr/local/bin/pi-sandboxd
COPY --from=builder /out/pi-box /usr/local/bin/pi-box

RUN chmod +x /usr/local/bin/pi-sandboxd /usr/local/bin/pi-box

USER pi
ENV HOME=/home/pi
ENV PI_SOCKET_PATH=/home/pi/.pi-box/sandboxd.sock
# Container deploys expose the HTTP API. Bind all interfaces and default the
# port; PaaS platforms (Render, Fly, …) inject $PORT. A non-loopback bind
# REQUIRES a bearer token — set PI_DAEMON_TOKEN or the daemon refuses to start
# (F23 T23.5).
ENV PI_HTTP_ADDR=0.0.0.0
ENV PORT=9001

EXPOSE 9001

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
    CMD curl -fsS "http://127.0.0.1:${PORT}/health" || exit 1

ENTRYPOINT ["pi-sandboxd"]
CMD []
