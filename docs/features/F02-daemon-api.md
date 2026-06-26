# F02: Daemon API

> Source: `SPEC.md` §6 Features F2
> Status: 🟢 Implemented
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F2 | Daemon API | `pi-sandboxd` local daemon exposing Unix socket HTTP API for sandbox operations | M1 |

## Expanded Specification

`pi-sandboxd` is a local HTTP daemon that exposes a RESTful API over a Unix socket (`~/.pi/sandboxd.sock`) by default. An optional localhost HTTP listener (`127.0.0.1:7777`) is available for development.

The daemon exposes 14 API endpoints organized by resource:

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/sandboxes` | Create sandbox session |
| GET | `/v1/sandboxes` | List sandbox sessions |
| GET | `/v1/sandboxes/{id}` | Inspect sandbox state |
| DELETE | `/v1/sandboxes/{id}` | Destroy sandbox session |
| POST | `/v1/sandboxes/{id}/clone` | Clone repository into workspace |
| POST | `/v1/sandboxes/{id}/exec` | Execute command (streaming) |
| POST | `/v1/sandboxes/{id}/files/write` | Write file to workspace |
| GET | `/v1/sandboxes/{id}/files/read` | Read file from workspace |
| GET | `/v1/sandboxes/{id}/diff` | Get workspace diff |
| GET | `/v1/sandboxes/{id}/patch` | Get workspace patch |
| POST | `/v1/sandboxes/{id}/artifacts/export` | Export artifacts |
| POST | `/v1/sandboxes/{id}/snapshot` | Create snapshot |
| POST | `/v1/sandboxes/{id}/rollback` | Rollback to snapshot |
| GET | `/v1/sandboxes/{id}/logs` | Get command logs |

The daemon manages:
- Session lifecycle (create, warm, TTL expiration, destroy)
- Backend dispatch (delegates to fast/compat backends)
- Metadata store (sandbox state under `~/.pi/sandboxes/<id>/`)
- Command history and logging

All responses use structured JSON. Errors include actionable messages per SPEC.md §28.

The daemon is stateless in terms of business logic — it delegates to backend implementations. This allows backends to be swapped without changing the API.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-2.1: `pi-sandboxd` listens on `~/.pi/sandboxd.sock`
- [x] AC-2.2: `POST /v1/sandboxes` creates a sandbox
- [x] AC-2.3: `GET /v1/sandboxes` lists sandboxes
- [x] AC-2.4: `GET /v1/sandboxes/{id}` returns sandbox state
- [x] AC-2.5: `DELETE /v1/sandboxes/{id}` destroys a sandbox
- [x] AC-2.6: All 14 endpoints respond with JSON
- [x] AC-2.7: Errors are actionable (SPEC.md §28 format)
- [x] AC-2.8: Optional localhost HTTP on `127.0.0.1:7777` for dev

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi-sandboxd/` | New Go module for daemon |
| `~/.pi/sandboxd.sock` | Unix socket listener |
| `127.0.0.1:7777` | Optional HTTP listener (dev) |
| `pkg/api/` | HTTP handler layer |
| `pkg/daemon/` | Daemon lifecycle management |
| `SPEC.md` §9 | Interface Contract — full API surface |

## Security Considerations

- Unix socket has default permissions `0755` — restrict to owner only in production
- No authentication on local socket (assumes local user trust)
- Optional localhost HTTP is development-only; never enable in production
- No network egress from daemon itself (delegates to backends)
- Request size limits on all endpoints to prevent DoS

Reference `SPEC.md` §8 (Security Model) for sandbox security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F8: Session Lifecycle | Internal feature | ⚠️ Partially — API needs lifecycle but can be stubbed |
| Cobra (Go CLI library) | External dependency | Available |
| Unix socket library (Go stdlib) | Standard library | Available |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go daemon in `cmd/pi-sandboxd/` |
| **Configuration** | Reads `~/.pi/sandboxd.sock` path from config |

**ADR references:** None yet.
**ADR gaps:** None identified.

### Surfacing an ADR need

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| How should the daemon handle backend failures (e.g., fast backend not available)? | F2, F3, F4 | ADR-NNN: Daemon error handling and backend fallback |

## Tasks

### T2.1: Daemon binary skeleton

**Description:** Create `cmd/pi-sandboxd/` with daemon lifecycle management. Unix socket listener, optional localhost HTTP, health check endpoint (`GET /health`).

