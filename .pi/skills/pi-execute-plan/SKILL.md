---
name: pi-execute-plan
description: Orchestrates execution of an approved plan - picks the next task, authors missing tasks, invokes the right skill, tracks progress, and gates on checkpoints. Use when a feature is ready to build, when resuming after a pause, or when a plan needs authoring before execution.
---

# Pi Execute Plan

> ⛔ **NEVER modify the block spec (`{spec_path}` per `.pi/block.yaml`) directly.**
> If you find spec gaps, write a proposal to `docs/proposals/PROP-{NNN}-{slug}.md`.
>
> 🔍 **Block config:** Per-block values (language, test commands, egress allowlist, upstream specs) live in `.pi/block.yaml`. Read it before starting.

> 📂 **Document placement rule** (enforced — see AGENTS.md § Document Placement Rules):
> `docs/plan.md` is a **navigation-only** file (active cursor + cross-feature dependency graph).
> **Task lists and task status live exclusively in `docs/features/F{N}-*.md`.**
> Never create `docs/plans/*.md` or any file under `docs/process/` or `docs/reviews/`.
> Reports and audits have no permanent home — act on them inline; git history is the audit trail.

## Overview

Drive plan execution methodically: author or repair the plan if needed, pick the next ready task, delegate to the appropriate skill, verify completion, update progress, and stop at checkpoints for review. This skill is the "conductor" - it doesn't implement, it orchestrates.

## When to Use

- A plan has been reviewed (READY verdict from `/skill:pi-review-plan`)
- Resuming work after a pause or session break
- After a checkpoint - deciding what's next
- When asked to "execute the plan" or "do the next task"
- When a plan is missing or incomplete - author it inline, then execute

**When NOT to use:** Making architecture decisions mid-execution (changes WHAT must be built; surface as a feature spec gap or PROP first).

## Process

### Step 1: Load Execution State

```
1. docs/plan.md                    → active cursor (current phase + open tasks)
2. docs/features/F{N}-*.md         → feature spec: tasks, ACs, verification plan
3. docs/features/INDEX.md          → overall progress
4. Last completed task              → where we left off
```

### Step 1.5: Author or repair the task list (inline, in the feature spec)

Before searching for the next task, check whether the feature spec is ready:

| Condition | Action |
|-----------|--------|
| Feature spec has no `## Tasks` section | Add tasks now — see **Task authoring** below |
| Feature spec exists but tasks are missing | Read spec § Acceptance Criteria; author tasks from ACs |
| Tasks exist but sizing is L or XL | Split before executing — see **Sizing guide** below |
| All tasks are ✅ | Run Step 8 (completion) immediately |
| Plan exists and has ready tasks | Proceed to Step 2 |

#### Task authoring template

```markdown
#### Task T{N}.{X}: {Short descriptive title}

**Description:** One paragraph - what this accomplishes and why.

**Acceptance criteria:**
- [ ] {Specific, testable condition}
- [ ] {Specific, testable condition}

**Verification:**
- [ ] Tests pass: `{verify.test}` (per `.pi/block.yaml`)
- [ ] Build succeeds: `{verify.build}`
- [ ] Manual: {what to check}

**Dependencies:** T{N}.{Y} | None
**Files:** `{code.module_root}{path}/`, `{code.tests_root}{path}/`
**Size:** XS | S | M  (L or XL → split before writing)
**Blocked by:** {external dependency, if any}
```

#### Checkpoint template

```markdown
### Checkpoint: After Tasks T{N}.{A}-T{N}.{B}
- [ ] All tests pass
- [ ] Build clean
- [ ] Contract compliance verified against `docs/contracts/{X}.md`
- [ ] Demo-able to team
- [ ] Human review before proceeding
```

Add a checkpoint after every 3-5 tasks.

#### Sizing guide

| Size | Files | Time | Rule |
|------|-------|------|------|
| XS | 1 | < 30 min | Single rule, single function |
| S | 1-2 | 30-60 min | One endpoint, one tool |
| M | 3-5 | 1-3 hr | Multi-component feature (e.g., new endpoint + handler + tests + mock update) |
| L | 5-8 | - | **Must split before executing** |
| XL | 8+ | - | **Must split before executing** |

