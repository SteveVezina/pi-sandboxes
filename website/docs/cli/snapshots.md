---
sidebar_position: 4
---

# Snapshots

Snapshots are warm-start and rollback inputs for a live sandbox. They are
not a deliverable export path.

```bash
pi-box box snapshot create   <name> <snapshot-name>
pi-box box snapshot list     <name>
pi-box box snapshot rollback <name> <snapshot-name>
pi-box box snapshot delete   <name> <snapshot-name>
```

```bash
pi-box box snapshot create myapp before-refactor
# ... make changes ...
pi-box box snapshot rollback myapp before-refactor
```

Snapshot content is stored content-addressed under
`~/.pi-box/snapshots/content-addressed-store/` — two snapshots of
identical workspace state share one stored copy. On a copy-on-write
filesystem (btrfs, XFS, APFS) the copy is a reflink; elsewhere it is a
plain recursive copy. `pi-box system prune` reclaims snapshot blobs no
snapshot still references.

:::note
Turning a sandbox snapshot into a **reusable template** is a separate
feature (F28, Local Template Library) — planned, not yet implemented.
:::
