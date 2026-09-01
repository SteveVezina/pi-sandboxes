# AGENTS.md — PI Agent Sandbox Runtime

> **Per-project config:** `.pi/block.yaml` defines the slug, spec path, code layout, verify commands, security egress allowlist, and lifecycle events for this project. All `pi-*` skills are project-agnostic and read from that file.

## The One Rule

**`SPEC.md` is the master source of truth.**
If code and spec disagree → spec wins. If spec is ambiguous → propose a spec amendment, don't work around it.

### ⛔ Spec Protection Policy

**NEVER directly modify `SPEC.md` — UNLESS applying an accepted PROP.**

This file is the upstream-controlled contract. When you find gaps, ambiguities, or errors:

1. **Document the gap** in `docs/proposals/PROP-{NNN}-{slug}.md`
2. **Flag it for human review** — human says "accept PROP-{NNN}" to approve
3. **When human accepts a PROP**, the agent applies it automatically:
   - Update `SPEC.md` with the proposed changes
   - Cascade changes into affected `docs/features/F{N}-*.md` specs
   - Update `docs/features/INDEX.md` and `docs/proposals/INDEX.md`
   - Trigger downstream effects (contracts, tests, generated code)
   - Commit with descriptive message
4. **Continue with the best interpretation** if non-blocking (document your assumption)
5. **Stop and wait** if the ambiguity is blocking

Proposals follow the template in `docs/proposals/TEMPLATE.md`.

> **Automation:** The `pi-apply-prop` skill handles PROP acceptance. When a human says
> "accept PROP-{NNN}", the agent reads the PROP, applies block spec changes, cascades
> into feature specs, updates INDEX files, and commits — all automatically.

### ⛔ PROP Queue Guard

**NEVER create a new PROP while there are unapproved proposals in `docs/proposals/`.**

Before writing `PROP-{NNN}`:
1. Check `docs/proposals/INDEX.md` for any proposals with status `🟡 Proposed` or `🟡 Partially resolved`
2. If unapproved proposals exist → **STOP**. Flag the human: "There are {N} unapproved PROP(s) pending your review. Please accept/deny them before I write a new one."
3. Only proceed to write a new PROP after all existing ones are `✅ Accepted`, `❌ Withdrawn`, or `🔴 Rejected`

This prevents proposal pile-up and ensures each gap gets proper human attention before introducing new ones.

**Exception:** If the new gap is a **blocking implementation blocker** (code cannot proceed), create the PROP but note in the proposal: "⚠️ BLOCKING — pending human approval. Existing unapproved PROPs: {list}."

### ⛔ PROP Numbering Guard

**NEVER create a PROP with a number that already exists or creates a gap.**

Before writing `PROP-{NNN}`:
1. List existing PROPs: `ls docs/proposals/PROP-*.md | xargs -I{} basename {} | sort`
2. Extract the highest number: e.g., `PROP-004-*` → next is `005`
3. **NEVER** skip numbers
4. **NEVER** reuse a number that already exists
5. If a number was withdrawn (file deleted), reuse it only if the withdrawal was recent (< 7 days) and document the reuse

**INDEX sync guard:**
- After writing a new PROP, update `docs/proposals/INDEX.md`
- After withdrawing a PROP, remove its entry from the INDEX
- After accepting a PROP, update its status in the INDEX
- **NEVER** leave the INDEX out of sync with actual files

## Context Loading Protocol

Before any implementation work:

1. Read `docs/design-principles.md` — platform invariants every decision must respect
2. Read `SPEC.md` § relevant feature
3. Read `docs/features/F{N}-*.md` for the feature spec + plan
4. Read `docs/contracts/*.md` for upstream interfaces you touch
5. Read `ARCHITECTURE.md` for the architectural overview
6. **Cross-check upstream implementations** in the paths listed in `.pi/block.yaml` § `upstream` — real OpenAPI specs and DAT.md files are the source of truth for endpoint shapes
7. If anything conflicts with a design principle → **stop and raise it**
8. If anything is unclear → **stop and propose a spec amendment**

## Spec-Driven Development

