# PROP-005: Move Pi Box Home Out of `~/.pi`

## Status
✅ Applied to block spec (2026-07-15)

## Block Spec Reference
`SPEC.md` §7 Acceptance Criteria, §15 Local filesystem layout, §17 Cache model, §18 Templates, §19 CLI requirements, §20 Daemon API, §21 Agent SDK requirements, §25 Snapshot and rollback, §27 Logs and telemetry, §34 Configuration file, §39 First coding-agent task list

## Problem

The block spec currently reserves `~/.pi` as the default local state root for the Pi sandbox runtime. That path collides with the Pi coding agent home directory on developer machines.

Observed example:

```text
Pi Home:    /Users/svezina/.pi (exists)
```

This creates two risks:

- The sandbox runtime may read, create, prune, or report state in a directory owned by another Pi tool.
- Users cannot tell whether `~/.pi` contains sandbox state, coding-agent state, or both.

Because the default home path appears in the public CLI surface, daemon socket default, SDK defaults, context store, Docker examples, and feature acceptance criteria, changing it is a block-spec amendment rather than a code-only cleanup.

## Requested By

2026-07-15: User observed `Pi Home: /Users/svezina/.pi (exists)` and requested changing the Pi Box home folder so it does not collide with the Pi coding agent.

## Proposed Amendment

Change the default host-side Pi sandbox runtime home from `~/.pi` to `~/.pi-box`.

Use "Pi Box home" for user-facing status and diagnostics when referring to this runtime-owned directory.

Copy-paste-ready block-spec edits:

- In `SPEC.md` §15, replace "Use predictable state under `~/.pi` by default." with "Use predictable state under `~/.pi-box` by default for host-side Pi sandbox runtime state."
- In `SPEC.md` §15, replace the filesystem tree root `~/.pi/` with `~/.pi-box/`.
- In `SPEC.md` §20, replace the default socket path `~/.pi/sandboxd.sock` with `~/.pi-box/sandboxd.sock`.
- In `SPEC.md` §34, replace the local context target `unix://~/.pi/sandboxd.sock` with `unix://~/.pi-box/sandboxd.sock`.
- In `SPEC.md` §34, replace `storage.root: ~/.pi` with `storage.root: ~/.pi-box`.
- Throughout `SPEC.md`, replace host-side sandbox runtime defaults under `~/.pi/...` with equivalent `~/.pi-box/...` paths.
- Add a note to §15: "Legacy `~/.pi` data is not automatically migrated or pruned by default; Pi Box leaves that directory untouched unless a future migration command is explicitly specified."

### Default paths

Replace host-side defaults as follows:

| Purpose | Current default | New default |
|---------|-----------------|-------------|
| Runtime home | `~/.pi` | `~/.pi-box` |
| Config file | `~/.pi/config.yaml` | `~/.pi-box/config.yaml` |
| Daemon socket | `~/.pi/sandboxd.sock` | `~/.pi-box/sandboxd.sock` |
| Context store | `~/.pi/contexts.yaml` | `~/.pi-box/contexts.yaml` |
| Sandbox metadata | `~/.pi/sandboxes/<id>/...` | `~/.pi-box/sandboxes/<id>/...` |
| Template store | `~/.pi/templates/` | `~/.pi-box/templates/` |
| Dependency caches | `~/.pi/caches/` | `~/.pi-box/caches/` |
| Secrets store | `~/.pi/secrets/` | `~/.pi-box/secrets/` |
| Logs and artifacts under session state | `~/.pi/sandboxes/<id>/logs/`, `~/.pi/sandboxes/<id>/artifacts/` | `~/.pi-box/sandboxes/<id>/logs/`, `~/.pi-box/sandboxes/<id>/artifacts/` |

### Configuration example

Update the default local context and storage root:

```yaml
contexts:
  active: local
  entries:
    local:
      target: unix://~/.pi-box/sandboxd.sock
      transport: unix
      auth:
        type: none

storage:
  root: ~/.pi-box
  maxTotalSize: 100Gi
```

### Migration behavior

The first implementation should not automatically migrate or delete legacy `~/.pi` data. The runtime must:

- Create `~/.pi-box` when needed.
- Leave `~/.pi` untouched.
- Report the new Pi Box home in `pi system status`, `pi system doctor`, API system info, and GUI diagnostics.
- Prefer the new default in SDK clients and context creation.

