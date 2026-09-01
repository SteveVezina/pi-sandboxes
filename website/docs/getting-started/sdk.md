---
sidebar_position: 4
---

# SDKs

Both SDKs are thin clients over the [daemon API](/api/overview). They
connect over the Unix socket, or over HTTP to a remote daemon (which
requires an auth token).

## TypeScript

`sdk/typescript`

```typescript
import { createClient } from '@pi-sandbox/sdk';

const client = createClient({ baseUrl: 'http://127.0.0.1:7777' });

const sandbox = await client.create({
  template: 'node-python',
  mode: 'fast',
  name: 'myapp',
});

await client.clone(sandbox.id, 'https://github.com/nodejs/node.git');

const result = await client.exec(sandbox.id, 'pnpm --version');
console.log(result.stdout);

// Stream a long-running command
for await (const event of client.execStream(sandbox.id, 'pnpm test')) {
  if (event.type === 'stdout') process.stdout.write(event.data);
  if (event.type === 'done') console.log(`exit ${event.exitCode}`);
}

await client.filesWrite(sandbox.id, '/workspace/hello.txt', 'Hello!');
await client.snapshotCreate(sandbox.id, 'before-change');
await client.snapshotRollback(sandbox.id, 'before-change');

await client.destroy(sandbox.id);
```

Methods: `create`, `list`, `get`, `destroy`, `destroyAll`, `exec`,
`execStream`, `clone`, `diff`, `patch`, `filesRead`, `filesWrite`, `logs`,
`artifactsList`, `artifactsPull`, `artifactsPack`, `snapshotCreate`,
`snapshotList`, `snapshotRollback`, `snapshotDelete`.

## Python

`sdk/python`

```python
from pi_sandbox import create_client

client = create_client(base_url="http://127.0.0.1:7777")

sandbox = client.create(template="node-python", mode="fast", name="myapp")
client.clone(sandbox.id, "https://github.com/nodejs/node.git")

result = client.exec(sandbox.id, "pnpm --version")
print(result.stdout)

for event in client.exec_stream(sandbox.id, "pnpm test"):
    if event.event_type == "stdout":
        print(event.data, end="")
    if event.event_type == "done":
        print(f"exit {event.exit_code}")

client.files_write(sandbox.id, "/workspace/hello.txt", "Hello!")
client.snapshot_create(sandbox.id, "before-change")
client.snapshot_rollback(sandbox.id, "before-change")

client.destroy(sandbox.id)
```

Methods mirror the TypeScript client with `snake_case` names.
