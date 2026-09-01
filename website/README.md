# PI Agent Sandbox Runtime — documentation site

User- and API-facing documentation for PI Sandbox, built with
[Docusaurus](https://docusaurus.io/). This is **separate** from the
spec-driven-development docs under `../docs/` (features, ADRs, proposals,
plan) and from `../AGENTS.md`.

## Local

```bash
cd website
npm install
npm start        # http://localhost:3000
```

## Build / doc-health gate

```bash
npm run build    # onBrokenLinks: 'throw' — a broken internal link fails the build
npm run typecheck
```

Run `npm run build` before considering any user-visible change done. See
`docs/contributing-docs.md` for the full contributor guide (when to update,
per-category rules).

## Deploy

Vercel, as a standalone project. Set the project's **Root Directory** to
`website/`. `vercel.json` provides the build/install commands. Deploy is
connected separately — not automated from this repo.
