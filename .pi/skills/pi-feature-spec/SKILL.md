---
name: pi-feature-spec
description: Creates detailed feature specs for a Pi block. Use when starting work on a feature, when a feature lacks a spec, or when extracting requirements from the block spec (see `.pi/block.yaml` § `block.spec_path`) into an actionable feature document with acceptance criteria and tasks.
---

# Pi Feature Spec Creator

> 📂 **Document placement rule:** Output goes to `docs/features/F{N}-{slug}.md` only.
> Never create files under `docs/plans/`, `docs/process/`, or `docs/reviews/`.

## Overview

Extract a feature from `{spec_path}` (the master spec) and produce a detailed, actionable feature specification with acceptance criteria, implementation tasks, and verification steps. The output is a `docs/features/F{N}-{slug}.md` file.

Every feature in this project originates from `{spec_path}` § Features table. This skill produces the working document that drives implementation.

> **Template note:** This skill is block-agnostic. `{spec_path}` points to whatever upstream spec file is your source of truth. For a new project, update `.pi/block.yaml` with your own paths.

## When to Use

- Starting work on a new feature (F1-F13)
- A feature in `docs/features/INDEX.md` shows 🔴 (no spec yet)
- You are bootstrapping a new project with this boilerplate
- You need to break a feature into implementable tasks
- Requirements seem ambiguous and need expansion before coding
- After a block spec update changes a feature's definition

**When NOT to use:** The feature spec already exists and is current. Go implement it instead.

## Process

### Step 1: Extract from Block Spec

Read `{spec_path}` and extract everything relevant to feature F{N}:

1. **Feature row** from § Features table (one-line description)
2. **Related acceptance criteria** from § Acceptance Criteria (map each to this feature)
3. **Related core concepts** from § Core Concepts
4. **Security implications** from § Sandbox Security Model
5. **Interface contract** from § Interface Contract (inputs/outputs affected)
6. **Dependencies** from § Dependencies (upstream blocks needed)
7. **Out of scope** from § Out of Scope (what this feature explicitly doesn't do)

### Step 2: Identify Ambiguities

Before writing the spec, list anything that is:
- Undefined in `{spec_path}`
- Contradictory between sections
- Dependent on decisions not yet made
- Missing detail needed for implementation

```markdown
## Spec Gaps Found in {spec_path}

| Gap | Section | Impact | Proposed Resolution |
|-----|---------|--------|---------------------|
| {what's missing} | {where} | {blocks what} | {suggestion to fix} |
```

**IMPORTANT:** If gaps are blocking, write a proposal in `docs/proposals/PROP-{NNN}-{slug}.md`.
NEVER directly modify `{spec_path}` — it is upstream-controlled.
Do not invent requirements. Do not work around ambiguity.

### Step 3: Determine Implementation Approach

For each feature, classify (these categories are illustrative — adapt to your block):

| Category | Meaning |
|----------|---------|
| **Upstream-provided** | An upstream component (LLM, external tool) already does this; this block wires it in |
| **Service-layer** | This block's production code handles it (in `{code.module_root}`) |
| **Infrastructure** | Deployment / runtime config (K8s manifests, security profiles, network policy) |
| **Integration** | Requires coordination with another block (cross-block contract) |
| **Configuration** | Existing component does it with the right flags/config; only config work needed |

This determines where implementation work goes (`{code.module_root}`, `deploy/`, `.pi/` config, etc.)

### Step 3.5: Cross-reference existing tests and map ACs (REQUIRED)

> The "phantom test file" anti-pattern: a spec cites `{code.tests_root}security/sandbox_security_test.go`,
> the file doesn't exist, but `{code.tests_root}security/sandbox_test.go` already has 7 real
> tests covering the same ACs. Result: redundant stubs, no real tests added, wrong citations.
> This step prevents that.

**Before writing any `**Files:**` section, run these checks:**

#### 1. Inventory existing tests

```bash
find {code.tests_root} -name "*_test.go" | sort
```

Then scan for near-duplicate filenames (same module, two files — e.g., `translator_test.go`
and `http_translator_test.go`):

```bash
find {code.tests_root} -name "*_test.go" -exec basename {} \; | sort | \
  awk -F_ 'NF>2{print $1"_"$2}' | sort | uniq -d
```

For each near-duplicate pair: read both files, determine which covers which ACs, cite
the correct one. Do not create a third file for the same module.

#### 2. Map each AC to an existing test

For each acceptance criterion:

```bash
# Search for AC keyword in existing tests (single-file grep to avoid line-number confusion)
grep -rn "keyword_from_AC" {code.tests_root} -l
# Then read the matching file to confirm it asserts the specific behaviour
```

- **Test found and covers AC** → cite that file in `**Files:**`
- **Test found but only mentions keyword** → read it; if it doesn't assert the AC, treat as not covered
- **No test found** → create a stub (t.Skip placeholder) and cite it

**Never create a stub for an AC that a real test already covers elsewhere.**

#### 3. Stub discipline

If no existing test covers an AC:
1. Create `{code.tests_root}{area}/{module}_test.go` with a `t.Skip` stub
2. Cite the stub in `**Files:**`
3. Mark the task `⚠️` (needs real test before it can be ✅)

#### 4. Final verification

```bash
# No missing spec-cited test files:
grep -rh 'Files:' docs/features/F*.md | grep -o '{code.tests_root}[^` ,)]*\.go' | sort -u | \
  while read f; do [ -f "$f" ] || echo "MISSING: $f"; done

