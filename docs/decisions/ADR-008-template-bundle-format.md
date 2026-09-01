# ADR-008: Template Bundle Format

## Status

Accepted (2026-08-31, human-accepted).

Cascade (2026-08-31): no block-spec change. F28 T28.3 unblocked. The
optional `SPEC.md` §18.1 wording clarification (registry refs are an
optional transport) remains a small recommended follow-up PROP, tracked
in F28 § Spec Gaps — not required for T28.3.

---

*(historical) Proposed (2026-08-31) — awaiting human acceptance.*

Unblocks: F28 T28.3 (`pi-box template export` / `import`). Resolves the
"Import/export bundle format + secret-exclusion guarantees" ADR gap
recorded in F28.

Relation to the block spec: `SPEC.md` §18.1 already specifies portable
bundles "for local/team transfer and backup" with "metadata, definition,
optional build artifacts, content digest, and provenance" and forbids a
central registry / marketplace / Docker-Hub-style pull-push. This ADR
chooses the concrete on-disk format and the transport rules; it does not
add a hosted service. A one-line SPEC §18.1 clarification (registry refs
are an *optional transport*, never mandatory) is recommended alongside
acceptance but is not a blocker.

## Context

F28 T28.1/T28.2 gave templates an extended metadata schema, a content
digest (`Template.ContentDigest`), fork lineage, and a local revision
store. T28.3 needs a way to move a template between Pi nodes.

Options considered:

1. **Custom tar bundle** — a `pi-box`-specific archive layout. Simple,
   fully offline, but yet another format to design, version, and tool.
2. **OCI artifact / OCI image layout** — the same content-addressed model
   templates already use for base images. `oras`, `skopeo`, `docker`,
   and `cosign` all understand it. Registry push/pull is a transport the
   user opts into with their own `docker login`.
3. **Mandatory OCI registry** — rejected: contradicts `SPEC.md` §18.1
   non-goals and the local-first invariant. Would need a PROP.

## Decision

### 1. Bundle = OCI image layout (`image-layout` spec), tarred

`pi-box template export <name> -o <file>` writes an **OCI image layout**
directory, tarred:

```
<bundle>.tar
  oci-layout                       {"imageLayoutVersion": "1.0.0"}
  index.json                       -> the manifest
  blobs/sha256/
    <manifest-digest>              application/vnd.pi-sandbox.template.manifest.v1+json
    <config-digest>                application/vnd.pi-sandbox.template.config.v1+json
    <definition-digest>            the template YAML (a layer)
    <artifact-digest>...           optional build artifacts (layers), only with --include-artifacts
```

- **Config blob** carries provenance: `contentDigest`, `lineage`,
  `source`, the exporting `pi-box` version, and an export timestamp.
- **Definition layer** is the template YAML, byte-identical to what
  `template validate` would accept.
- **Manifest** `annotations` include
  `org.opencontainers.image.title = <name>`,
  `dev.pi-sandbox.template.digest = <contentDigest>`.
- Media types are `vnd.pi-sandbox.template.*` so a bundle is
  distinguishable from a container image but still a valid OCI artifact.

### 2. Transport: file by default, OCI ref optional

- `pi-box template export <name> -o bundle.tar` — always available,
  offline.
- `pi-box template export <name> --to oci://ghcr.io/org/tmpl:v1` —
  optional; uses the caller's ambient registry auth (`~/.docker/config.json`
  / credential helpers). The daemon **never** stores registry
  credentials and **never** runs a registry.
- `pi-box template import <bundle.tar | oci://ref>` — resolves either.
- Nothing is fetched implicitly. Sandbox `create` never pulls a template
  from a registry.

### 3. Trust and secret exclusion

- An imported template is installed with `source.type: imported` and is
  **untrusted** until the user runs `template import` explicitly — there
  is no auto-install.
- `import` runs `Template.Validate` and the daemon policy check before
  writing; a bundle that fails either is rejected, not installed.
- `export` refuses to include, and `import` strips: any `env:`/`file:`
  credential material, absolute host paths outside the template model,
  and (for artifact layers) files matching the secret-exclusion denylist
  (`*.pem`, `*.key`, `id_*`, `.env*`, `.git-credentials`, `.netrc`,
  cloud-config dirs). Workspace source files are never included.
- The manifest is content-addressed; `import` reports the
  `contentDigest` so two bundles can be compared before install. Optional
  `cosign` signature verification is a follow-up, not required for M8.

### 4. What this ADR does not do

- No hosted marketplace, no discovery/search, no `pi-box` account.
- No registry operated by Pi Sandbox.
- No change to how base images resolve (PROP-007 still governs `base`).

## Consequences

- T28.3 implements against `github.com/opencontainers/image-spec` types +
  a small OCI-layout reader/writer (or the `oras-go` content store). No
  bespoke archive format.
- Users who already run a registry (GHCR, ECR, Harbor, Zot) get
  team distribution for free; users who don't lose nothing — the tar
  bundle is the baseline.
- `cosign` signing / SBOM attachment become natural later additions
  because the artifact is standard OCI.
- The secret-exclusion denylist is shared with any future snapshot
  export (T28.2b) — one list, one place.

## References

- `SPEC.md` §18.1 (Local template library — portable bundles)
- `docs/features/F28-local-template-library.md` § T28.3, § ADR gaps
- `docs/decisions/ADR-005` (runtime driver contract — shared OCI engine)
- PROP-006 (F28 acceptance), PROP-007 (image resolution)
- OCI Image Layout Specification; OCI Artifacts guidance; ORAS
