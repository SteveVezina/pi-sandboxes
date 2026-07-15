# PROP-007: Resolve Template `base` Field to OCI Image Name

## Status
✅ Applied to block spec (2026-07-15)

## Block Spec Reference
`SPEC.md` §6 Features (F4, F5), §18 Templates, §20 Daemon API

## Problem

Sandbox creation in `compat` mode fails silently because the template's `base` field (e.g., `debian-slim`) is never resolved to a full OCI image name (e.g., `docker.io/library/debian:bookworm-slim`).

### What happens today

1. User runs `pi-box box create myapp python --mode compat`
2. Daemon creates a sandbox entry in the store with `state: WARM`
3. Daemon calls `compat.CreateContainer()` with `spec.Image = ""`
4. `CreateContainer()` returns `error: container image is required`
5. Sandbox stays WARM but no container exists
6. `pi-box box inspect myapp` returns `sandbox not found`
7. `docker ps` shows no PI containers

### What should happen

1. User runs `pi-box box create myapp python --mode compat`
2. Daemon resolves template `python` → `base: debian-slim` → `docker.io/library/debian:bookworm-slim`
3. Daemon calls `compat.CreateContainer()` with `spec.Image = "docker.io/library/debian:bookworm-slim"`
4. Container is created and started
5. Sandbox is WARM with a running container

## Root Cause

**F4 (Compat Backend)** spec says: *"Pulls/uses the template's OCI image"* — but never specifies **how** the template's `base` field gets resolved to a full image name.

**F5 (Template System)** spec says: *"Templates define OCI images"* — but the template YAML only has `base: debian-slim`, not a full image reference.

**No integration test** exercises the full flow: `create sandbox` → `resolve template` → `create container`.

## Proposed Amendment

### 1. Add image resolution to template system

Add an `image` field to the Template struct that stores the fully qualified OCI image name. The `base` field becomes a shorthand that gets resolved.

**Template YAML:**
```yaml
name: python
base: debian-slim          # shorthand
image: docker.io/library/debian:bookworm-slim  # resolved
```

**Go struct (`pkg/template/template.go`):**
```go
type Template struct {
    Name    string            `yaml:"name"`
    Runtime string            `yaml:"runtime"`
    Base    string            `yaml:"base"`
    Image   string            `yaml:"image"`   // NEW: resolved OCI image
    Tools   []string          `yaml:"tools"`
    Mounts  map[string]string `yaml:"mounts"`
    Caches  map[string]string `yaml:"caches"`
    Network string            `yaml:"network"`
}
```

### 2. Add image resolution function

Add `ResolveImage(base string) string` to `pkg/template/` that maps shorthand base names to full OCI image references:

```go
func ResolveImage(base string) string {
    mappings := map[string]string{
        "debian-slim":      "docker.io/library/debian:bookworm-slim",
        "node":             "docker.io/library/node:22-bookworm",
        "python":           "docker.io/library/python:3.13-bookworm",
        "go":               "docker.io/library/golang:1.24-bookworm",
        "rust":             "docker.io/library/rust:1.80-bookworm",
        "debian":           "docker.io/library/debian:bookworm",
        "ubuntu":           "docker.io/library/ubuntu:24.04",
    }
    if img, ok := mappings[base]; ok {
        return img
    }
    // If base already looks like a full image reference, return as-is
    if strings.Contains(base, "/") || strings.Contains(base, ":") {
        return base
    }
    return "docker.io/library/" + base + ":latest"
}
```

### 3. Update sandbox creation flow

In `pkg/api/sandbox_create.go`, resolve the image before creating the container:

```go
// Resolve template base to full OCI image
image := template.ResolveImage(t.Base)
if t.Image != "" {
    image = t.Image  // explicit image overrides resolved image
}
```

### 4. Add state verification

Before marking a sandbox as `WARM`, verify the runtime resource exists:

```go
// After container creation, verify it's running
state := container.State()
if state != "running" {
    // Clean up and return error
    return nil, fmt.Errorf("container not running after creation: %s", state)
}
```

### 5. Add e2e test

Add `tests/integration/compat_e2e_test.go`:

```go
func TestSandboxCreationCompat(t *testing.T) {
    // Create sandbox via API
    resp := createSandbox(t, "e2e-test", "python", "compat")
    assert.Equal(t, "WARM", resp.State)

    // Verify container exists
    containers := listContainers(t)
    assert.Contains(t, containers, resp.ID)

    // Verify exec works
    result := execCommand(t, resp.ID, "python --version")
    assert.Equal(t, 0, result.ExitCode)
}
```

## Impact Analysis

| Component | Change |
|-----------|--------|
| `pkg/template/template.go` | Add `Image` field, add `ResolveImage()` function |
| `pkg/api/sandbox_create.go` | Resolve image before container creation |
| `pkg/runtime/compat/create.go` | Add state verification after container creation |
| `tests/integration/compat_e2e_test.go` | New e2e test |
| `docs/features/F04-session-lifecycle.md` | Update task plan |
| `docs/features/F15-compat-backend.md` | Update acceptance criteria |

## Dependencies

- F5: Template System (adds `image` field to templates)
- F8: Session Lifecycle (adds state verification)

## Out of Scope

- Multi-registry support (e.g., private registries) — can be added via template `registry` field
- Image pull policy (always, if-not-present) — default to `if-not-present`
- Image cache management — handled by template build process (PROP-006)

## Cascade completed

Applied on 2026-07-15:

- **Block spec:** No submodule changes required (image resolution is implementation detail)
- **Feature specs cascaded:**
  - `docs/features/F15-compat-backend.md` — added resolved gap section
  - `docs/features/F07-template-system.md` — added resolved gap section, `Image` field to Template struct
  - `docs/features/F04-session-lifecycle.md` — added resolved gap section for state verification
- **Code changes:**
  - `pkg/template/template.go` — added `Image` field, `ResolveImage()`, `ResolveTemplateImage()`
  - `pkg/api/sandbox_create.go` — added `createCompatContainer()`, image resolution, state verification
  - `tests/integration/compat_e2e_test.go` — new e2e test for compat sandbox creation
- **Tests:** All 30 test packages pass (including new e2e test)
- **INDEX files:** `docs/proposals/INDEX.md` updated, `docs/features/INDEX.md` unchanged (no status changes needed)
