---
name: pi-review-prop
description: Reviews a proposed PROP before human acceptance — checks completeness, block-spec accuracy, impact analysis, and downstream consistency. Use when a PROP has been written and needs validation before "accept PROP-{NNN}" is approved.
---

# Pi Review Prop

> Validates that a `PROP-{NNN}` is ready for human acceptance.
> Catches missing fields, inaccurate block-spec references, incomplete
> impact analysis, and conflicts with accepted ADRs **before** the
> cascade in `pi-apply-prop` runs.

> 📂 **Document placement rule:** Review verdict is inline (in the response) only.
> Never write files to `docs/reviews/`, `docs/process/`, or `docs/plans/`.

## When to use

- A new PROP has been authored and needs validation before the human is asked to accept it
- A pending PROP has been sitting in `docs/proposals/` and needs a freshness check
- Before invoking `/skill:pi-apply-prop` — review first, apply second

**When NOT to use:**
- Applying an already-accepted PROP → use `/skill:pi-apply-prop`
- Writing a new PROP from scratch → write it directly, then invoke this skill

## Process

### Step 1: Load context

```
1. docs/proposals/PROP-{NNN}-*.md          → the PROP under review
2. docs/proposals/TEMPLATE.md              → expected structure
3. docs/proposals/INDEX.md                 → status + dependency tags
4. {spec_path}  → verify referenced sections exist
5. docs/decisions/                         → check ADR conflicts
6. docs/features/                          → check feature spec impact accuracy
7. upstream block paths (see .pi/block.yaml § upstream)              → cross-check against upstream block implementations
```

### Step 2: Structural completeness check

A valid PROP must contain all of these. Flag any that's missing or empty:

| Field | Required content |
|-------|-----------------|
| **Status** | One of: 🟡 Proposed, 🟡 Partially resolved, ✅ Accepted, ✅ Applied to block spec, ❌ Withdrawn |
| **Block Spec Reference** | File path + section name; the section must actually exist in `the block spec` |
| **Problem** | What gap, ambiguity, or error this PROP addresses (with evidence) |
| **Proposed Amendment** | Exact text to add/modify/delete in the block spec (copy-paste-ready) |
| **Rationale** | Why this change is correct, not just "the code already does this" |
| **Impact** | List of: Features affected, ADRs affected, Implementation blocked? (yes/no) |
| **Assumption (if non-blocking)** | What the agent will assume while awaiting acceptance |
| **Requested By** | Date + context |

### Step 3: Block spec accuracy

For every block spec quote / reference in the PROP:

1. **Section exists?** Verify the referenced `§ Section` actually exists in `the block spec`.
2. **Quote accurate?** If the PROP quotes "Current text", confirm it matches the live spec character-for-character. Inaccurate quotes mean the PROP was written against a stale version — flag for refresh.
3. **Proposed amendment well-formed?** The "Proposed Amendment" must be copy-paste-ready text. If it says "Add something about X", that's not a real amendment — flag as a draft.
4. **No spec drift sneaking in.** Confirm the amendment changes only what the PROP claims to change — no piggy-backed edits.

### Step 4: ADR conflict check

For every ADR cited in the PROP:

1. Read the ADR (or at minimum its `## Decision` and `## Status` sections).
2. Check whether the proposed amendment **contradicts** the ADR's decision.
3. If a contradiction exists:
   - **The ADR must be deprecated or amended** in the same cascade.
   - The PROP must list the ADR in `## Impact` under "ADRs affected" with the planned outcome (deprecated / amended / superseded).
   - If the PROP claims "ADRs affected: none" but a contradiction is found → flag as INCOMPLETE.

### Step 5: Feature spec impact accuracy

For every feature `F{N}` listed in the PROP's `## Impact`:

1. Open `docs/features/F{N}-*.md`.
2. Confirm the feature genuinely depends on the changed area. If listed but unaffected → flag as inflated impact.
3. Check whether feature **specs that should be listed but aren't** — search the feature directory for the old assumption:
   ```bash
   grep -rn "{old_text}" docs/features/
   ```
   Any hit not in the PROP's impact list → flag as missing impact.

### Step 6: Cross-check upstream block implementations

Per `AGENTS.md` Step 5: real OpenAPI specs and DAT.md files are the source of truth, not just block spec prose.

For PROPs that touch interfaces / contracts:

```bash
ls upstream block paths (see .pi/block.yaml § upstream)
# Session Manager: upstream block paths (see .pi/block.yaml § upstream)sessionmanager/api/openapi/sessionmanager/sessionmanager.yaml
# Workspaces:      upstream block paths (see .pi/block.yaml § upstream)workspaces/doc/api-contract.md
# Secret Manager:  upstream block paths (see .pi/block.yaml § upstream)secretmanager/docs/DAT.md
# Orchestrator:    upstream block paths (see .pi/block.yaml § upstream)orchestrator/docs/dat.md
```

Verify the proposed endpoint shapes, field names, and status codes match what the **upstream** block actually implements. If the PROP proposes an endpoint the upstream block doesn't have, flag it.

### Step 7: PROP Queue Guard

Per `AGENTS.md` § PROP Queue Guard:

1. Check `docs/proposals/INDEX.md` for any **other** PROPs with status `🟡 Proposed` or `🟡 Partially resolved`.
2. If unapproved PROPs exist and this PROP is **not** marked as a blocking implementation blocker → flag: pile-up risk. Recommend the human accept/deny the older PROPs first.
3. If the PROP is marked blocking, verify it notes the pending PROPs: "⚠️ BLOCKING — pending human approval. Existing unapproved PROPs: {list}."

### Step 8: PROP Numbering Guard

Per `AGENTS.md` § PROP Numbering Guard:

```bash
ls docs/proposals/PROP-*.md | xargs -I{} basename {} | sort
```

1. The PROP number must be `last_existing_number + 1` — no gaps, no duplicates.
2. If a number was withdrawn, reuse it only if the withdrawal was < 7 days ago and documented.
3. Verify `docs/proposals/INDEX.md` has a row for this PROP. If missing → flag.

### Step 9: Conflict with already-accepted PROPs

If two unapproved PROPs modify overlapping block spec sections, flag for human attention. Acceptance order will matter, and the cascade in one may invalidate the amendment in the other.

### Step 10: Produce verdict

```markdown
## Review: PROP-{NNN} — {title}

**Status under review:** {current status in PROP file}
**Block spec section:** {referenced section}

### 🔴 Blockers (must fix before acceptance)
- {issue + suggested fix}

### 🟡 Warnings (should fix, non-blocking)
- {issue + suggestion}

### 🟢 Strengths
- {what's well done}

### Verdict: READY_TO_ACCEPT | NEEDS_REVISION | BLOCKED

**Confidence:** {High | Medium | Low}
**Recommended next action:**
- READY_TO_ACCEPT → human accepts → `/skill:pi-apply-prop`
- NEEDS_REVISION → author addresses blockers → re-review
- BLOCKED → conflict with another PROP or ADR; resolve order first
```

## Guardrails

- **NEVER** modify the PROP file during review — only produce a verdict. The author addresses issues.
- **NEVER** apply the PROP from this skill — that's `pi-apply-prop`'s job after human acceptance.
- **NEVER** approve a PROP whose block spec quote doesn't match the live spec.
- **NEVER** approve a PROP that contradicts an accepted ADR without listing the ADR in `## Impact`.
- **NEVER** approve a PROP that violates the PROP Queue Guard or Numbering Guard.

## Related skills

- `/skill:pi-apply-prop` — apply an accepted PROP (cascade into specs)
- `/skill:pi-feature-spec` — author / revise a feature spec (which can surface gaps that become PROPs)
- `/skill:pi-review-spec` — review a feature spec (different scope — features, not block-spec amendments)
