# F15: SDKs

> Source: `SPEC.md` §6 Features F15
> Status: 🟢 Implemented (re-verified 2026-08-31 — all ACs hold, targeted tests pass, no AC-masking skips)
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F15 | SDKs | TypeScript and Python SDKs with streaming output support | M3 |

## Expanded Specification

TypeScript and Python SDKs provide programmatic access to the sandbox runtime API. They wrap the HTTP API and provide a developer-friendly interface for coding agents and other tools.

TypeScript SDK:
```ts
const box = await client.sandboxes.create({
  template: "node-python",
  mode: "fast"
});

await box.clone("https://github.com/acme/app");
const result = await box.exec("pnpm test", { timeoutMs: 60000 });
const diff = await box.diff();
```

Python SDK:
```python
box = client.sandboxes.create(template="python", mode="fast")
box.clone("https://github.com/acme/app")
result = box.exec("uv run pytest -q", timeout_ms=60000)
diff = box.diff()
```

Both SDKs must support:
- Streaming output (stdout/stderr via callbacks or async iterators)
- All API operations (create, clone, exec, files, diff, patch, artifacts, snapshot, rollback, logs)
- Error handling with actionable messages (SPEC.md §28)
- JSON mode for machine consumption

SDK packages:
- `sdk/typescript/` — TypeScript SDK
- `sdk/python/` — Python SDK

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-15.1: TypeScript SDK: `client.sandboxes.create()`, `.clone()`, `.exec()`, `.diff()`
- [x] AC-15.2: Python SDK: `client.sandboxes.create()`, `.clone()`, `.exec()`, `.diff()`
- [x] AC-15.3: Both support streaming output

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `sdk/typescript/` | TypeScript SDK |
| `sdk/python/` | Python SDK |
| F2: Daemon API | SDKs consume the API |

## Security Considerations

- SDKs connect to local Unix socket (same as CLI)
- No secrets in SDK API keys (local trust model)
- Streaming output handled securely (no secrets in logs)

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F2: Daemon API | Internal feature | SDKs consume the API |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | TypeScript + Python SDK packages |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T16.1: TypeScript SDK

**Description:** Implement TypeScript SDK with all sandbox operations and streaming output.

**Acceptance criteria:**
- [x] `client.sandboxes.create()` creates sandbox
- [x] `box.clone()` clones repository
- [x] `box.exec()` executes command with streaming output
- [x] `box.diff()` gets workspace diff
- [x] All API operations implemented
- [x] Streaming output via callbacks or async iterators
- [x] Error handling with actionable messages

**Verification:**
- [x] `npm install` succeeds in `sdk/typescript/`
- [x] TypeScript compilation succeeds
- [x] Unit tests pass

**Files:** `sdk/typescript/src/`, `sdk/typescript/package.json`, `sdk/typescript/tsconfig.json`
**Size:** M
**Depends on:** F2 (Daemon API)

### T16.2: Python SDK

**Description:** Implement Python SDK with all sandbox operations and streaming output.

**Acceptance criteria:**
- [x] `client.sandboxes.create()` creates sandbox
- [x] `box.clone()` clones repository
- [x] `box.exec()` executes command with streaming output
- [x] `box.diff()` gets workspace diff
- [x] All API operations implemented
- [x] Streaming output via generators or callbacks
- [x] Error handling with actionable messages

**Verification:**
- [x] `pip install -e sdk/python/` succeeds
- [x] Python type checking passes
- [x] Unit tests pass

**Files:** `sdk/python/src/`, `sdk/python/pyproject.toml`, `sdk/python/README.md`
**Size:** M
**Depends on:** F2 (Daemon API)

## Verification Plan

- [x] TypeScript SDK compiles and tests pass
- [x] Python SDK installs and tests pass
- [x] Both SDKs support streaming output
- [x] Both SDKs implement all API operations
- [x] Error messages are actionable per SPEC.md §28

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| SDK streaming API not fully specified | §14 Agent SDK requirements | Add: "TypeScript: async iterators; Python: generators" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- SDK for other languages (future)
- SDK authentication (local trust model)
- SDK connection pooling (future)
- SDK retry logic with backoff (future)
