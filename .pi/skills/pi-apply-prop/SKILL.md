---
name: pi-apply-prop
description: Apply an accepted proposal — modify block spec, cascade into ADRs and feature specs, update INDEX files, reset stale task statuses. Use when a human says "accept PROP-{NNN}", "approve PROP-{NNN}", or "apply PROP-{NNN}".
---

# Pi Apply Prop

> Automates the **proposal acceptance cascade**. When a human approves a PROP, this skill:
> 1. Applies the changes to `{spec_path}` (the only sanctioned path to modify the block spec)
> 2. Cascades into affected ADRs
> 3. Cascades into affected feature specs and their `## Tasks` tables
> 4. Updates INDEX files
> 5. Updates the proposal record
> 6. Verifies cascade completeness

> 📂 **Document placement rule:** This skill writes only to existing locations
> (`{spec_path}`, `docs/decisions/`, `docs/features/`,
> `docs/proposals/`, `docs/plan.md`). Never create files under
> `docs/plans/`, `docs/process/`, or `docs/reviews/`.

## When to use

- Human says "accept PROP-{NNN}", "approve PROP-{NNN}", "apply PROP-{NNN}", or "merge PROP-{NNN}"

## Process

### Step 1: Read the PROP

```bash
cat docs/proposals/PROP-{NNN}-*.md
```

Extract:
- **Block spec changes** — exact text to add/modify/delete
- **ADRs affected** — listed in `## Impact`
- **Features affected** — listed in `## Impact`
- **Rationale** — to use in commit message

### Step 2: Apply block spec changes (only sanctioned path)

> **This is the ONLY case where the agent may modify `{spec_path}`.**

The block spec lives in the `your-spec-submodule` **git submodule**. Commit there independently before committing in the parent repo.

1. Apply edits exactly as specified in the PROP.
2. Verify diff:
   ```bash
   cd your-spec-submodule && git diff blocks-specs/the block spec
   ```
3. Commit inside the submodule:
   ```bash
   cd your-spec-submodule && git add blocks-specs/the block spec && git commit -m "spec: apply PROP-{NNN} — {title}"
   ```

If the PROP requires no block spec changes (e.g., resolved by ADR only), skip to Step 3.

### Step 3: Cascade into affected ADRs

For every ADR listed in the PROP's `## Impact`:

1. **Read the ADR in full.** Find every paragraph, example, constraint, or diagram that reflected the old assumption.
2. **Update or annotate:**
   - Decision still valid, constraint changed → update the section + add dated note: `*(Updated YYYY-MM-DD per PROP-{NNN}: {what changed})*`
   - Decision partially superseded → add `## Amendments` section at the bottom listing what changed, with link to the new ADR or to `the block spec`
   - Decision fully superseded → set `## Status` to `Deprecated — superseded by ADR-{NNN}` and add a one-line note at the top
3. **Never delete historical content** — strike through or annotate. Future readers need to understand what changed and why.

### Step 4: Cascade into affected feature specs

For every feature in the PROP's `## Impact` (and every feature listed in ADRs updated in Step 3):

1. Open `docs/features/F{N}-*.md`
2. Check every section that could reference the old assumption:
   - **Status line**
   - **Expanded Specification**
   - **Acceptance Criteria**
   - **Implementation Approach**
   - **Tasks** (the `## Tasks` table)
   - **Spec Gaps**
   - **Verification Plan**
3. Update each section to reflect the new decision.
4. **Reset task statuses to ⚠️** for any task whose AC or implementation approach changed.
   - A task row that says ✅ while its feature spec AC has changed is a lie. Fix it in the **same commit** as the spec change.
   - Add a dated note to the task row: `*(2026-MM-DD: AC updated per PROP-{NNN})*`
5. Update the spec's `Status:` line to 🟡 if any task was reset.

### Step 5: Update INDEX files

- [ ] `docs/features/INDEX.md`:
  - For every feature whose spec changed in Step 4 → Spec Status → 🟡
  - For every feature with reset tasks → Impl Status → 🟡 or ⚠️
  - Update dependency descriptions that referenced the old mechanism
- [ ] `docs/proposals/INDEX.md`:
  - Update the PROP's status to `✅ Applied to block spec` (or `✅ Resolved by ADR` if no block spec changes)

### Step 6: Update the proposal record

- [ ] In `docs/proposals/PROP-{NNN}-*.md`:
  - Set `## Status` to `✅ Applied to block spec (YYYY-MM-DD)` (or `✅ Resolved (YYYY-MM-DD)`)
  - Add a `## Cascade completed` section listing which ADRs, feature specs, and plan rows were updated and on what date

### Step 7: Update plan.md Active Cursor (if relevant)

If the cascade changes what is currently in progress (a previously blocked feature now unblocked, or a new feature now in scope):

- [ ] Update `docs/plan.md` **Active Cursor** to reflect the new open work

### Step 8: Cascade completeness check

Verify no normative reference to the old assumption remains:

```bash
# Replace {old_text} with a distinctive phrase from the old assumption
grep -rn "{old_text}" docs/features/ docs/decisions/
# Expected: 0 matches (or only in clearly historical / struck-through / annotated context)
```

If any normative hit is found → update it before considering the cascade done.

### Step 9: Commit (parent repo)

```bash
cd {repo-root}
git add docs/decisions/ docs/features/ docs/proposals/ docs/plan.md your-spec-submodule
git commit -m "feat: apply PROP-{NNN} — {title}

Accepted by human on {date}.

- Block spec: {one-line summary}
- ADRs cascaded: ADR-{X}, ADR-{Y}
- Feature specs cascaded: F{N}, F{M}
- Tasks reset to ⚠️: T{X}.{Y}, T{X}.{Z} (AC changed)
- INDEX files updated"
```

## Guardrails

- **NEVER** modify the block spec without a human-approved PROP.
- **NEVER** skip the ADR/feature-spec cascade after a block spec change. An accepted proposal that has not been cascaded is documentation debt that silently corrupts future feature work.
- **NEVER** leave INDEX files out of sync with actual state.
- **NEVER** delete historical content from an ADR — strike through or annotate.
- **ALWAYS** commit submodule (`your-spec-submodule`) and parent repo separately.
- **ALWAYS** reset task status to ⚠️ when its AC changes — same commit as the spec edit.
- **ALWAYS** run Step 8 completeness check before considering the cascade done.

## Why this cascade exists

This skill was extracted from `pi-runtime-architect` after the ADR-004/ADR-008/PROP-005 sequence showed that accepting a topology change without cascading it left F5, F6, F12, F13, ADR-002, and ADR-004 partially stale for multiple sessions. The cascade is not optional — it is the atomic unit that makes a PROP acceptance "real."

## Related skills

- `/skill:pi-review-prop` — review a proposed PROP before acceptance
- `/skill:pi-feature-spec` — author / revise a feature spec (which can surface ADR needs)
- `/skill:pi-execute-plan` — resume implementation work after a cascade
- `/skill:pi-contract-sync` — regenerate contract digests if upstream changed
