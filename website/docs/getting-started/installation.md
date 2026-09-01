---
sidebar_position: 1
---

# Installation

## Prerequisites

- **Linux** for the `fast` and `secure` backends. macOS / Windows work via
  the `compat` (container) backend with Docker or Podman.
- **Go 1.21+** to build from source.
- **Git** and **curl**.
- A container engine (**Docker** or **Podman**) for `compat` / `secure`
  modes.

## From source

```bash
git clone https://gitlab.com/pi-sandbox/pi-sandbox-runtime.git
cd pi-sandbox-runtime
make install
```

Or build the two binaries directly:

```bash
go build -o bin/pi-box       ./cmd/pi-box/
go build -o bin/pi-sandboxd  ./cmd/pi-sandboxd/
```

## Docker

```bash
docker build -t pi-sandbox .
docker run -d \
  --name pi-sandbox \
  -v ~/.pi-box:/home/pi/.pi-box \
  -p 9001:9001 \
  pi-sandbox
```

## Pre-built binaries

See the
[releases page](https://gitlab.com/pi-sandbox/pi-sandbox-runtime/-/releases).

## Verify

```bash
pi-box --version
pi-sandboxd --socket ~/.pi-box/sandboxd.sock &
pi-box system doctor
```

`system doctor` checks the `~/.pi-box` layout, config, disk space,
permissions, and which runtime backends are available on this host.