#### Resetting tasks when a spec changes

When a feature spec is revised, any previously-✅ tasks whose ACs changed must be reset:

| Was the task's... | Reset to |
|------------------|----------|
| Description changed (same goal, new mechanism) | ⚠️ needs re-verify |
| Acceptance criteria changed | ⚠️ needs re-verify |
| Replaced entirely by new tasks | ❌ obsolete - strike out, add new rows |
| Unchanged | ✅ keep as-is |

Also update: checkpoint criteria that described the old mechanism; dependency chains that included the old task IDs.

**Rule: a plan row that still says ✅ but whose spec changed is a lie. Reset it.**

#### Dependency analysis

When authoring a set of tasks, map blocking relationships before committing order:

```
Task A ──→ Task B ──→ Task D
                  ╲
Task C ──────────→ Task E
```

Parallelizable = no shared dependencies. Sequential = dependency chain. Blocked = waiting on external (use mocks).

#### Risk tracking

After authoring or updating tasks, update the risk table in `docs/plan.md` (risks only — no task rows):

| Risk | Status | Mitigation |
|------|--------|------------|
| {description} | New / Active / Mitigated / Closed | {action} |

### Step 2: Determine Next Task

Selection rules (in priority order):

1. **Blocked tasks** → skip, note blocker
2. **Checkpoint reached?** → STOP, run checkpoint verification
3. **Dependencies satisfied?** → task is READY
4. **Multiple ready tasks?** → pick the one that unblocks the most downstream work
5. **All tasks done?** → run final verification, mark feature complete

```markdown
## Execution State: F{N}

| Task | Status | Blocker |
|------|--------|---------|
| T{N}.1 | ✅ Done | - |
| T{N}.2 | ✅ Done | - |
| T{N}.3 | 🔵 NEXT | - |
| T{N}.4 | ⏸️ Waiting | Depends on T{N}.3 |
| T{N}.5 | ⏸️ Waiting | Depends on T{N}.3 |
| T{N}.6 | 🚫 Blocked | ADR-005 not accepted |
```

### Step 3: Delegate to Appropriate Skill

| Task Type | Delegate To |
|-----------|------------|
| Write production code | **Inline** — Step 3 (spec-first gate + TDD) |
| Infrastructure/deploy | `/skill:pi-runtime-sre` |
| Create or extend a mock service | **Inline** — Step 3 (mocks in `{code.mocks_root}` per `.pi/block.yaml`; follow existing patterns; reference `docs/contracts/{block}.md`) |
| Architecture decision needed | **STOP execution**. Surface as a feature spec gap or `PROP-{NNN}`. Cascade via `/skill:pi-apply-prop` after acceptance. |
| Documentation only | Inline - each skill owns its doc surface (developer → API docs + CLAUDE.md; SRE → runbooks + AGENTS.md; architect → ADRs + AGENTS.md decisions) |
| Contract sync needed | `/skill:pi-contract-sync` |
| Plan missing or incomplete | **Inline - Step 1.5** (no skill switch) |

**Provide the delegated skill with:**
- Exact task description from the plan
- Acceptance criteria for this specific task
- Relevant file paths
- Dependencies that were just completed (context)

### Step 3: Implement the Task (inline — spec-first + TDD)

#### 3a. Spec-First Gate (mandatory — must answer before any code edit)

Before writing or modifying ANY code, answer all of these:

| Question | If yes → |
|----------|----------|
| Am I adding/changing a public API (function signature, route shape, CLI flag, env var)? | **STOP**. Update the feature spec first. |
| Am I adding/modifying an acceptance criterion (or *implying* a new one with my code)? | **STOP**. Update the feature spec first. |
| Am I picking a configurable default that ships in the repo (resource limits, timeouts, retries, batch sizes)? | **STOP**. Update the feature spec first. |
| Am I adding/changing an error-handling contract (what gets cleaned up on failure, what gets retried, what gets reported)? | **STOP**. Update the feature spec first. |
| Am I adding/changing a verification workflow (a new make target, smoke script, test category)? | **STOP**. Update the feature spec first. |
| The block spec or feature spec is silent on what I'm about to do? | **STOP**. Either find the right spec home and update it, or write a proposal in `docs/proposals/PROP-{NNN}-{slug}.md`. |

