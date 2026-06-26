FROM golang:1.24-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o /out/pi-sandboxd ./cmd/pi-sandboxd/main.go
RUN go build -ldflags="-s -w" -o /out/pi ./cmd/pi/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    curl \
    ripgrep \
    jq \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r pi && useradd -r -g pi -d /home/pi -s /bin/bash pi
RUN mkdir -p /home/pi/.pi/sandboxes /home/pi/.pi/templates /home/pi/.pi/caches
RUN chown -R pi:pi /home/pi

COPY --from=builder /out/pi-sandboxd /usr/local/bin/pi-sandboxd
COPY --from=builder /out/pi /usr/local/bin/pi

RUN chmod +x /usr/local/bin/pi-sandboxd /usr/local/bin/pi

USER pi
ENV HOME=/home/pi
ENV PI_SOCKET_PATH=/home/pi/.pi/sandboxd.sock

EXPOSE 9001

ENTRYPOINT ["pi-sandboxd"]
CMD ["--socket", "/home/pi/.pi/sandboxd.sock"]
