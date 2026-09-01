---
sidebar_position: 5
---

# Logs & history

## logs

```bash
pi-box box logs <name>
```

Full command log entries for the sandbox — each with the command, exit
code, duration, timeout / truncation status, and the captured
stdout/stderr.

## history

```bash
pi-box box history <name>
```

A summary of executed commands (no stdout/stderr bodies): sequence,
command, exit code, duration.

## egress

```bash
pi-box box egress <name>
```

The egress-proxy **denials** recorded for the sandbox — one line per
denied outbound request (`timestamp`, `host`, `reason`). Only populated
when the daemon runs with `--egress-proxy-port` and the sandbox is in
`restricted` network mode. Credential material never appears in this log.