If any answer is yes:
1. Open the relevant `docs/features/F{N}-*.md` (or write a `PROP-{NNN}`).
2. Add the new AC / API description / default / contract.
3. **Save the spec edit before touching code.** Code follows spec, never the other way around.
4. Note the spec change in your commit message.

If all answers are no → the change is purely an implementation detail (refactor, naming, internal helper) and may proceed.

#### 3b. Identify the Acceptance Criterion

Find the exact AC in `docs/features/F{N}-*.md` you're implementing. Each AC traces back to the block spec (`{spec_path}`). If no clear AC exists, you have failed 3a — go back and update the spec.

#### 3c. Write the Failing Test First (RED)

Before creating any test file: `grep -rn "keyword_from_AC" {code.tests_root} -l` to check if an existing test already covers the AC. Only create a new file if none does. Test name pattern: see `.pi/block.yaml` § `test.naming_pattern`.

#### 3d. Implement Minimum Code (GREEN)

Least code to make the test pass. No premature abstraction. No "while I'm here" additions. Follow patterns in `{code.module_root}`.

#### 3e. Verify Cross-Cutting Concerns

- [ ] No skip stubs masking an unimplemented AC (e.g. `t.Skip`, `pytest.skip`, `it.skip` — per language)
- [ ] `{verify.test}` green (per `.pi/block.yaml`)
- [ ] Security: no path escape, no unapproved network egress (check against `.pi/block.yaml` § `security.egress_allowlist`), no privilege escalation
- [ ] Errors caught at boundary; structured error response (not raw stack traces)
- [ ] Structured logs with run-level traceability (run_id / workspace_id / session_id, per block spec)
- [ ] Works against mock services (`{verify.mocks_up}`)

#### 3f. Refactor (only if needed)

Clean up only if readability suffers. Function length < 30 lines (aim for 15). Extract only when a pattern repeats 3+ times.

### Step 4: Verify Task Completion

After the delegated skill finishes, verify:

- [ ] Task's acceptance criteria met (run the verification command)
- [ ] Tests pass: {verify.test command from .pi/block.yaml}
- [ ] No regressions: full test suite still green
- [ ] Cross-cutting: traceability envelope, security, logging

```bash
# Run task-specific verification
{verification command from task spec}

# Run regression check
{verify.test}
```

### Step 5: Update Progress (mandatory — two files)

When a task is verified complete, update **both** files **before**
marking the next task. This is the **anti-drift gate**.

**Before marking any task ✅, verify test coverage inline:**

```
[ ] AC-to-test mapping verified - each AC maps to an existing real test (not a phantom file)
[ ] No skip stubs - test file has no `t.Skip`, `t.Skipf`, or `t.SkipNow` masking an unimplemented AC
[ ] No redundant stubs - stubs only for genuine gaps, not for ACs covered elsewhere
- [ ] {verify.test} — green with no hangs
```

*(These checks are the same checks run in `pi-review-spec` Step 8. Run them inline here;
no external skill switch needed.)*

All checks must pass. A task is not ✅ if any item is open.

#### 5a. Mark the task in the **feature spec** `docs/features/F{N}-{slug}.md`

Update the task's `Status` column in `## Tasks`:

```diff
-| T{N}.3 | {description} | 🔴 |
+| T{N}.3 | {description} | ✅ |
```
#### 5b. Update the feature spec `docs/features/F{N}-{slug}.md`

Change the `Status:` line to reflect the new state:
- If this was the **last remaining task** for the feature → `> Status: 🟢 Implemented`
- If there are **remaining tasks** → `> Status: 🟡 Partially implemented (T{X.Y} remaining)`
- If the feature is **blocked on a proposal** → `> Status: 🟡 Blocked (PROP-{NNN})`

#### 5c. Update `docs/features/INDEX.md`

