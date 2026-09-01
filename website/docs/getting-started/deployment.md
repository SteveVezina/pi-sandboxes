---
sidebar_position: 5
---

# Deploying a remote daemon

`pi-sandboxd` is normally local. To run it on another host and reach it from
your workstation, you have two safe shapes:

1. **Private network / tunnel (preferred).** Run the daemon loopback-only and
   reach it over SSH port-forward, Tailscale, or WireGuard. No token strictly
   required if the transport itself authenticates (SSH), but a token is still
   recommended for `http` contexts on a shared tailnet.
2. **Public bind with a bearer token.** Bind `0.0.0.0` and set
   `PI_DAEMON_TOKEN`. The daemon **fails to start** on a non-loopback bind
   without a token.

## Container / PaaS

The repo ships a `Dockerfile` and a `render.yaml` blueprint.

Environment the daemon reads:

| Var | Purpose |
|-----|---------|
| `PORT` | HTTP port (PaaS platforms inject this) |
| `PI_HTTP_ADDR` | HTTP bind host — set to `0.0.0.0` in a container |
| `PI_DAEMON_TOKEN` | **required** bearer token for a non-loopback bind |
| `PI_SOCKET_PATH` | Unix socket path (defaults to `~/.pi-box/sandboxd.sock`) |

### Render

`render.yaml` defines two services — the daemon (Docker) and this docs site
(static). On first deploy Render generates `PI_DAEMON_TOKEN`; copy it from the
service's *Environment* tab.

```bash
docker build -t pi-sandboxd .
docker run -p 9001:9001 -e PI_DAEMON_TOKEN=$(openssl rand -hex 32) pi-sandboxd
```

The health check is `GET /health` (never requires a token).

## Pointing the client at it

Create an `http` context whose `auth.token_env` names an env var holding the
same token:

```bash
export PI_REMOTE_TOKEN=<the token from Render>
pi-box context create prod https://pi-sandboxd.onrender.com \
  --transport http \
  --auth bearer-token \
  --token-env PI_REMOTE_TOKEN
pi-box context use prod
```

See [contexts](/cli/contexts) for the full command surface.

## Security notes

- A public daemon exposes sandbox **create / exec / shell / files**. The token
  is the only thing between the internet and code execution — treat it like a
  root password, rotate it, scope network access with the platform firewall.
- The token is compared in constant time and is never logged.
- `GET /health` and `OPTIONS` pre-flight are the only unauthenticated routes.
