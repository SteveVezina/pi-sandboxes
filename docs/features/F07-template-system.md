# F07: Template System

> Source: `SPEC.md` §6 Features F5
> Status: 🟢 Implemented
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F5 | Template System | Declarative template definition, build, and management (base, node, python, go, rust, node-python, polyglot) | M1 |

## Expanded Specification

Templates define the language/toolchain environment for sandbox sessions. They are declarative YAML files that specify:
- Base OS image or rootfs
- Installed tools and versions
- Cache mount points
- Network policy defaults
- Resource defaults

Initial templates:
- **base** — Minimal: bash/dash, busybox/coreutils, git, curl, ca-certificates, openssh-client, tar, gzip, zstd, unzip, jq, ripgrep
- **node** — Node.js LTS + npm + pnpm + corepack
- **python** — Python 3.x + uv + pip + venv support
- **go** — Go stable toolchain + GOMODCACHE + GOCACHE configured
- **rust** — rustc + cargo + rustup optional + sccache optional
- **node-python** — Node.js 22 + pnpm + Python 3.13 + uv
- **polyglot** — All of the above

Template file example (node-python):
```yaml
name: node-python
runtime: auto
base: debian-slim
tools:
  - git
  - curl
  - ripgrep
  - jq
  - node:22
  - pnpm
  - python:3.13
  - uv
mounts:
  workspace: /workspace
  artifacts: /artifacts
  caches:
    npm: /cache/npm
    pnpm: /cache/pnpm
    uv: /cache/uv
    pip: /cache/pip
network:
  default: restricted
```

Templates are stored under `~/.pi/templates/`. The template manager handles:
- Listing available templates
- Inspecting template details
- Building template images (for compat backend)
- Updating templates from remote sources (future)
- Pruning unused templates

CLI commands:
```bash
pi template list
pi template inspect <name>
pi template build <name>
pi template update <name>
pi template prune
```

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-5.1: `base`, `node`, `python`, `go`, `rust`, `node-python`, `polyglot` templates defined
- [ ] AC-5.2: `pi template list` shows available templates
- [ ] AC-5.3: `pi template inspect <name>` shows template details
- [ ] AC-5.4: Templates configure correct toolchains and cache mounts

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/template/` | Template manager |
| `~/.pi/templates/` | Template storage |
| F3: Fast Backend | Templates define what's mounted in fast mode |
| F4: Compat Backend | Templates define Docker images for compat mode |

## Security Considerations

- Templates are trusted local files (no remote template loading by default)
- Template tool versions pinned (no `latest` tags)
- No secrets in templates
- Cache mounts scoped per template (not world-readable)

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F3: Fast Backend | Internal feature | ⚠️ Templates define mounts for fast backend |
| F4: Compat Backend | Internal feature | ⚠️ Templates define images for compat backend |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go template manager in `pkg/template/` |
| **Configuration** | YAML template files under `~/.pi/templates/` |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T7.1: Template metadata store

**Description:** Implement template metadata store. YAML files under `~/.pi/templates/<name>/template.yaml`. CRUD for template definitions.

**Acceptance criteria:**
- [ ] Templates stored as YAML under `~/.pi/templates/<name>/template.yaml`
- [ ] Template schema: name, runtime, base, tools, mounts, network
- [ ] `List()` returns all template names
- [ ] `Get(name)` returns template details
- [ ] `Create(name, yaml)` writes template file
- [ ] `Delete(name)` removes template directory

**Verification:**
- [ ] `go build ./pkg/template/...`
- [ ] Unit tests for template YAML parsing

**Files:** `pkg/template/store.go`, `pkg/template/template.go`
**Size:** S
**Depends on:** None

### T7.2: Initial templates

**Description:** Ship base, node, python, go, rust, node-python, polyglot templates as default templates.

**Acceptance criteria:**
- [ ] All 7 templates defined with correct toolchains
- [ ] base template has: bash, git, curl, ripgrep, jq, etc.
- [ ] node template has: node:22, npm, pnpm, corepack
- [ ] python template has: python:3.13, uv, pip, venv
- [ ] go template has: go stable, GOMODCACHE, GOCACHE
- [ ] rust template has: rustc, cargo
- [ ] node-python has: node:22, pnpm, python:3.13, uv
- [ ] polyglot has all tools from all templates

**Verification:**
- [ ] `go build ./pkg/template/...`
- [ ] `pi template list` shows all 7 templates
- [ ] `pi template inspect node-python` shows correct details

**Files:** `~/.pi/templates/base/template.yaml`, `~/.pi/templates/node/template.yaml`, `~/.pi/templates/python/template.yaml`, `~/.pi/templates/go/template.yaml`, `~/.pi/templates/rust/template.yaml`, `~/.pi/templates/node-python/template.yaml`, `~/.pi/templates/polyglot/template.yaml`
**Size:** S
**Depends on:** T7.1 (metadata store)

### T7.3: Template CLI commands

**Description:** Implement `template list/inspect/build/update/prune` CLI commands.

**Acceptance criteria:**
- [ ] `pi template list` shows available templates
- [ ] `pi template inspect <name>` shows template details
- [ ] `pi template build <name>` builds template image (stub for compat backend)
- [ ] `pi template prune` removes unused templates
- [ ] `--json` flag produces valid JSON output

**Verification:**
- [ ] `go build ./cmd/pi/...`
- [ ] Integration test: template commands work

**Files:** `cmd/pi/template/list.go`, `cmd/pi/template/inspect.go`, `cmd/pi/template/build.go`, `cmd/pi/template/update.go`, `cmd/pi/template/prune.go`
**Size:** S
**Depends on:** T7.2 (initial templates)

## Verification Plan

- [ ] `go build ./pkg/template/...` succeeds
- [ ] All 7 templates defined and loadable
- [ ] `pi template list` shows all templates
- [ ] `pi template inspect <name>` shows correct details
- [ ] Template YAML parsing handles valid and invalid input

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Template build process for compat backend not specified | §11 Templates | Add: "Template build creates OCI image from base + tools" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Remote template registry (future)
- Template versioning (future)
- Template inheritance/composition (future)
- Template building for fast backend (fast uses host-installed tools)
