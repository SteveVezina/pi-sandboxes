# ADR-003: Remote Context and Auth Model

## Status
Accepted

## Context

F22 and F23 require local and remote daemon contexts, SSH/Tailscale/WireGuard-friendly access, secure local-to-remote API authentication, and a remote workstation workflow without redesigning the daemon API.

## Decision

Context state is stored in `~/.pi-box/contexts.yaml`.

*(Updated 2026-07-15 per PROP-005: the default host-side Pi Box home moved from `~/.pi` to `~/.pi-box` to avoid colliding with the Pi coding agent home.)*

Required context fields are:

- `target`: daemon endpoint URI
- `transport`: `unix`, `http`, or `ssh`
- `auth.type`: `none`, `bearer-token`, or `ssh-agent`

The active context may be overridden per command with `--context <name>`.

Remote daemon access keeps the existing daemon HTTP API unchanged. Supported transports are:

- `unix`: local Unix socket
- `http`: direct HTTP endpoint, intended for private networks such as Tailscale or WireGuard
- `ssh`: SSH-forwarded daemon access

Authentication rules:

- `unix` contexts may use `auth.type: none`.
- `http` contexts require bearer-token auth.
- `ssh` contexts use SSH agent authentication for the transport.
- Bearer tokens are stored outside sandbox workspaces and are never injected into sandbox environment variables.
- Remote auth failures return actionable errors and do not fall back to unauthenticated access.

*(Clarified 2026-09-01: bearer-token auth is enforced by daemon-side middleware,
not only attached by the client. The daemon reads the expected token from the
`PI_DAEMON_TOKEN` environment variable (never a flag, never on disk). All routes
except `GET /health` and `OPTIONS` require `Authorization: Bearer <token>` when a
token is set; a non-loopback HTTP bind without a token is a startup error. This
adds a middleware, not a route change, so "the daemon HTTP API is unchanged"
still holds. See F23 T23.5.)*

## Consequences

- CLI and SDK clients can route to remote daemons without changing daemon routes.
- Credential material stays outside sandbox workspaces.
- Remote auth behavior is explicit and testable.

## References

- `SPEC.md` §31 Milestone 6
- `SPEC.md` §34 Configuration file
- `docs/features/F22-remote-daemon-contexts.md`
- `docs/features/F23-remote-daemon-transport-auth.md`
- PROP-003