# No magic hardcoded namespace strings in tests:
grep -rn '"{block.slug}"' {code.tests_root} | \
  grep -v '# \|default\|reason='
```

Both must return empty output before any `**Files:**` section is finalised.

### Step 3.6: Verify ALL cited file paths exist (REQUIRED — W1)

> **REQUIRED:** The phantom-path problem applies to ALL files in `**Files:**`
> sections, not just test files. Dockerfiles, source modules, scripts,
> deploy manifests — any cited path that doesn't exist is a spec lie.

For **every** path listed in any task's `**Files:**` section:

```bash
# For each basename cited (e.g., Dockerfile.sandbox, k8s_manager.py):
find . -name "{basename}" 2>/dev/null
```

Rules:
- **File exists** → cite the exact path returned by `find`. Do not guess.
- **File doesn't exist yet** → cite it as `path/to/file (new — to be created)`.
- **Never cite a path that doesn't exist without the `(new — to be created)` annotation.**

Common failures this prevents:
- `sandbox/Dockerfile.test` — this file no longer exists (ADR-007 superseded; use `deploy/Dockerfile.sandbox`)
- `{code.module_root}internal/runtime/run_handler_test.go` listed twice → second entry should be a different file

Also run the **duplicate basename check** (W2):
```bash
# Scan each task's Files section for repeated basenames
# A basename appearing twice is almost always a copy-paste error
```
If the same basename appears more than once in a single task's `**Files:**`
list, one of the entries is wrong — identify the correct second file before
finalising the spec.

### Step 4: Write the Feature Spec

Save to `docs/features/F{N}-{slug}.md` using this template:

```markdown
# F{N}: {Feature Name}

> Source: `{spec_path}` § Features F{N}
> **Template project:** This skill is part of a generic boilerplate. Replace Pi-specific references with your project's upstream spec source.
> Status: 🟡 Spec written
> Category: {Pi-provided | Service-layer | Infrastructure | Integration | Configuration}

## Definition (from block spec)

{Exact text from {spec_path} Features table}

## Expanded Specification