```
docs/design-principles.md (INVARIANTS — non-negotiable constraints)
    │
    ├── must be respected by ALL decisions and specs
    │       └── when changed: recalibrate all Dependents listed in the principle
    │
SPEC.md (WHAT + WHY — master source of truth)
    │
    ├── defines Features F1–F27
    │       │
    │       └── docs/features/F{N}-{slug}.md
    │               ├── Feature spec (extracted from block spec)
    │               ├── Acceptance criteria (traceable to block spec)
    │               ├── Task plan (ordered, testable)
    │               └── References ADRs (never owns them)
    │
    ├── implies architectural questions (HOW)
    │       │
    │       └── docs/decisions/ADR-{N}-{slug}.md
    │               ├── Block-level (cross-cutting, affects multiple features)
    │               ├── Authored using template in /skill:pi-feature-spec
    │               └── Feature specs REFERENCE these, never own them
    │
    ├── defines Interface Contract (inputs/outputs)
    │       └── validated against: docs/contracts/*.md
    │
    ├── defines Security Model
    │       └── verified by: tests/security/
    │
    └── defines Acceptance Criteria (15+ items)
            └── each traced to a feature + test
```

### ADR Governance

- **ADRs are block-level.** They answer HOW the block spec's requirements are met.
- **Triggered by:** the block spec (ambiguity, cross-cutting concern) or feature spec gaps.
- **Feature specs *reference* ADRs** (in "Implementation Approach") but never own/create them.
- **Feature specs *surface* ADR needs** (in "Spec Gaps") and the ADR is authored using the template in `/skill:pi-feature-spec` § Surfacing an ADR need.
- **One ADR may serve many features.**

## Core Design Principle

**Fix the root cause, never work around it.**

If an agent, subagent, skill, tool, or spec introduces issues:
1. **Stop** — don't add compensating code
2. **Investigate** — find the actual problem
3. **Fix at the source** — update the skill, agent, or tool (for spec issues → propose amendment)
4. **Verify** — confirm it works without workarounds

## Project Structure

```
├── cmd/                      # Entry points
│   ├── pi/                   # CLI binary (Cobra-based)
│   ├── pi-sandboxd/          # Daemon binary
│   ├── pi-agentd/            # Agent-side daemon (MicroVM guest)
│   ├── pi-init/              # MicroVM guest init
│   └── pi-vmm-manager/       # MicroVM lifecycle manager
├── pkg/                      # Core library packages
│   ├── api/                  # REST / WebSocket API handlers
│   ├── daemon/               # Daemon lifecycle & management
│   ├── exec/                 # Command execution engine
│   ├── runtime/              # Runtime backends (fast, compat, secure, microvm)
│   ├── sandbox/              # Sandbox lifecycle management
│   ├── workspace/            # File system operations
│   ├── template/             # Template system
│   ├── policy/               # Policy enforcement
│   ├── snapshot/             # Snapshot & rollback
│   ├── cache/                # Dependency cache management
│   ├── secrets/              # Secrets management
│   ├── network/              # Network policy
│   ├── logs/                 # Log collection & history
│   ├── artifacts/            # Artifact export
│   ├── git/                  # Git operations
│   ├── context/              # Execution context
│   ├── system/               # System commands
│   ├── terminal/             # Terminal emulation
│   ├── mcp/                  # MCP protocol support
│   ├── remote/               # Remote daemon support
│   ├── gui/                  # GUI workbench
│   ├── types/                # Shared types
│   ├── python/               # Python SDK
│   └── typescript/           # TypeScript SDK
├── tests/                    # Integration & unit tests
├── docs/                     # Specs, features, contracts, decisions
├── examples/                 # Usage examples
├── mocks/                    # Mock services
├── specs/                    # Block spec & design docs
├── go.mod / go.sum           # Go module definition
├── Makefile                  # Build & test automation
├── Dockerfile                # Container build
└── SPEC.md                   # Master spec (READ-ONLY)
```

## Document Placement Rules (mandatory)

Every document in `docs/` belongs to exactly one category. Before creating or moving any file, check this table:

| Document type | Correct location | Rule |
|---------------|-----------------|------|
| Feature contract (what + ACs + tasks + status) | `docs/features/F{N}-*.md` | One file per feature |
| Feature index / dashboard | `docs/features/INDEX.md` | Single file |
| Architecture decision (permanent HOW) | `docs/decisions/ADR-{N}-{slug}.md` | Immutable once accepted |
| Spec change needing human approval | `docs/proposals/PROP-{NNN}-{slug}.md` | Required before any code |
| Upstream API digest | `docs/contracts/{service}.md` | One per upstream service |
| Platform invariants | `docs/design-principles.md` | Single file |
| Developer setup | `docs/dev-setup.md` | Single file |

**Nothing else belongs in `docs/`.** In particular:
- ❌ No `docs/plans/` subdirectory (migration plans, phased plans, concept maps)
- ❌ No `docs/process/` subdirectory (audits, drift reports, retrospectives)
- ❌ No `docs/reviews/` subdirectory (review outputs)
- ❌ No one-time reports or milestone markers

> Git history is the audit trail. Point-in-time reports have no permanent home — act on them and discard.

## Spec-First Discipline (mandatory)

The following categories of change MUST be specified before they are coded.
"Specified" means a `docs/features/F{N}-*.md` edit (or `docs/proposals/PROP-{NNN}-{slug}.md`
if the block spec is the right home and you can't edit it). Code follows
spec, never the other way around.

| Category | Examples | Lives in |
|----------|----------|----------|
| **Public API surface** | function signatures on classes used across modules, HTTP route shapes, CLI flags, env vars consumed by the service | feature spec § Interface Impact + task acceptance criteria — **also update the matching `website/docs/` page in the same change** (see "Keep user-facing docs current") |
| **Acceptance criteria** | any new condition the code must satisfy or any change to an existing one | feature spec § Acceptance Criteria |
| **Configurable defaults that ship in the repo** | timeouts, retry counts, batch sizes, TTLs, max iterations | feature spec § Implementation Approach (or a referenced ADR) |
| **Error-handling contracts** | what gets cleaned up on failure, what gets retried, how partial state is reported, exit-code interpretations | feature spec § Acceptance Criteria + relevant task |
| **Verification workflows** | new make targets, smoke scripts, test categories, env vars that gate tests | feature spec § Verification Plan + relevant task |
| **Cross-cutting envelope/event shapes** | adding fields to events, changing event schema | relevant feature spec |
| **Design principle changes** | any change to a DP-NNN entry in `docs/design-principles.md` | update the principle first, then recalibrate all listed Dependents |

**Defensive test before any code edit:** *"If the spec author reviewed my
next commit tomorrow, would they say 'I'd have written that into the
spec'? If yes, the spec needs the change first."*

## Skills (`.pi/skills/`)

### Lifecycle: Define → Review → Execute

| Stage | Skill | Purpose |
|-------|-------|---------|
| **Spec** | `/skill:pi-feature-spec` | Author / revise a feature spec; surface ADR needs (with template) |
| **Spec** | `/skill:pi-review-spec` | Validate spec completeness, traceability, testability |
| **Plan** | `/skill:pi-review-plan` | Validate task list coverage, ordering, feasibility |
| **Execute** | `/skill:pi-execute-plan` | Pick task → spec-first gate → TDD → pre-completion review (Step 8a) |
| **Execute** | `/skill:pi-runtime-sre` | Infrastructure / deploy / security work |

### Proposal Lifecycle (block spec changes only)

| Stage | Skill | Purpose |
|-------|-------|---------|
| **Review** | `/skill:pi-review-prop` | Validate a proposed PROP before human acceptance |
| **Apply** | `/skill:pi-apply-prop` | Apply an accepted PROP (block spec + ADR + feature spec cascade) |

### Supporting Skills

| Skill | Purpose |
|-------|---------|
| `/skill:pi-contract-sync` | Refresh contract digests from upstream submodule |

### Development Workflow (strict ordering)

```
block-spec.md (READ-ONLY; modified only via /skill:pi-apply-prop)
        │
        ▼
DEFINE  pi-feature-spec  ──▶  pi-review-spec  (🟢 APPROVED)
                                       │
                                       ▼
PLAN                            pi-review-plan   (🟢 READY)
                                       │
                                       ▼
EXECUTE                         pi-execute-plan
                                (spec-first → TDD → Step 8a review)
                                       │
                                       ▼
                                feature ✅ done

Block-spec amendment path (parallel):
    PROP written  ──▶  pi-review-prop  ──▶  human accepts  ──▶  pi-apply-prop (cascade)

At ANY point if spec is ambiguous:
    → Write docs/proposals/PROP-{NNN}-{slug}.md
    → Flag for human review
```

## Build & Test

```bash
# Build all binaries
make build

# Build individual binaries
make build-sandboxd   # pi-sandboxd
make build-pi         # pi

# Install to GOPATH
make install

# Run tests
make test             # with race detector
make test-short       # without race detector
make test-coverage    # with coverage report

# Lint & format
make lint
make format

# Mock services
make mock-up          # Start mock services
make mock-down        # Stop mock services
```

### Binary Reference

| Binary | Path | Purpose |
|--------|------|---------|
| `pi-box` | `cmd/pi-box/main.go` | CLI entry point |
| `pi-sandboxd` | `cmd/pi-sandboxd/main.go` | Sandbox daemon |
| `pi-agentd` | `cmd/pi-agentd/` | Agent-side daemon (MicroVM guest) |
| `pi-init` | `cmd/pi-init/` | MicroVM guest init |
| `pi-vmm-manager` | `cmd/pi-vmm-manager/` | MicroVM lifecycle manager |

## Mock Services

```bash
docker compose -f mocks/docker-compose.mocks.yml up -d
# Orchestrator:9001, Gateway:9002, Sandbox Manager:9003, Secret Manager:9004
```

## Security Constraints (non-negotiable)

- No filesystem escape beyond mount_path
- No network access except: (a) Pi Runtime service endpoint, (b) Pi Gateway, (c) approved Git remotes
- Seccomp profile restricts syscalls
- cgroups enforce CPU/memory/disk limits
- Run as unprivileged user inside container
- Git credentials injected just-in-time, never persisted
- Host home directory not mounted by default
- Docker socket not mounted by default
- Cloud metadata credentials (169.254.169.254) blocked by default

## Cross-Cutting Requirements

Run-level traceability (`workspace_id`, `actor_id`, `run_id`, `sandbox_id`) is carried in the dispatch payload and completion report.

### Lifecycle Events

Pi Runtime emits these **service-level lifecycle events** to the Orchestrator.
They are distinct from the raw SSE agent stream (which is forwarded verbatim).

| Lifecycle Event | Source |
|-----------------|--------|
| `pi.sandbox.created` | Service emits on Pod creation |
| `pi.run.started` | Pi `agent_start` event |
| `pi.run.completed` | Pi `agent_end` event |
| `pi.sandbox.destroyed` | Service emits on Pod destruction (TTL or explicit) |
| `pi.artifact.delivered` | After successful Workspaces POST /output |

## Keep user-facing docs current

`website/` (Docusaurus, deployed separately to Vercel) is the user- and
API-facing documentation — separate from the spec-driven docs under
`docs/` (features, ADRs, proposals, plan) and from this file.

Whenever a task adds, removes, or meaningfully changes something a user of
the CLI, daemon API, or SDK would need to know — a new/changed `pi-box`
command or flag, a new/changed `/v1/*` route or request/response shape, a
new template or config key, a new `pi-sandboxd` flag, a changed
runtime-mode behavior or default, a new lifecycle event, or a changed
install/start step — update the matching page under `website/docs/` **in
the same change**, not a follow-up pass. Full guide:
`website/docs/contributing-docs.md`.

Before the change is done, run `pnpm run docs:build` from the repo root
(exposed as `verify.docs` in `.pi/block.yaml`; it runs the `website/`
Docusaurus build through turbo). `onBrokenLinks: 'throw'` turns a broken
internal link into a build failure. Purely internal changes
(refactors, test-only, deploy plumbing) need no doc update — use judgment.

Rule of thumb: **any "yes" on the Spec-First Discipline "Public API
surface" row also means a `website/docs/` page is due in the same
change.**

## Quality Gates

Before any feature is complete:
- [ ] Traces to acceptance criterion in `SPEC.md`
- [ ] Feature spec exists in `docs/features/`
- [ ] Tests pass (against mock services)
- [ ] Security constraints verified
- [ ] Feature status updated in `docs/features/INDEX.md`
- [ ] User-facing docs updated if the change touches a CLI/API/SDK/template/config surface, and `pnpm run docs:build` passes
