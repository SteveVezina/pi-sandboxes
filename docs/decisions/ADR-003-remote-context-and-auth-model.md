# ADR-003: Remote Context and Auth Model

## Status
Accepted

## Context

F22 and F23 require local and remote daemon contexts, SSH/Tailscale/WireGuard-friendly access, secure local-to-remote API authentication, and a remote workstation workflow without redesigning the daemon API.

## Decision

Context state is stored in `~/.pi/contexts.yaml`.

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