{Detailed description of what this feature means in the context of this block's architecture. What exactly happens, who does what, what the user / Orchestrator observes.}

## Acceptance Criteria

Mapped from `{spec_path}` § Acceptance Criteria:

- [ ] AC-{N}.1: {Specific testable criterion}
- [ ] AC-{N}.2: {Specific testable criterion}
- [ ] ...

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to {spec_path} acceptance criteria

## Interface Impact

{Which endpoints/events/payloads this feature affects}
{Reference docs/contracts/ for upstream contract details}

## Security Considerations

{What security controls apply to this feature}
{Reference {spec_path} § Sandbox Security Model}

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| {block or feature} | {upstream block / internal feature} | {available / mocked / TBD} |

## Implementation Approach

{How we implement this given the block's architecture (cite relevant ADRs)}
{Reference relevant ADRs — features REFERENCE ADRs, never own them}

**ADR references:** List which existing ADRs apply to this feature.
**ADR gaps:** If this feature needs an architectural decision not yet made,
note it in `### ADR gaps` under § Spec Gaps as a question (not a proposed answer).
Feature specs **surface** ADR needs; the ADR itself is written separately and lives in `docs/decisions/`.

### Surfacing an ADR need (not writing the ADR)

When the spec needs an architectural decision that doesn't yet exist:

1. **Phrase it as a question**, not an answer (e.g., "How does this block reach upstream service X across cluster boundaries?" not "Use nginx-ingress with path-based routing").
2. **Tag it in the `### ADR gaps` table** of § Spec Gaps with the affected features and a proposed ADR title.
3. **If the gap is blocking implementation**, also write a `PROP-{NNN}` so the human can prioritize the decision.
4. **The ADR itself** is authored as a regular `docs/decisions/ADR-{NNN}-{slug}.md` file after the human prioritizes the question. Use the template below.

### ADR template (when an ADR is authored to resolve a surfaced gap)

```markdown
# ADR-{NNN}: {Title}

## Status
Proposed | Accepted | Deprecated | Superseded by ADR-{XXX}

## Date
YYYY-MM-DD

## Scope
- **Topology:** single-cluster only | split-environment only | both (same mechanics) | both (different mechanics)
- **Question type:** protocol | topology | framework | operational
- **Does NOT answer:** {explicit out-of-scope}
- **Depends on / extends:** ADR-{NNN} ({what constraint it inherits})

## Context
Why this decision is needed. What constraints exist (spec, security, team, timeline)?

## Options Considered

### Option A: {Name}
- **Pros:** ...
- **Cons:** ...

### Option B: {Name}
- **Pros:** ...
- **Cons:** ...

## Decision
What we chose and why. Be specific.

## Consequences
- What changes in the codebase/infra
- What becomes easier / harder
- What we explicitly accept as trade-offs

## References
- Relevant spec sections
- Contract digests consulted
- External documentation
```

**Rule: an ADR that answers a protocol question must not also embed a topology answer unless topology is the question being decided.** If you find yourself writing both, split the ADR.

## Tasks

### T{N}.1: {First task title}

**Description:** {What this task accomplishes}

**Acceptance criteria:**
- [ ] {Specific, testable}

**Verification:**
- [ ] `{test command}`

**Files:** `{paths}`
**Size:** {XS|S|M}
**Depends on:** {other tasks or None}

### T{N}.2: {Second task title}
...

## Verification Plan

{How to verify the entire feature works end-to-end}
{Include both automated tests and manual checks}

## Spec Gaps (if any)

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| {what's unclear} | {section} | {proposed wording for docs/proposals/PROP-NNN.md} |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| {architectural question} | {F1, F8, ...} | ADR-{N}: {title} |

Note: ADRs are block-level. Flag the need here; author the ADR file as a separate commit using the ADR template in § Surfacing an ADR need above.
## Out of Scope

{What this feature explicitly does NOT do — from {spec_path} § Out of Scope}
```

### Step 5: Update Feature Index

After creating or revising a feature spec:

- [ ] `docs/features/INDEX.md` Spec Status: set to 🟡 (written/revised)
      A spec that was previously 🟢 (reviewed) and has been changed drops
      back to 🟡 — a revised spec is not a reviewed spec.
- [ ] `docs/features/INDEX.md` Impl Status: if the revision obsoletes
      any previously-✅ tasks, reset them to ⚠️ or 🟡 in the Impl
      Status column with a short note.
- [ ] `docs/features/INDEX.md` dependency diagram: update any
      description that referenced the old mechanism.

After **revising** an existing feature spec, also update:

- [ ] The `## Tasks` table in **this spec file**: retitle/re-describe stale tasks, reset status of stale tasks from ✅ to ⚠️.
- [ ] `docs/features/INDEX.md` `Impl Status` column for this feature.
- [ ] `docs/plan.md` **Active Cursor** if the revision changes what phase is current.
- [ ] For every AC added or modified, ensure there is a corresponding task row in `## Tasks`. If an AC is new and has no task, add one.

**Rule: a spec revision is not complete until the INDEX and task table are
consistent with it. Updating the `.md` spec file alone is never sufficient.**

### Step 6: Flag Spec Gaps for Human Review

If you found ambiguities in Step 2:

```markdown
## 🚨 ACTION REQUIRED: Block Spec Amendments Needed

The following issues in `{spec_path}` need resolution before
implementation can proceed. Proposals written to `docs/proposals/`:

1. {Gap description} — {which section} — see `docs/proposals/PROP-{NNN}.md`
2. ...

**Agent cannot modify `the block spec` directly.** Human must review proposals
and apply accepted amendments to the upstream spec.
```

**After filing any PROP — run the proposal completeness sweep (W9):**

```bash
# Find every spec gap that has a "Proposed Fix" or "Proposed Amendment" entry
# but no corresponding PROP-NNN file reference:
grep -rh "Proposed\|PROP-" docs/features/ | grep -v "PROP-[0-9]"
```

For every gap row that lacks a `PROP-NNN` reference:
1. Check `docs/proposals/` — a proposal may already exist under a different name.
2. If no matching proposal exists → create a stub `docs/proposals/PROP-{NNN}-{slug}.md`
   immediately (even a one-line stub is better than an invisible gap).
3. Update the gap row in the feature spec to cite the new `PROP-{NNN}`.

> **Why:** Filing one proposal while leaving sibling gaps without proposals
> creates invisible technical debt. The completeness sweep makes all open
> gaps visible and human-reviewable at the same time.

## Feature Spec Quality Checklist

Before considering a feature spec complete:

- [ ] Every acceptance criterion traces to `{spec_path}`
- [ ] No invented requirements (everything sourced from spec)
- [ ] Ambiguities flagged, not worked around
- [ ] Implementation approach references ADRs
- [ ] Tasks are sized S or M (break down L)
- [ ] Each task has verification step
- [ ] Security considerations addressed
- [ ] Dependencies identified with status

## Example: Feature groupings for our architecture

Since Pi provides F2-F5, those features get "thin" specs confirming Pi handles them
and documenting how we verify they work inside the sandbox:

- **F2-F5 (Pi-provided):** Spec confirms Pi provides this. Tasks = integration tests proving it works inside the sandbox.
- **F1, F8, F9, F12 (Sandbox infrastructure):** Spec details container config, security, lifecycle. Tasks = infrastructure code.
- **F6, F7, F10, F11 (Orchestration):** Spec details the translation/bridging logic. Tasks = service code.
- **F13 (Integration):** Spec details multi-block coordination. Tasks = integration code + upstream mocks.

## Batch Creation

To create specs for multiple related features at once:

```
/skill:pi-feature-spec

Create feature specs for F1, F8, F9, and F12 (the sandbox cluster).
These are interdependent — the sandbox container (F1) is the foundation
for resource limits (F8), session persistence (F9), and security (F12).
```

The skill will produce 4 files, noting cross-dependencies between them.
