# F25: GUI Workspace Authorization

> Source: `SPEC.md` §6 Features F25
> Status: 🟡 Spec written
> Category: Client / Security

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F25 | GUI Workspace Authorization | Explicit project-folder selection, allowed folder management, and safe bind/copy workspace setup for GUI-launched sessions | M7 |

## Expanded Specification

The GUI must make host workspace access explicit. Before a GUI-launched local session can use a project folder, the user selects a folder, sees the selected path, and chooses or accepts the workspace mode. The default is `copy`.

Allowed folders are GUI preferences that help the user manage repeated folder choices. They are not security enforcement. The daemon remains authoritative for filesystem policy, mount decisions, and rejection of unsafe requests.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-28.1: User must explicitly select a project folder before GUI-launched local workspace access
- [ ] AC-28.2: Selected project folder is displayed before session creation
- [ ] AC-28.3: Default workspace mode is `copy`
- [ ] AC-28.4: `bind` mode requires explicit opt-in
- [ ] AC-28.5: Allowed folders can be listed and removed from GUI settings
- [ ] AC-28.6: Host home directory, SSH keys, cloud config, Kubernetes config, and Docker socket are not mounted by default

## Interface Impact

| Component | Impact |
|-----------|--------|
| `apps/gui/` | Folder picker, authorization flow, allowed folder settings (new — to be created) |
| `~/.pi/config.yaml` | Stores GUI preferences under `gui.allowedFolders` |
| `pi-sandboxd` API | Enforces workspace mode and mount policy |

## Security Considerations

- The GUI must not mount `$HOME` by default.
- The GUI must not mount SSH keys, cloud config, Kubernetes config, or Docker socket by default.
- `bind` mode must be opt-in and visible before session creation.
- Daemon policy overrides GUI preferences.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F17: Policy Enforcement | Internal feature | Implemented |
| F24: Cross-Platform GUI Workbench | Internal feature | Spec written |
| ADR-004 | Architecture decision | Accepted |

## Implementation Approach

Implement workspace authorization as a GUI flow backed by local preferences and daemon policy. Store allowed folders as user-level GUI preferences. Pass selected workspace settings to daemon session creation; do not perform direct host mounts from UI code.

**ADR references:** ADR-004 (GUI Workbench Architecture and Trust Boundaries).
**ADR gaps:** None.

## Tasks

### T25.1: Folder selection and review flow ⚠️

**Description:** Add explicit project-folder selection and confirmation before local workspace access.

**Acceptance criteria:**
- [ ] Folder picker requires explicit user action
- [ ] Selected path is visible before session creation
- [ ] Default workspace mode is `copy`
- [ ] `bind` requires explicit opt-in

**Verification:**
- [ ] GUI component tests for mode defaults and opt-in state
- [ ] Manual smoke verifies folder review before session creation

**Files:** `apps/gui/ (new — to be created)`
**Size:** M
**Depends on:** F24

### T25.2: Allowed folder preferences ⚠️

**Description:** Persist, list, and remove GUI allowed folder preferences.

**Acceptance criteria:**
- [ ] Allowed folders persist under GUI preferences
- [ ] Settings can list allowed folders
- [ ] Settings can remove allowed folders

**Verification:**
- [ ] Unit tests for preference read/write

**Files:** `apps/gui/ (new — to be created)`, `~/.pi/config.yaml` schema handling (new — to be created if needed)
**Size:** M
**Depends on:** T25.1

### T25.3: Policy-safe session creation ⚠️

**Description:** Ensure GUI-launched session creation preserves daemon filesystem policy.

**Acceptance criteria:**
- [ ] GUI passes workspace mode to daemon create request
- [ ] Unsafe host mounts are rejected or surfaced as daemon policy errors
- [ ] Host home, SSH keys, cloud config, Kubernetes config, and Docker socket are not mounted by default

**Verification:**
- [ ] Integration test against mock daemon policy rejection
- [ ] Security smoke for default create-session payload

**Files:** `apps/gui/ (new — to be created)`
**Size:** M
**Depends on:** T25.2

## Verification Plan

- [ ] Component tests verify folder review and workspace mode defaults
- [ ] Preference tests verify allowed folder management
- [ ] Integration tests verify daemon policy errors are displayed

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| — | — | — |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Treating GUI folder preferences as daemon security policy
- Mounting host secrets by default
