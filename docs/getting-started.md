# Getting Started with PI Sandbox

> Deploy your first sandbox and send prompts to a coding agent in under 5 minutes.

## Prerequisites

- **Linux** (recommended) or macOS/Windows via Docker
- **Go 1.21+** (to build from source)
- **Git** (for cloning repositories)
- **curl** (for API calls)

## Quick Start (5 minutes)

### Step 1: Build the binaries

```bash
# Clone the project
git clone https://github.com/pi-sandbox/pi-sandbox.git
cd pi-sandbox

# Build CLI and daemon
go build -o bin/pi-box ./cmd/pi-box/
go build -o bin/pi-sandboxd ./cmd/pi-sandboxd/
```

### Step 2: Start the daemon

```bash
# Start the sandbox daemon (runs in background)
bin/pi-sandboxd &

# Verify it's running
bin/pi-box system status
```

The daemon listens on `~/.pi-box/sandboxd.sock` by default.

### Step 3: Check your setup

```bash
# Validate configuration and available runtimes
bin/pi-box system doctor
```

Expected output:
```
=== pi-sandbox Doctor ===

Status: OK

  [OK]      Pi home directory exists: /home/user/.pi
  [OK]      Directory exists: sandboxes
  [OK]      Directory exists: templates
  [OK]      Directory exists: caches
  [OK]      Available runtime backends: fast
  [OK]      Best available mode: fast
  [OK]      Permissions OK
```

### Step 4: Create a sandbox

```bash
# Create a sandbox with the node-python template
bin/pi-box box create myapp node-python
```

Expected output:
```
Created sandbox: <sandbox-id>
```

### Step 5: Clone a repository

```bash
# Clone any public Git repo into the sandbox
bin/pi-box box clone myapp https://github.com/nodejs/node.git
```

### Step 6: Run commands

```bash
# Execute any command inside the sandbox
bin/pi-box box exec myapp -- echo "Hello from sandbox!"
```

With JSON output for machine consumption:
```bash
bin/pi-box box exec myapp -- pnpm --version --json
```

### Step 7: List sandboxes

```bash
bin/pi-box box list
```

### Step 8: Destroy the sandbox

```bash
bin/pi-box box destroy myapp
```

## Interactive Shell

Open an interactive shell session in a sandbox:

```bash
bin/pi-box box shell myapp
```

This starts a REPL that:
- Reads commands from your terminal
- Executes them inside the sandbox
- Prints stdout/stderr output
- Type `exit` or `quit` to close

Example:
```
myapp> ls
myapp> cat package.json
myapp> exit
```

## SDK Usage

### TypeScript

```typescript
import { createClient } from './sdk/typescript/src/client';

const client = createClient({ baseUrl: 'http://localhost:7777' });

// Create a sandbox
const sandbox = await client.create({
  template: 'node-python',
  mode: 'fast',
  name: 'myapp',
});

// Clone a repo
await client.clone(sandbox.id, 'https://github.com/nodejs/node.git');

// Execute a command
const result = await client.exec(sandbox.id, 'pnpm --version');
console.log(result.stdout);

// Stream output for long-running commands
for await (const event of client.execStream(sandbox.id, 'watch')) {
  if (event.type === 'stdout') {
    process.stdout.write(event.data);
  }
  if (event.type === 'done') {
    console.log(`\nDone: exitCode=${event.exitCode}`);
  }
}

// Read/write files
const content = await client.filesRead(sandbox.id, '/workspace/package.json');
await client.filesWrite(sandbox.id, '/workspace/hello.txt', 'Hello!');

// Snapshot and rollback
await client.snapshotCreate(sandbox.id, 'before-change');
// ... make changes ...
await client.snapshotRollback(sandbox.id, 'before-change');

// Destroy when done
await client.destroy(sandbox.id);
```

### Python

