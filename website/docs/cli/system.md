---
sidebar_position: 6
---

# System

Local state inspection and maintenance for `~/.pi-box`.

```bash
pi-box system status       # daemon connection + active/total sandbox counts
pi-box system doctor       # validate config, dirs, disk space, permissions, runtimes
pi-box system prune        # remove destroyed-sandbox state, orphaned data, logs > 30 days
pi-box system disk-usage   # size breakdown by sandboxes / templates / caches / images / logs
```

`system prune` asks for confirmation before destructive operations; pass
`--yes` to skip it.

`system doctor` also creates a default `~/.pi-box/config.yaml` if missing
(non-destructive) and reports which runtime backends are usable on this
host.