A later migration command may be specified separately if preserving old sandbox state becomes necessary.

### Container and examples

Docker and Makefile examples should mount the new host-side path. Internal container paths may either move to `/home/pi/.pi-box` for consistency or continue using `/home/pi/.pi` if isolated inside the container; the host-side default must be `~/.pi-box`.

## Rationale

`~/.pi-box` is specific to this sandbox runtime, readable as a user-owned tool home, and avoids claiming the broader `~/.pi` namespace used by the Pi coding agent. Leaving legacy `~/.pi` data untouched prevents accidental deletion or mutation of another tool's state while still giving new installs and diagnostics a collision-free default.

## Impact

Features affected:

- F01 CLI Entry Point: default config path and CLI examples.
- F02 Daemon API: default Unix socket path and daemon-created home directory.
- F03 Fast Backend: cgroup/state paths under runtime home.
- F04 Session Lifecycle: metadata store and TTL config paths.
- F07 Template System: template store path.
- F09 Logs and History: log storage paths.
- F10 System Commands: status, doctor, prune, and disk-usage root.
- F11 Benchmarks: benchmark config path.
- F13 Cache Model: cache root.
- F14 Snapshot and Rollback: snapshot metadata path.
- F15 SDKs: Python and TypeScript default local socket paths.
- F17 Policy Enforcement: default policy config path.
- F22 Remote Daemon Contexts: context store and synthetic local context target.
- F25 GUI Workspace Authorization: GUI allowed-folder preferences path.
- F27 GUI Settings and Diagnostics: GUI defaults and diagnostics path.

ADRs affected:

- ADR-003 Remote Context and Auth Model: update the default context store path from `~/.pi/contexts.yaml` to `~/.pi-box/contexts.yaml`.

Implementation blocked:

- Yes. Code changes that move the default runtime home are blocked until this proposal is accepted because the current block spec explicitly requires `~/.pi`.

## Assumption While Awaiting Acceptance

Until accepted, implementation continues to treat `~/.pi` as the specified default. Investigation may identify affected code and tests, but the runtime-home default must not be changed in code.

## Acceptance Criteria Changes

Update existing acceptance criteria and examples that mention `~/.pi` to use `~/.pi-box` for the sandbox runtime default.

Affected criteria include, but may not be limited to:

- `pi-sandboxd` listens on `~/.pi-box/sandboxd.sock` by default.
- Snapshot metadata is stored under `~/.pi-box/sandboxes/<id>/snapshots/`.
- Contexts persist in `~/.pi-box/contexts.yaml`.
- System commands inspect and maintain `~/.pi-box/`.
- Configuration defaults use `~/.pi-box/config.yaml`.

## Cascade Required on Acceptance

When this PROP is accepted:

1. Update `SPEC.md` to replace the Pi sandbox runtime default home with `~/.pi-box`.
2. Update affected feature specs, especially F01, F02, F03, F04, F07, F09, F10, F11, F13, F14, F15, F17, F22, F25, and F27 where they mention local state paths.
3. Update ADR-003 if it still names `~/.pi/contexts.yaml` as the default context store.
4. Update code defaults in `pkg/system`, `pkg/context`, `pkg/cache`, `pkg/snapshot`, `pkg/template`, `pkg/secrets`, `pkg/policy`, runtime backends, CLI defaults, SDK defaults, Dockerfile, Makefile, and README/getting-started examples.
5. Update tests that assert `~/.pi` paths.
6. Update `docs/proposals/INDEX.md` to mark this proposal applied after the cascade is complete.

## Implementation Blocked?

Resolved. The default runtime home may move to `~/.pi-box`.

## Cascade completed

Completed on 2026-07-15:

- Updated `SPEC.md` to make `~/.pi-box` the default host-side Pi sandbox runtime home and to leave legacy `~/.pi` data untouched by default.
- Updated ADR-003 with the new default context store path.
- Updated affected feature specs and marked impacted implemented features as needing re-verification.
- Updated code defaults in runtime home, context, cache, snapshot, template, secret, policy, compat runtime, SDK, Docker, Makefile, README, and tests.
- Updated `docs/features/INDEX.md` and `docs/proposals/INDEX.md`.
