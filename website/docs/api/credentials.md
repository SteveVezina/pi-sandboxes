---
sidebar_position: 5
---

# Egress & credentials

When the daemon runs with `--egress-proxy-port <n>`, a `restricted`-mode
sandbox's outbound HTTP(S) is routed through a daemon-owned forward proxy
that enforces the per-sandbox allowlist, logs denials, and injects
approved credentials — the sandbox never sees the token.

## Network policy

Set at create time via `network.mode` (`none` / `restricted` / `open`) and
`network.allow` (extra hosts for `restricted`). The default-deny set
(cloud metadata `169.254.169.254`, host gateway, host localhost) applies
in **every** mode, including `open`.

## Register a credential

`POST /v1/credentials`

```json
{
  "id": "gh",
  "name": "ci-bot",
  "type": "git-token",
  "hosts": ["github.com"],
  "injectAs": "header",
  "valueFrom": { "env": "GITHUB_TOKEN" }
}
```

The value comes from one of `value` (literal), `valueFrom.env` (a variable
in the **daemon's** environment), or `valueFrom.file` (an absolute path
that must **not** be under `~/.pi-box`). It is held in daemon memory only —
never written to disk.

`type` is `git-token` or `registry-auth`. Returns `201 { "id": "gh" }`.

## List credentials

`GET /v1/credentials` — rules only; every `value` field is `"[redacted]"`.
The secret never appears in a response body.

## Injection

For an approved plaintext-HTTP request to a matching host, the proxy adds
an `Authorization` header:

- `git-token` → `Basic base64("x-access-token:<value>")`
- `registry-auth` → `Basic base64("<name>:<value>")`

A sandbox-supplied `Authorization` header is overwritten, not merged.

:::note
HTTPS (`CONNECT`) tunnels are end-to-end encrypted in this no-MITM
baseline, so header injection there needs an in-sandbox git credential
helper — not implemented yet. Network-layer single-endpoint isolation
(drop everything except the proxy) is also pending (Linux host).
:::