Update the `Impl Status` column for the feature. Use the legend:
- `🟢 Implemented` - all tasks done
- `🟡 {done}/{total} tasks` - partial
- `🔴 Not started` - no tasks done
- `⚪ Blocked ({reason})` - blocked

> **⚠️ This is the anti-drift gate.** The INDEX and
> feature-spec `Status:` lines must never be stale.
>
> **Rule:** Every time a task row in the **feature spec** changes from ⏳ to ✅,
> the feature spec `Status:` line AND the INDEX `Impl Status` column
> **must** be updated. No exceptions.
>
> **Verification before moving on:**
> ```bash
> grep -q "F{N}" docs/features/INDEX.md
> grep -q "F{N}" docs/features/F{N}-*.md
> # Both must return 0 (found) before proceeding.
> ```

#### 5d. Update dependency status in OTHER feature specs (W3)

When a feature reaches 🟢 Implemented, its feature ID appears in the dependency
tables of other specs - often still marked 🔴 or 🟡 from when it was first cited.

```bash
# Find every spec that still shows this feature as not-done:
grep -rl "F{N}.*🔴\|F{N}.*🟡" docs/features/ 2>/dev/null
# (replace {N} with the feature number, e.g., F12)
```

For each file returned: open it, find the dependency row for F{N}, and update
the status to 🟢. Example:

```diff
- | F12 (Security boundary) | Internal - ... | 🔴 Not started |
+ | F12 (Security boundary) | Internal - ... | 🟢 Implemented |
```

> **Why this step exists:** Step 5 previously only updated the completed
> feature's own files. Every other spec's dependency table was never touched,
> leaving 🔴/🟡 status on features that had been complete for weeks - causing
> confusion in spec reviews and drift audits.

#### 5e. Close resolved spec gaps (W4)

When a task directly resolves a row in the feature spec's `§ Spec Gaps` section
(e.g., "default timeout not specified" → implemented as 30s), mark it closed
in the spec *in the same commit as the task completion*:

```diff
- | Container startup timeout not specified | § Features F1 | Add: "..." |
+ | ~~Container startup timeout not specified~~ | § Features F1 | **Resolved 2026-06-09:** `ready_timeout_s=30s` in-cluster, `60s` local-dev; constructor-overridable. |
```

Alternatively, if the gap row is fully resolved and noise-free, remove it.

> **Rule:** An open-looking gap row with no resolution marker is indistinguishable
> from an active open issue. Every resolved gap must be explicitly closed.
> Do not leave "Proposed Fix: Add..." rows sitting in specs whose implementations
> already shipped the fix.

A task may only be marked `⏳ (blocked on T{X.Y})` when **T{X.Y}
actually exists as another task row in the feature spec** and references the
gap that's blocking the current task. Marking a task `⏳ (deferred -
needs X)` where X is not itself a spec'd task is the same anti-pattern
as writing code without a spec: it hides work in informal notes
instead of making it visible.

If you discover a missing dependency mid-execution:
1. Add the dependency as a new task in the **feature spec** (with
   acceptance criteria and verification, like any other task).
2. Update the feature spec's `## Tasks` table.
3. Update `docs/plan.md` **Active Cursor** section if the new task changes the current phase.
4. Update the original task's `Depends on` to cite it.

See the **anti-drift gate** rule (Step 5b/5c above) for the retro that
motivated this.

#### Reverse drift rule (mandatory)

**When a feature spec changes, the plan must be updated in the same commit.**

This is the mirror of the anti-drift gate above. The anti-drift gate says:
"every ✅ task completion must update the spec." The reverse drift rule says:
"every spec change that touches an acceptance criterion, implementation
approach, or verification step of a task must reset that task's plan row."

| Spec change | Plan action required |
|-------------|---------------------|
| AC rewritten or added to a task | Reset task status ✅ → ⚠️ in the **feature spec task table** |
| Implementation approach changed by ADR | Reset task status ✅ → ⚠️ in the **feature spec task table** |
| New constraint added that changes what the task produces | Reset task status ✅ → ⚠️ in the **feature spec task table** |
| Annotation only (no behaviour change) | No reset required, but add a dated note to the plan row |

