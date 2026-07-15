# Gap Analysis: F1-F19 vs SPEC.md

> Generated: 2026-06-26
> Scope: F1 (CLI) through F19 (Runtime Selection)
> Excludes: F20-F23 (M5-M6 — owned by another agent)

---

## Summary

| Feature | Status | Gap Severity | Notes |
|---------|--------|-------------|-------|
| F1: CLI Entry Point | ⚠️ Partial | Medium | Commands registered but most are stubs |
| F2: Daemon API | ⚠️ Partial | Medium | Only 5 of 13 required endpoints wired |
| F3: Fast Backend | ✅ Complete | None | Full implementation + stubs |
| F4: Compat Backend | ✅ Complete | None | OCI runtime detection + lifecycle |
| F5: Template System | ✅ Complete | None | All 7 templates defined |
| F6: Workspace & File Ops | ✅ Complete | None | Clone, read, write, diff, patch |
| F7: Command Execution | ✅ Complete | None | Timeout, truncation, exit codes |
| F8: Session Lifecycle | ✅ Complete | None | Meta, state machine, TTL, orphans |
| F9: Artifact Export | ✅ Complete | None | List, pull, pack |
| F10: Logs & History | ✅ Complete | None | Entries, listing |
| F11: Secrets & Network | ✅ Complete | None | Modes, allowlist, broker |
| F12: Cache Model | ✅ Complete | None | Scoped mounts, pruning |
| F13: Snapshot & Rollback | ✅ Complete | None | Create, list, rollback, delete |
| F14: Benchmarks | ⚠️ Partial | Low | 13 defined but most are stubs |
| F15: SDKs | ✅ Complete | None | TS + Python clients |
| F16: System Commands | ✅ Complete | None | Status, doctor, prune, disk-usage |
| F17: Policy Enforcement | ✅ Complete | None | Defaults match SPEC.md §22 |
| F18: Secure Backend | ✅ Complete | None | gVisor runtime + stubs |
| F19: Runtime Selection | ✅ Complete | None | Detection, fallback, docs |

---

## Detailed Gaps

### F1: CLI Entry Point — ⚠️ Partial (Medium)

**What's done:**
- All CLI commands registered: `create`, `list`, `inspect`, `destroy`, `clone`, `exec`, `shell`, `files`, `diff`, `patch`, `artifacts`, `snapshot`, `logs`
- `files` subcommand has: `list`, `read`, `write`
- `artifacts` subcommand has: `list`, `pull`, `pack`
- `snapshot` subcommand has: `create`, `list`, `rollback`, `delete`

**Gaps:**
- Most top-level commands (`create`, `list`, `inspect`, `destroy`, `clone`, `exec`, `shell`, `diff`, `logs`) are stubs that print "not yet implemented" and exit 1
- Files subcommands (`list`, `read`, `write`) are stubs
- Artifacts subcommands (`list`, `pull`, `pack`) are stubs
- Snapshot subcommands (`create`, `list`, `rollback`, `delete`) are stubs

**SPEC requirement (Section 12):**
- `pi-box box create` — stub
- `pi-box box list` — stub
- `pi-box box inspect` — stub
- `pi-box box clone` — stub
- `pi-box box exec` — stub (but API handler exists)
- `pi-box box files list/read/write` — stub
- `pi-box box artifacts list/pull/pack` — stub
- `pi-box box snapshot create/list/rollback/delete` — stub
- `pi-box box diff` — stub
- `pi-box box logs` — stub

**Impact:** The CLI is a skeleton. The daemon API has real handlers for CRUD + exec, but the CLI doesn't wire through to them.

---

### F2: Daemon API — ⚠️ Partial (Medium)

**What's done:**
- `POST /v1/sandboxes` — CreateSandbox ✅
- `GET /v1/sandboxes` — ListSandboxes ✅
- `GET /v1/sandboxes/{id}` — GetSandbox ✅
- `DELETE /v1/sandboxes/{id}` — DeleteSandbox ✅
- `POST /v1/sandboxes/{id}/exec` — ExecSandbox ✅
- `GET /health` — HealthHandler ✅

**Gaps (8 missing endpoints):**
- ❌ `POST /v1/sandboxes/{id}/clone` — no handler
- ❌ `POST /v1/sandboxes/{id}/files/write` — no handler
- ❌ `GET /v1/sandboxes/{id}/files/read` — no handler
- ❌ `GET /v1/sandboxes/{id}/diff` — no handler
- ❌ `GET /v1/sandboxes/{id}/patch` — no handler
- ❌ `POST /v1/sandboxes/{id}/artifacts/export` — no handler
- ❌ `POST /v1/sandboxes/{id}/snapshot` — no handler
- ❌ `POST /v1/sandboxes/{id}/rollback` — no handler
- ❌ `GET /v1/sandboxes/{id}/logs` — no handler

