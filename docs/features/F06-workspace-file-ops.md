# F06: Workspace & File Operations

> Source: `SPEC.md` §6 Features F6
> Status: 🟢 Implemented
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F6 | Workspace & File Operations | Clone, file read/write, diff, patch, pull/push within sandbox sessions | M1 |

## Expanded Specification

Workspace operations manage the filesystem inside sandbox sessions. The workspace is the primary data plane — where the coding agent reads, writes, and modifies code.

Operations:
1. **Clone** — Clone a Git repository into the sandbox workspace. Supports HTTPS and SSH URLs. SSH credentials are brokered (not blindly mounted).
2. **File read** — Read a file from the sandbox workspace. Returns file content as string.
3. **File write** — Write content to a file in the sandbox workspace. Creates parent directories if needed.
4. **Diff** — Compute git diff of the workspace (unstaged + staged changes). Returns unified diff format.
5. **Patch** — Export workspace changes as a git patch. Same format as diff but intended for `git apply`.
6. **Pull** — Copy files from sandbox workspace to host destination.
7. **Push** — Copy files from host to sandbox workspace.

Workspace modes (from SPEC.md §9):
- `copy` — Copy repo/files into sandbox workspace. Safest default.
- `bind` — Bind mount explicit host directory. User must opt in.
- `overlay` — Read-only base plus writable upperdir. Good for snapshots and rollback.

The workspace is managed by the backend (fast: mount namespace view; compat: container filesystem). The API layer provides a uniform interface regardless of backend.

CLI commands:
```bash
pi box clone <name> <url>
pi box files list <name> <path>
pi box files read <name> <path>
pi box files write <name> <path> <content>
pi box diff <name>
pi box patch <name>
pi box files pull <name> <src> <dest>
pi box files push <name> <src> <dest>
```

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-6.1: `pi box clone <repo>` clones a repository into sandbox workspace
- [ ] AC-6.2: `pi box files read <id> <path>` reads a file from sandbox
- [ ] AC-6.3: `pi box files write <id> <path>` writes a file to sandbox
- [ ] AC-6.4: `pi box diff <id>` shows workspace diff
- [ ] AC-6.5: `pi box patch <id>` exports workspace as patch
- [ ] AC-6.6: Clone supports HTTPS and SSH URLs
- [ ] AC-6.7: SSH credentials brokered (not blindly mounted)

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/workspace/` | Workspace management |
| `pkg/git/` | Git operations (clone, diff, patch) |
| F2: Daemon API | Workspace endpoints |
| F8: Session Lifecycle | Workspace directory management |

## Security Considerations

- SSH credentials brokered — not dumped into sandbox environment
- Clone URL validated (no file:// or git:// protocols by default)
- File paths validated against workspace directory (no path traversal)
- Bind mount mode requires explicit user opt-in
- File write creates parent directories safely (no symlink following)

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F8: Session Lifecycle | Internal feature | Available |
| Git | External tool | Required for clone/diff/patch |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go workspace/git packages |
| **Integration** | Git CLI for clone/diff/patch operations |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T6.1: Git clone

**Description:** Implement repository clone into sandbox workspace. Supports HTTPS and SSH URLs. SSH credentials brokered via credential helper.

**Acceptance criteria:**
- [ ] `pi box clone demo https://github.com/acme/app` clones repo
- [ ] `pi box clone demo git@github.com:acme/app.git` clones via SSH
- [ ] HTTPS clone uses credentials from config or prompts user
- [ ] SSH clone uses brokered credentials (not mounted into sandbox)
- [ ] Clone URL validated (rejects file://, git:// by default)
- [ ] Workspace directory created if missing

**Verification:**
- [ ] `go build ./pkg/workspace/...`
- [ ] Integration test: clone HTTPS repository
- [ ] Integration test: clone SSH repository with brokered credentials

**Files:** `pkg/workspace/clone.go`, `pkg/git/clone.go`
**Size:** M
**Depends on:** F8 (Session Lifecycle — workspace directory)

### T6.2: File read/write

**Description:** Implement file read and write operations inside sandbox workspace.

**Acceptance criteria:**
- [ ] `files read` returns file content as string
- [ ] `files write` creates parent directories if needed
- [ ] File paths validated against workspace directory (no path traversal)
- [ ] Binary files handled correctly (read as bytes, write as bytes)
- [ ] Large files streamed (not loaded entirely into memory)

**Verification:**
- [ ] `go build ./pkg/workspace/...`
- [ ] Unit tests for path traversal prevention
- [ ] Integration test: read/write files in sandbox

**Files:** `pkg/workspace/files_read.go`, `pkg/workspace/files_write.go`
**Size:** M
**Depends on:** F8 (Session Lifecycle)

### T6.3: Diff and patch

**Description:** Implement git diff and patch export from workspace.

**Acceptance criteria:**
- [ ] `pi box diff <id>` returns unified diff of workspace changes
- [ ] `pi box patch <id>` returns git patch format
- [ ] Diff includes both staged and unstaged changes
- [ ] Patch can be applied with `git apply`
- [ ] Empty workspace returns empty diff/patch

**Verification:**
- [ ] `go build ./pkg/workspace/...`
- [ ] Integration test: diff shows changes after file write
- [ ] Integration test: patch applies cleanly with `git apply`

**Files:** `pkg/workspace/diff.go`, `pkg/workspace/patch.go`, `pkg/git/diff.go`, `pkg/git/patch.go`
**Size:** S
**Depends on:** T6.1 (clone — workspace must have a repo)

### T6.4: Pull and push

**Description:** Implement file pull (sandbox → host) and push (host → sandbox) operations.

**Acceptance criteria:**
- [ ] `pi box files pull <id> <src> <dest>` copies files from sandbox to host
- [ ] `pi box files push <id> <src> <dest>` copies files from host to sandbox
- [ ] Directories copied recursively
- [ ] Progress reported for large transfers

**Verification:**
- [ ] `go build ./pkg/workspace/...`
- [ ] Integration test: pull/push files

**Files:** `pkg/workspace/files_pull.go`, `pkg/workspace/files_push.go`
**Size:** S
**Depends on:** T6.2 (file read/write)

## Verification Plan

- [ ] `go build ./pkg/workspace/...` succeeds
- [ ] Clone works for HTTPS and SSH repositories
- [ ] File read/write works with path validation
- [ ] Diff/patch works for staged and unstaged changes
- [ ] Pull/push works for files and directories
- [ ] SSH credentials never appear in sandbox environment

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| SSH credential broker mechanism not specified | §16 Secrets model | Add: "Use credential-helper or ssh-agent forwarding scoped to git" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- File watching/notifications (future)
- Real-time file sync (future)
- Archive extraction during clone (future)
- File permissions management (future)