**⚠️ = "needs re-verification"** - the implementation may be correct, but
it must be re-checked against the updated spec before the task can return to ✅.

**Rule: a spec edit and its plan reset are a single atomic unit. Committing
a spec change without the corresponding plan reset leaves silent technical
debt - the plan says the work is done when its contract has changed.**

This rule was added after the ADR-008 cascade left T12.2, T1.3, T12.3,
T12.4, and T13.4 marked ✅ while their feature spec ACs were updated to
require split-environment NetworkPolicy shapes that the existing
implementation does not satisfy.

**After any plan row is reset to ⚠️**, check test coverage for the affected tasks
before any implementation work begins: use `pi-review-spec` Step 8 on the
feature spec to inventory gaps and resync the plan. Test gaps must be identified
before code is written, not discovered after.

### Step 6: Checkpoint Gate

When a checkpoint is reached:

```markdown
### 🚦 CHECKPOINT: After Tasks T{N}.1-T{N}.3

Running verification:
- [ ] All tests pass: `{verify.test}` → {result}
- [ ] Build clean: `{verify.build}` → {result}
- [ ] Contract compliance: checked against `docs/contracts/{X}.md` → {result}
- [ ] Security: no new path/network/privilege issues → {result}
- [ ] **Spec drift check - two directions:**
      1. Code → Spec: every code-touching
         task in this block has a corresponding `docs/features/F{N}-*.md`
         edit (or proposal). If `{code.module_root}` changed but no spec did, list which
         task the change came from and confirm the spec genuinely needed no
         update.
      2. Spec → Feature spec: for every spec file touched in this block, verify
         that `docs/features/F{N}-*.md` task descriptions and statuses still match
         the spec. If the spec changed a task's contract, the feature spec task row
         must NOT still say ✅. Reset stale rows to ⚠️ before continuing.
      → {result}

**Checkpoint verdict:** PASS / FAIL

**If FAIL:** Stop. Identify which task introduced the issue.
             Fix before continuing. Do NOT skip ahead.

**If PASS:** Proceed to next task block.
             Request human review if checkpoint specifies it.
```

### Step 7: Handle Blockers

When a task is blocked mid-execution:

| Blocker Type | Action |
|-------------|--------|
| Missing ADR | **STOP**. Document the gap in a feature spec `### ADR gaps` section or write `PROP-{NNN}`. Resume after acceptance + cascade. |
| Upstream dependency | Check if mock exists → if not, create mock first → resume |
| **Spec gap or implicit decision** - ANY of the following: a public API shape, a new/modified acceptance criterion, a configurable default, an error-handling contract, a verification workflow, or anything the spec is silent on | **STOP** → update `docs/features/F{N}-*.md` (or write `docs/proposals/PROP-{NNN}-{slug}.md`) → spec change committed BEFORE any code edit → then resume |
| **Proposal accepted while task was blocked** — a pending PROP has been approved and `the block spec` updated | **Run `/skill:pi-apply-prop` first** to cascade the change into ADRs, feature specs, and plan cursor — THEN resume task. Starting the task before the cascade is complete reintroduces stale-assumption drift. |
| Test failure in other feature | Investigate → fix → re-verify → resume |
| Human review required | Stop → present state → wait for human |

> **Defensive test:** "If the spec author reviewed my next commit, would
> they say 'I'd have written that into the spec'? If yes → update the
> spec first."

### Step 8: Completion

When all tasks for a feature are done:

1. **Run full verification plan** (from feature spec § Verification Plan)
2. **Run inline review checklist (Step 8a)** before marking the feature complete
3. **Final consistency check** — the anti-drift gate:
   - `docs/features/F{N}-*.md` `Status:` line → `🟢 Implemented`
   - `docs/features/F{N}-*.md` `## Tasks` table → all rows ✅
   - `docs/features/INDEX.md` `Impl Status` for F{N} → `🟢 Implemented`
   - `docs/plan.md` **Active Cursor** → updated to reflect feature complete
   - If any of these is stale → fix it before presenting the completion report.
4. **Present completion report:**