**Impact:** The daemon can create, list, get, delete, and exec sandboxes. But clone, file ops, diff, patch, artifacts, snapshot, rollback, and logs are all missing from the API layer. The underlying packages exist (`pkg/workspace/`, `pkg/artifacts/`, `pkg/snapshot/`, `pkg/logs/`) but aren't wired through the HTTP API.

---

### F14: Benchmarks — ⚠️ Partial (Low)

**What's done:**
- All 13 benchmark names defined
- Framework exists (`pkg/bench/framework.go`)
- `warm_exec_echo`, `warm_exec_shell`, `file_scan_rg`, `artifact_export_20mb` have real implementations

**Gaps:**
- `git_clone_small` — stub (sleep 10ms)
- `pnpm_install_cached` — stub (sleep 10ms, disabled)
- `uv_sync_cached` — stub (sleep 10ms, disabled)
- `go_test_cached` — stub (sleep 10ms, disabled)
- `cargo_test_cached` — stub (sleep 10ms, disabled)
- `snapshot_create` — stub (sleep 10ms, disabled)
- `snapshot_rollback` — stub (sleep 10ms, disabled)
- `parallel_10` — stub (sleep 100ms, disabled)
- `parallel_100` — stub (sleep 500ms, disabled)

**Impact:** Benchmarks exist as definitions but most measure nothing real. The framework can run them but results are artificial.

---

### F20-F21: MicroVM — ⚠️ Partial (Low priority, other agent's scope)

**What's done (by current agent):**
- `pkg/runtime/microvm/protocol.go` — Frame codec ✅ (NDJSON, base64 streams, exec request/result)
- `pkg/runtime/microvm/vsock.go` — vsock client stub ✅ (Linux build tag)
- `pkg/runtime/microvm/vsock_stub.go` — vsock stub ✅ (non-Linux)
- `pkg/runtime/microvm/exec.go` — exec engine ✅ (Linux build tag)

**Gaps (owned by other agent per plan):**
- ❌ `cmd/pi-vmm-manager/` — binary not created
- ❌ `pkg/runtime/microvm/firecracker.go` — Firecracker lifecycle
- ❌ `pkg/runtime/microvm/disk.go` — workspace disk management
- ❌ `pkg/runtime/microvm/snapshot.go` — template snapshot restore
- ❌ `pkg/runtime/microvm/artifacts.go` — artifact export via guest control plane
- ❌ `cmd/pi-init/` — guest init binary
- ❌ `cmd/pi-agentd/` — guest agent binary
- ❌ `pkg/runtime/microvm/files.go` — file transfer protocol
- ❌ `pkg/runtime/microvm/runtime.go` — microVM runtime interface

---

### F22-F23: Remote Daemon — Not Started (Other agent's scope)

- ❌ `pi-box context` CLI commands
- ❌ Remote transport (SSH/Tailscale/WireGuard)
- ❌ Context persistence (`~/.pi-box/contexts.yaml`)

---

## Cross-Cutting Issues

### 1. CLI → API wiring gap (F1 + F2)
The CLI commands exist as cobra commands but call `os.Exit(1)` instead of making HTTP calls to the daemon API. The daemon API has handlers for CRUD + exec but is missing 8 endpoints. Both need fixing:
- Add 8 API handler files
- Wire CLI commands through to the daemon API

### 2. Benchmark stubs (F14)
Most benchmarks use `time.Sleep()` instead of real operations. This is acceptable for MVP but needs real implementations before performance targets can be validated.

### 3. SDK streaming support (F15)
SPEC says "SDKs must support streaming output" but SDK clients may not implement SSE/streaming for exec responses.

### 4. Template building (F5)
Template `build`, `update`, `prune` CLI commands are stubs. Template definitions exist in `pkg/template/defaults.go` but the build pipeline isn't wired.

---

## Recommendations

### Critical (blocks AC-1, AC-6, AC-7, AC-8, AC-9, AC-10)
1. **Wire 8 missing API endpoints** — create handler files for clone, files, diff, patch, artifacts, snapshot, rollback, logs
2. **Wire CLI commands through to daemon API** — replace stubs with HTTP client calls

### Important (blocks AC-14)
3. **Implement real benchmark functions** — replace `time.Sleep()` with actual operations

### Nice-to-have (future)
4. Implement template `build`/`update`/`prune`
5. Add SDK streaming support
6. Complete F20-F23 (owned by other agent)
