package template

// Default template YAML definitions.
const baseTemplate = `name: base
runtime: auto
base: debian-slim
tools:
  - bash
  - git
  - curl
  - ca-certificates
  - openssh-client
  - tar
  - gzip
  - zstd
  - unzip
  - jq
  - ripgrep
mounts:
  workspace: /workspace
  artifacts: /artifacts
network: restricted
`

const nodeTemplate = `name: node
runtime: auto
base: debian-slim
tools:
  - bash
  - git
  - curl
  - ca-certificates
  - openssh-client
  - node:22
  - npm
  - pnpm
  - corepack
mounts:
  workspace: /workspace
  artifacts: /artifacts
caches:
  npm: /cache/npm
  pnpm: /cache/pnpm
network: restricted
`

const pythonTemplate = `name: python
runtime: auto
base: debian-slim
tools:
  - bash
  - git
  - curl
  - ca-certificates
  - openssh-client
  - python:3.13
  - uv
  - pip
  - venv
mounts:
  workspace: /workspace
  artifacts: /artifacts
caches:
  uv: /cache/uv
  pip: /cache/pip
network: restricted
`

const goTemplate = `name: go
runtime: auto
base: debian-slim
tools:
  - bash
  - git
  - curl
  - ca-certificates
  - openssh-client
  - go:stable
mounts:
  workspace: /workspace
  artifacts: /artifacts
caches:
  GOMODCACHE: /cache/go/mod
  GOCACHE: /cache/go/build
network: restricted
`

const rustTemplate = `name: rust
runtime: auto
base: debian-slim
tools:
  - bash
  - git
  - curl
  - ca-certificates
  - openssh-client
  - rustc
  - cargo
mounts:
  workspace: /workspace
  artifacts: /artifacts
caches:
  cargo: /cache/cargo
  rustup: /cache/rustup
network: restricted
`

const nodePythonTemplate = `name: node-python
runtime: auto
base: debian-slim
tools:
  - bash
  - git
  - curl
  - ca-certificates
  - openssh-client
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
network: restricted
`

const polyglotTemplate = `name: polyglot
runtime: auto
base: debian-slim
tools:
  - bash
  - git
  - curl
  - ca-certificates
  - openssh-client
  - node:22
  - npm
  - pnpm
  - python:3.13
  - uv
  - go:stable
  - rustc
  - cargo
mounts:
  workspace: /workspace
  artifacts: /artifacts
caches:
  npm: /cache/npm
  pnpm: /cache/pnpm
  uv: /cache/uv
  pip: /cache/pip
  GOMODCACHE: /cache/go/mod
  GOCACHE: /cache/go/build
  cargo: /cache/cargo
  rustup: /cache/rustup
network: restricted
`
