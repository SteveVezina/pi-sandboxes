# PI Agent Sandbox Runtime — documentation site

User- and API-facing documentation for PI Sandbox, built with
[Docusaurus](https://docusaurus.io/). This is **separate** from the
spec-driven-development docs under `../docs/` (features, ADRs, proposals,
plan) and from `../AGENTS.md`.

Part of the repo's **pnpm + turbo** workspace (`pi-sandbox-docs`).

## Local

```bash
# from the repo root
pnpm install
pnpm run docs:dev       # http://localhost:3000

# or directly
pnpm --filter pi-sandbox-docs run start
```

## Build / doc-health gate

```bash
pnpm run docs:build     # turbo -> docusaurus build; onBrokenLinks: 'throw'
pnpm --filter pi-sandbox-docs run typecheck
```

Run `pnpm run docs:build` before considering any user-visible change done.
See `docs/contributing-docs.md` for the full contributor guide.

## Deploy

Vercel, standalone project. Set the project's **Root Directory** to
`website/` and enable *Include source files outside of the Root Directory*
so the build sees the root `pnpm-workspace.yaml` and `pnpm-lock.yaml`.
`vercel.json` provides `pnpm` install/build commands.