```python
from pi_sandbox import create_client

client = create_client(base_url='http://localhost:7777')

# Create a sandbox
sandbox = client.create(template='node-python', mode='fast', name='myapp')

# Clone a repo
client.clone(sandbox.id, 'https://github.com/nodejs/node.git')

# Execute a command
result = client.exec(sandbox.id, 'pnpm --version')
print(result.stdout)

# Stream output for long-running commands
for event in client.exec_stream(sandbox.id, 'watch'):
    if event.event_type == 'stdout':
        print(event.data, end='')
    if event.event_type == 'done':
        print(f'\nDone: exit_code={event.exit_code}')

# Read/write files
content = client.files_read(sandbox.id, '/workspace/package.json')
client.files_write(sandbox.id, '/workspace/hello.txt', 'Hello!')

# Snapshot and rollback
client.snapshot_create(sandbox.id, 'before-change')
# ... make changes ...
client.snapshot_rollback(sandbox.id, 'before-change')

# Destroy when done
client.destroy(sandbox.id)
```

## API Reference

### Daemon API (Unix socket)

The daemon exposes a REST API over a Unix socket at `~/.pi-box/sandboxd.sock`.

#### Create Sandbox
```http
POST /v1/sandboxes
Content-Type: application/json

{
  "name": "myapp",
  "template": "node-python",
  "mode": "fast"
}
```

#### List Sandboxes
```http
GET /v1/sandboxes
```

#### Get Sandbox
```http
GET /v1/sandboxes/{id}
```

#### Destroy Sandbox
```http
DELETE /v1/sandboxes/{id}
```

#### Clone Repository
```http
POST /v1/sandboxes/{id}/clone
Content-Type: application/json

{ "url": "https://github.com/example/repo.git" }
```

#### Execute Command
```http
POST /v1/sandboxes/{id}/exec
Content-Type: application/json

{
  "command": "pnpm test",
  "cwd": "/workspace",
  "timeoutMs": 60000,
  "maxOutputBytes": 8388608
}
```

#### Read File
```http
GET /v1/sandboxes/{id}/files/read?path=/workspace/package.json
```

#### Write File
```http
POST /v1/sandboxes/{id}/files/write
Content-Type: application/json

{ "path": "/workspace/file.txt", "content": "Hello!" }
```

#### Get Diff
```http
GET /v1/sandboxes/{id}/diff
```

#### Get Patch
```http
GET /v1/sandboxes/{id}/patch
```

#### List Artifacts
```http
GET /v1/sandboxes/{id}/artifacts/list
```

#### Pull Artifacts
```http
POST /v1/sandboxes/{id}/artifacts/pull
Content-Type: application/json

{ "destination": "/tmp/artifacts" }
```

#### Pack Artifacts
```http
POST /v1/sandboxes/{id}/artifacts/pack
Content-Type: application/json

{ "output": "/tmp/artifacts.tar.gz" }
```

#### Create Snapshot
```http
POST /v1/sandboxes/{id}/snapshot/create
Content-Type: application/json

{ "name": "before-refactor" }
```

#### Rollback to Snapshot
```http
POST /v1/sandboxes/{id}/snapshot/rollback
Content-Type: application/json

{ "name": "before-refactor" }
```

#### List Logs
```http
GET /v1/sandboxes/{id}/logs
```

## Available Runtimes

| Mode | Backend | Description |
|------|---------|-------------|
| `fast` | Linux namespaces + cgroups + seccomp | Fastest, trusted environments |
| `compat` | OCI container (runc/Podman) | Maximum compatibility |
| `secure` | gVisor (runsc) | Strong isolation for untrusted code |

Check available runtimes:
```bash
bin/pi-box system doctor
```

## Available Templates

| Template | Tools |
|----------|-------|
| `base` | bash, git, curl, jq, ripgrep |
| `node` | Node.js 22, npm, pnpm |
| `python` | Python 3.13, uv, pip |
| `go` | Go stable toolchain |
| `rust` | rustc, cargo |
| `node-python` | Node.js 22, pnpm, Python 3.13, uv |
| `polyglot` | All of the above |

## System Commands

```bash
# Check daemon status
bin/pi-box system status

# Validate configuration
bin/pi-box system doctor

# Remove old sandbox state
bin/pi-box system prune

# Show storage breakdown
bin/pi-box system disk-usage
```

## Next Steps

- [SPEC.md](../SPEC.md) — Full specification
- [API Contract](../docs/contracts/) — Detailed API schemas
- [Templates](../pkg/template/defaults.go) — Template definitions
- [SDKs](../sdk/) — TypeScript and Python client libraries