```markdown
## Feature F{N} Execution Complete

**Feature:** {name}
**Tasks completed:** {X}/{Y}
**Duration:** {start date} → {end date}
**Tests:** {count} passing

### Acceptance Criteria Status

| AC | Status | Evidence |
|----|--------|----------|
| AC-{N}.1 | ✅ | Test: `test_name` |
| AC-{N}.2 | ✅ | Test: `test_name` |

### Deviations from Plan

- {any changes made during execution}

### Discoveries

- {anything learned that affects other features}

### Ready for Review

Run the inline pre-completion review (Step 8a) before presenting the report.
```

### Step 8a: Pre-Completion Code Review (inline)

Run this review **before** marking the feature 🟢 Implemented and **before** presenting the completion report. Every check is mandatory.

#### Security pass (blocking — any violation = stop, fix, re-run)

| Check | Look for |
|-------|----------|
| Path traversal | Any path operation without `validatePath()`-style guard (resolve + startsWith mountPath) |
| Mount escape | `../`, symlink following, or path joining without validation |
| Network egress | Any outbound call from sandboxed code paths that is NOT in `.pi/block.yaml` § `security.egress_allowlist` |
| Credential handling | Secrets written to disk, logged, or stored in memory beyond immediate use |
| Resource bypass | Code that disables or circumvents cgroup limits / resource requests |
| Shell injection | User-controlled strings interpolated into shell commands without sanitization |
| Privilege escalation | setuid, added capabilities, or root operations |
| Information leak | Internal paths, secrets, host info in API responses or events |

#### Contract compliance pass

| Check | Verify against |
|-------|---------------|
| API request/response shapes | `docs/contracts/{block}.md` digest |
| Event types & payloads | Block spec § Outputs; if the block relays a raw agent stream (e.g. SSE), no envelope schema applies to that stream — only to service-level lifecycle events listed in `.pi/block.yaml` § `lifecycle_events` |
| Completion report | All required fields per block spec § Outputs (`status`, `usage`, `error?`, plus any block-specific fields) |
| Error format | Structured error responses — never raw stack traces |
| Streaming destination | If the block produces a streaming output (events / SSE / WebSocket), it goes only to the consumers named in the block spec § Outputs |

#### Test coverage pass

| Check | Standard |
|-------|----------|
| Every new branch / function / error path has a test | No exceptions |
| Test names follow `Test{Module}_{Behavior}_{Outcome}` | Descriptive, specific |
| Tests are independent | No order dependency, no shared mutable state |
| Security has negative tests | "Cannot read outside mount", "Cannot reach unapproved network" |
| Mocks return realistic responses | Match contract digests |

#### Code quality pass

| Check | Standard |
|-------|----------|
| Function length | < 30 lines (15 preferred); > 50 requires justification |
| Error handling | Caught at boundaries; never swallowed silently |
| Logging | Structured (slog) with `run_id` / `workspace_id` / `session_id` |
| Naming | Specific. Avoid bare `data`, `result`, `handle` |
| Dependencies | One direction. No circular imports |
| Statefulness | Service is stateless unless the block spec declares otherwise — durable state lives in upstream blocks (e.g. Session Manager, Workspaces, Secret Manager) named in the spec |
| Hardcoded values | No secrets, URLs, or credentials in code |

#### Verdict

```markdown
### Pre-Completion Review: F{N}

**🔴 Blockers (must fix):** {none | list}
**🟡 Warnings (should fix):** {none | list}
**🟢 Strengths:** {what's well done — specific}
**Coverage gaps:** {missing test scenarios | none}
**Verdict:** APPROVE | REQUEST_CHANGES | BLOCK
```

If BLOCK → fix the blocker, re-run 8a. Only proceed to Step 8 final report when verdict is APPROVE.

## Execution Principles

1. **One task at a time.** Never start T{N}.4 before T{N}.3 is verified.
2. **Fail fast.** If verification fails, stop immediately. Don't accumulate debt.
3. **Document everything.** Deviations, discoveries, assumptions - all recorded.
4. **Respect checkpoints.** They exist for a reason. Never skip them.
5. **Stay in lane.** This skill orchestrates. It doesn't implement, design, or review.
