---
name: pi-contract-sync
description: Regenerates contract digests from upstream Pi platform specs. Use when specs have been updated (git submodule pull) and you need to refresh the compact contract files that agents load instead of full specs.
---

# Pi Contract Sync

> ⛔ **NEVER modify the block spec (`{spec_path}` per `.pi/block.yaml`) directly.**
> This skill syncs FROM upstream specs into local digests — it never writes upstream.
>
> 🔍 **Block config:** This block consumes / is consumed by other blocks listed in `.pi/block.yaml` § `upstream`. That list drives what gets synced.

## When to use

- After running `cd your-spec-submodule && git pull` (specs updated upstream)
- When a contract digest feels outdated or incomplete
- When a new upstream block interface is added that this block consumes
- When you see a mismatch between code and contract digest
- When a new upstream entry is added to `.pi/block.yaml` § `upstream`

## Instructions

1. Check what changed in the specs submodule:
   ```bash
   cd your-spec-submodule && git log --oneline -5
   git diff HEAD~1 -- blocks-specs/
   ```

2. For each upstream block listed in `.pi/block.yaml` § `upstream`, re-extract the interface into `docs/contracts/{block}.md`. Typical mapping:
   - `block-orchestrator.md` → `docs/contracts/orchestrator.md`
   - `block-session-manager.md` → `docs/contracts/session-manager.md`
   - `block-gateway.md` → `docs/contracts/gateway.md`
   - `block-workspaces.md` → `docs/contracts/workspaces.md`
   - `block-secret-manager.md` → `docs/contracts/secret-manager.md`

3. **Cross-check against the real implementation** (per `AGENTS.md` Context Loading Protocol Step 5). The OpenAPI spec / DAT.md file in `upstream block paths (see .pi/block.yaml § upstream){block}/` is the source of truth for endpoint shapes, **not** just the block spec prose. Paths are listed in `.pi/block.yaml` § `upstream`.

4. Extract ONLY:
   - Endpoints this block calls or receives
   - Request/response payloads relevant to this block
   - Authentication requirements
   - Key behavioral contracts

5. Do NOT include:
   - Endpoints other consumers use
   - Internal implementation details of upstream blocks
   - Features irrelevant to this block

6. Update the "Last synced" date in each digest header.

7. If a new upstream dependency appears, create a new digest file **and** add an entry to `.pi/block.yaml` § `upstream`.

## Contract Digest Template

```markdown
# Contract Digest: [Upstream Block] → [This Block]

> Extracted from `your-spec-submodule/blocks-specs/[block-file].md`
> Cross-checked against: `upstream block paths (see .pi/block.yaml § upstream)[block]/[real-spec-path]`
> Last synced: YYYY-MM-DD

## What [this block] [sends to / receives from] [upstream block]

### `METHOD /path` — Description

```json
{ payload }
```

## Key Behaviors

- Bullet points of important contract rules
```

## Guardrails

- **NEVER** modify upstream specs (`your-spec-submodule/` is read-only from this block's perspective).
- **NEVER** include endpoints this block doesn't actually use — keeps digests focused.
- **ALWAYS** verify the digest against the real implementation file in `upstream block paths (see .pi/block.yaml § upstream)`, not just the block spec prose.
- **ALWAYS** update `Last synced` and the commit hash if applicable, so reviewers can tell digest freshness.