**Acceptance criteria:**
- [x] Daemon starts and listens on `~/.pi/sandboxd.sock`
- [x] `GET /health` returns `{"status": "ok"}`
- [x] Daemon creates `~/.pi/` directory if missing
- [x] Optional `--http-port` flag enables `127.0.0.1:7777`

**Verification:**
- [x] `go build ./cmd/pi-sandboxd/...`
- [x] `curl --unix ~/.pi/sandboxd.sock http://localhost/health`

**Files:** `cmd/pi-sandboxd/main.go`, `cmd/pi-sandboxd/daemon.go`, `pkg/daemon/listener.go`
**Size:** S
**Depends on:** None

### T2.2: Sandbox CRUD endpoints

**Description:** Implement `/v1/sandboxes` CRUD endpoints (POST, GET list, GET by ID, DELETE). Each returns JSON responses per the Interface Contract (SPEC.md §9).

**Acceptance criteria:**
- [x] `POST /v1/sandboxes` creates sandbox metadata and returns sandbox ID
- [x] `GET /v1/sandboxes` returns list of sandbox IDs with status
- [x] `GET /v1/sandboxes/{id}` returns full sandbox state
- [x] `DELETE /v1/sandboxes/{id}` marks sandbox for cleanup and returns 200
- [x] All endpoints return structured JSON responses

**Verification:**
- [x] `go build ./cmd/pi-sandboxd/...`
- [x] Integration tests against mock daemon

**Files:** `pkg/api/sandbox_create.go`, `pkg/api/sandbox_list.go`, `pkg/api/sandbox_get.go`, `pkg/api/sandbox_delete.go`
**Size:** M
**Depends on:** F8 (Session Lifecycle — metadata store)

### T2.3: Workspace endpoints

**Description:** Implement clone, files read/write, diff, patch endpoints.

**Acceptance criteria:**
- [x] `POST /v1/sandboxes/{id}/clone` accepts repo URL and returns task ID
- [x] `POST /v1/sandboxes/{id}/files/write` writes file to workspace
- [x] `GET /v1/sandboxes/{id}/files/read` returns file content
- [x] `GET /v1/sandboxes/{id}/diff` returns workspace diff
- [x] `GET /v1/sandboxes/{id}/patch` returns workspace patch

**Verification:**
- [x] `go build ./cmd/pi-sandboxd/...`
- [x] Integration tests

**Files:** `pkg/api/sandbox_clone.go`, `pkg/api/files_read.go`, `pkg/api/files_write.go`, `pkg/api/sandbox_diff.go`, `pkg/api/sandbox_patch.go`
**Size:** M
**Depends on:** F8 (Session Lifecycle), F6 (Workspace & File Ops)

### T2.4: Exec, artifacts, snapshot, logs endpoints

**Description:** Implement exec (streaming), artifacts export, snapshot, rollback, and logs endpoints.

**Acceptance criteria:**
- [x] `POST /v1/sandboxes/{id}/exec` accepts command, streams stdout/stderr, returns exit code/duration/truncated/timedOut
- [x] `POST /v1/sandboxes/{id}/artifacts/export` exports artifacts
- [x] `POST /v1/sandboxes/{id}/snapshot` creates snapshot
- [x] `POST /v1/sandboxes/{id}/rollback` restores snapshot
- [x] `GET /v1/sandboxes/{id}/logs` returns command history

**Verification:**
- [x] `go build ./cmd/pi-sandboxd/...`
- [x] Integration tests

**Files:** `pkg/api/sandbox_exec.go`, `pkg/api/artifacts_export.go`, `pkg/api/sandbox_snapshot.go`, `pkg/api/sandbox_rollback.go`, `pkg/api/sandbox_logs.go`
**Size:** M
**Depends on:** F7 (Command Execution), F9 (Artifact Export), F13 (Snapshot & Rollback), F10 (Logs & History)

## Verification Plan

- [x] `go build ./cmd/pi-sandboxd/...` succeeds
- [x] All 14 endpoints respond with JSON
- [x] Health check returns `{"status": "ok"}`
- [x] CRUD operations work with metadata store
- [x] Errors are actionable per SPEC.md §28

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Exec endpoint streaming protocol not fully specified | §9 Interface Contract | Add SSE or chunked transfer details |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| How should the daemon handle backend failures? | F2, F3, F4 | ADR-NNN: Daemon error handling and backend fallback |

Note: ADRs are block-level. Flag the need here; author the ADR file as a separate commit.

## Out of Scope

- Authentication (local socket assumes local trust)
- Remote daemon access (Milestone 6)
- Rate limiting (future hardening)
- Metrics/telemetry endpoints (future)
