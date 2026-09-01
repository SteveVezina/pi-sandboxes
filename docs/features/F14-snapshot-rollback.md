# F13: Snapshot & Rollback

> Source: `SPEC.md` §6 Features F13
> Status: ⚠️ Needs re-verify
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F13 | Snapshot & Rollback | Filesystem-level snapshot creation and rollback (overlay/reflink) | M2 |

## Expanded Specification

Snapshots provide point-in-time copies of a sandbox's filesystem state, enabling rollback to previous states. This is critical for the coding agent inner loop — the agent can snapshot before a risky operation (refactor, major dependency install) and rollback if things go wrong.

Initial implementation uses filesystem-level snapshots:
- **Overlay upperdir copy** — Copy the overlay upperdir (fast mode)
- **Reflink copy** — When filesystem supports it (btrfs, xfs)
- **tar/zstd fallback** — When reflink not available

Snapshot metadata is stored in a daemon-owned content-addressed store under `~/.pi-box/snapshots/`. Snapshots are warm-start and rollback inputs only; they are not an export channel.

Operations:
1. **Create** — Create a named snapshot of the current workspace state
2. **List** — List all snapshots for a sandbox
3. **Rollback** — Restore workspace to a named snapshot
4. **Delete** — Remove a snapshot

CLI commands:
```bash
pi-box box snapshot <name> <action> [name]
pi-box box snapshots <name>
pi-box box rollback <name> <snapshot-name>
pi-box box snapshot delete <name> <snapshot-name>
```

MicroVM mode (future) should support:
- Template snapshots
- Snapshot-first template restore
- Explicit reseed-on-restore hook

For now, the MVP uses filesystem-level snapshots.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-13.1: `pi-box box snapshot <id> <name>` creates a named snapshot
- [x] AC-13.2: `pi-box box rollback <id> <name>` restores to snapshot
- [x] AC-13.3: Snapshot creation uses overlay upperdir or reflink
- [ ] AC-13.4: Snapshot metadata stored in a daemon-owned content-addressed store under `~/.pi-box/snapshots/` *(2026-07-15: AC updated per PROP-009)*
- [ ] AC-13.5: Snapshots are warm-start and rollback inputs only, not an export channel *(2026-07-15: added per PROP-009)*

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/snapshot/` | Snapshot management |
| `~/.pi-box/snapshots/` | Daemon-owned content-addressed snapshot storage |
| F8: Sandbox Lifecycle | Snapshot lifecycle management |
| F3: Fast Backend | Overlay upperdir snapshot |
| F4: Compat Backend | Container snapshot (future) |

## Security Considerations

- Snapshots stored under `~/.pi-box/` with restricted permissions
- Rollback overwrites workspace — warn user before destructive operation
- Snapshot size validated (prevent disk exhaustion)
- No symbolic link following during snapshot creation

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F8: Sandbox Lifecycle | Internal feature | Available |
| F3: Fast Backend | Internal feature | Overlay upperdir snapshot |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go snapshot package |
| **Infrastructure** | Overlayfs/reflink filesystem operations |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T14.1: Snapshot creation ⚠️

**Description:** Implement snapshot creation using overlay upperdir copy or reflink into daemon-owned content-addressed storage. *(2026-07-15: storage/export semantics updated per PROP-009.)*

**Acceptance criteria:**
- [x] `pi-box box snapshot demo before-refactor` creates named snapshot
- [x] Snapshot uses overlay upperdir copy (fast mode)
- [x] Snapshot uses reflink if filesystem supports it (btrfs, xfs)
- [x] Snapshot falls back to tar/zstd if reflink unavailable
- [ ] Snapshot metadata stored under daemon-owned `~/.pi-box/snapshots/`
- [ ] Snapshot cannot be used as a deliverable export path
- [x] Snapshot size validated before creation (prevent disk exhaustion)

**Verification:**
- [x] `go build ./pkg/snapshot/...`
- [x] Integration test: snapshot created successfully
- [x] Integration test: snapshot uses reflink on supported filesystem

**Files:** `pkg/snapshot/create.go`, `pkg/snapshot/metadata.go`
**Size:** M
**Depends on:** F8 (Sandbox Lifecycle)

### T14.2: Snapshot listing and rollback

**Description:** Implement snapshot listing and rollback.

**Acceptance criteria:**
- [x] `pi-box box snapshots demo` lists all snapshots with timestamps
- [x] `pi-box box rollback demo before-refactor` restores workspace to snapshot
- [x] Rollback overwrites workspace with snapshot contents
- [x] Rollback asks for confirmation (destructive operation)
- [x] `pi-box box snapshot delete demo before-refactor` removes snapshot

**Verification:**
- [x] `go build ./pkg/snapshot/...`
- [x] Integration test: rollback restores correct state
- [x] Integration test: delete removes snapshot

**Files:** `pkg/snapshot/list.go`, `pkg/snapshot/rollback.go`, `pkg/snapshot/delete.go`
**Size:** M
**Depends on:** T14.1 (snapshot creation)

## Verification Plan

- [x] `go build ./pkg/snapshot/...` succeeds
- [ ] Snapshot creation works with overlay/reflink/tar fallback into daemon-owned storage
- [x] Snapshot listing shows all snapshots
- [x] Rollback restores correct state
- [x] Delete removes snapshot
- [x] Benchmark: snapshot_create < 500ms (SPEC.md §19)
- [x] Benchmark: snapshot_rollback < 500ms (SPEC.md §19)

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Snapshot fallback strategy not specified | §18 Snapshot and rollback | Add: "Use overlay upperdir copy; fall back to tar/zstd if reflink unavailable" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| How to handle snapshot rollback when workspace has uncommitted changes? | F14 | ADR-NNN: Snapshot rollback conflict resolution |

## Out of Scope

- **Snapshotting a sandbox into a reusable template** — that is F28 (PROP-006, `SPEC.md` §18.1). F13 covers warm-start / rollback snapshots of a live sandbox; F28's template snapshot reuses these primitives but produces a daemon-managed template asset.
- MicroVM template snapshots (Milestone 5)
- Snapshot compression (future)
- Snapshot diff/comparison (future)
- Snapshot sharing between sandboxes (future)
- Automatic snapshot before risky operations (future)
