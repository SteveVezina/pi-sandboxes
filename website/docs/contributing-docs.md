---
sidebar_position: 7
title: Contributing to these docs
---

# Contributing to these docs

This site (`website/`, Docusaurus) is the **user- and API-facing**
documentation. It is separate from:

- `docs/` — spec-driven-development machinery (feature specs, ADRs, PROPs,
  `plan.md`, contracts). Not published here.
- `AGENTS.md` — agent / contributor working instructions.
- `SPEC.md` — the block spec (changed only through a PROP).

## The rule

Whenever a change adds, removes, or meaningfully alters something a user of
the CLI, daemon API, or SDK would need to know, update the matching page
under `website/docs/` **in the same change**, not a follow-up pass.

Triggers:

- a new or changed `pi-box` command or flag
- a new or changed `/v1/*` route, request field, or response shape
- a new template, config key, or `pi-sandboxd` flag
- a changed runtime-mode behavior or default
- a new lifecycle event or event payload change
- a changed install / build / start step

Purely internal changes (refactors, test-only, deploy plumbing with no
user-visible effect) need no doc update — use judgment.

## Categories

| Folder | Covers | Update when |
|--------|--------|-------------|
| `getting-started/` | install, quickstart, shell, SDKs | the build/start flow, base URLs, or an SDK method changes |
| `cli/` | one page per `pi-box` command group | a command, subcommand, flag, or default changes — check against `cmd/pi-box/` |
| `api/` | one page per endpoint group | a route or request/response shape changes — check against `pkg/daemon/router.go` and `pkg/api/*.go` request structs |
| `runtimes/` | the four modes, host requirements, selection | a backend requirement, default, or the selection logic changes |
| `architecture/` | invariants, layers, security model, `~/.pi-box` layout | a security default, the driver contract, or the on-disk layout changes |

## Authoring rules

- **Document the code, not the aspiration.** If `SPEC.md` /
  `ARCHITECTURE.md` and the code disagree, the site follows the code, and
  the drift is noted in the relevant `docs/features/F{N}-*.md` § Spec Gaps
  (do not edit `SPEC.md` without a PROP).
- Mark not-yet-implemented surfaces with a `:::note` / `:::warning`
  admonition rather than omitting them.
- Keep request/response examples real (copied from handler structs or
  tests), not illustrative pseudocode.
- Links to `docs/features/*` or ADRs: use the GitLab source URL — those
  files are not part of this site.

## New pages

Drop a `.md` file into the right folder; the sidebar auto-generates
(`sidebars.ts`). Only add a `_category_.json` for a new top-level category.

## Before it's done

```bash
pnpm run docs:build      # from the repo root — turbo runs the website build
```

`onBrokenLinks: 'throw'` makes a broken internal link a build failure, not
a silent gap. `.pi/block.yaml` exposes this as `verify.docs`.
