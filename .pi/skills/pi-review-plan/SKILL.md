---
name: pi-review-plan
description: Reviews implementation plans and task breakdowns for feasibility, completeness, correct ordering, and sizing before execution begins. Use after a plan or task list is created and before any implementation work starts.
---

# Pi Review Plan

> ⛔ **NEVER modify `{spec_path}` directly.**
> If you find spec gaps, write a proposal to `docs/proposals/PROP-{NNN}-{slug}.md`.

> 📂 **Document placement rule:** Review verdict is inline (in the response) only.
> Task status updates go to the **feature spec** `## Tasks` table only. Never write to `docs/reviews/` or `docs/process/`.

## Overview

Validate that an implementation plan is executable, correctly ordered, properly sized, and covers all acceptance criteria before committing resources to execution. This is the quality gate between "plan written" and "development begins."

## When to Use

- After a feature spec's `## Tasks` section has been authored or updated
- Before starting execution of a feature's task list
- When a plan has been sitting idle and needs freshness check
- After ADR changes that might invalidate task ordering
- When a checkpoint review reveals plan drift

**When NOT to use:** Executing tasks (use `pi-execute-plan`), reviewing code (use `/skill:pi-execute-plan` Step 8a inline review).

## Process

### Step 1: Load Context + Drift Audit

```
1. docs/features/F{N}-*.md        → feature spec: tasks, ACs, verification plan
2. docs/plan.md                   → active cursor (current phase + dependency graph)
3. docs/features/INDEX.md         → feature index (consistency check!)
4. docs/decisions/ADR-*.md        → constraints on ordering/approach
5. docs/contracts/*.md            → interface constraints
6. {spec_path}  → acceptance criteria (all covered?)
```

#### Step 1.5: Anti-Drift Consistency Check (mandatory)

Before validating the task list, verify that the INDEX and feature specs are in sync.

```
For each feature F{N} being reviewed:
  1. Count tasks in docs/features/F{N}-*.md: total, done (✅), remaining (⏳/🚫/⚠️)
  2. Read docs/features/F{N}-*.md `Status:` line
  3. Read docs/features/INDEX.md `Impl Status` column
  4. Check: do both agree?
     - If feature spec says all ✅ but INDEX says "Not started" → STALE
     - If any mismatch → flag it and fix it before proceeding
```

**If drift is found:**
1. Report the mismatch
2. Fix the INDEX and feature-spec `Status:` line to match the task table
3. Continue the review - do NOT skip the plan review because of stale docs

### Step 2: Coverage Check - AC to Task Mapping

Every acceptance criterion in the feature spec MUST map to at least one task:

```markdown
| AC | Mapped to Task(s) | Coverage |
|----|-------------------|----------|
| AC-{N}.1 | T{N}.2, T{N}.3 | ✅ Full |
| AC-{N}.2 | - | 🔴 MISSING |
| AC-{N}.3 | T{N}.5 | ⚠️ Partial (negative case not covered) |
```

**If any AC has no task → plan is incomplete**

### Step 3: Dependency & Ordering Check

For each task, verify:

- [ ] Dependencies are explicitly stated
- [ ] No circular dependencies
- [ ] Blocked tasks have a mitigation (mock? alternative path?)
- [ ] Sequential ordering is correct (can't test what isn't built)
- [ ] Parallelizable tasks are identified

```
Expected: A → B → C (where B needs A's output)
Found:    B listed before A → ❌ ORDERING ERROR
```

### Step 4: Sizing & Decomposition Check

| Check | Pass Criteria |
|-------|---------------|
| No task sized L or XL | Must be broken down further |
| Each task ≤ 3 files touched | Bigger = probably needs splitting |
| Each task has ONE clear outcome | Not "implement X and also Y" |
| Each task has verification step | "How do I know it's done?" |
| Time estimates are realistic | S = 30-60min, M = 1-3hr |

### Step 5: Feasibility Check

| Check | What to verify |
|-------|----------------|
| Mock availability | Tasks against upstream blocks - are mocks ready? |
| ADR prerequisites | Does the plan assume decisions not yet made? |
| Tool/infra readiness | Does the plan assume infrastructure not yet built? |
| Skill gaps | Does the plan require expertise not documented? |
| Risk coverage | Are high-risk tasks early (fail fast)? |

### Step 6: Checkpoint Check

- [ ] Checkpoints placed after every 3-5 tasks
- [ ] Each checkpoint has verification criteria
- [ ] At least one checkpoint requires human review
- [ ] "Demo-able" milestones identified for team visibility

### Step 7: Produce Review Verdict

```markdown
## Plan Review: {Feature or Phase Name}

**Reviewer:** Agent
**Date:** YYYY-MM-DD
**Plan scope:** {tasks N-M / feature F{N} / phase P{N}}

### Coverage

| Acceptance Criteria | Tasks | Status |
|--------------------|-------|--------|
| AC-{N}.1 | T{N}.2 | ✅ |
| AC-{N}.2 | - | 🔴 No coverage |

**Coverage:** {X}/{Y} ACs mapped ({Z}%)

### Ordering & Dependencies

- [ ] ✅ No circular dependencies
- [ ] ✅ Sequential ordering correct
- [ ] ⚠️ {issue description}

### Sizing

- [ ] ✅ All tasks S or M
- [ ] 🔴 Task T{N}.4 is L - needs decomposition

### Feasibility

- [ ] ✅ Mocks available for all upstream deps
- [ ] ⚠️ ADR-005 not yet accepted - blocks T{N}.7

### Checkpoints

- [ ] ✅ Checkpoints every 3-5 tasks
- [ ] ⚠️ No human review checkpoint before risky section

### Verdict: READY | REVISE | BLOCKED

**Summary:** {one paragraph}

**Must fix before execution:**
1. {specific issue}
2. ...

**Suggested improvements:**
1. {optimization}
2. ...
```

### Step 8: Update Status

If READY:
- Feature status can advance to 🔵 (in progress) when execution begins
- Plan is locked - changes require re-review

If REVISE:
- Fix the plan inline using `pi-execute-plan` Step 1.5 (task authoring, splitting, reset table)
- Re-review after fixes

## Anti-Patterns to Flag

| Anti-Pattern | Why It's Bad | Fix |
|--------------|-------------|-----|
| "Implement feature X" (one giant task) | Untestable, untrackable | Break into S/M tasks |
| Tasks with no verification | Can't tell when done | Add test command |
| All tasks sequential | No parallelism, slow | Find independent work |
| No checkpoints | Drift goes unnoticed | Add after every 3-5 tasks |
| Assuming upstream ready | Gets blocked | Use mocks, note risk |
| Tasks that "also" do something else | Scope creep | One outcome per task |
