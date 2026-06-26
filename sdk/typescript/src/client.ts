/**
 * Pi Sandbox TypeScript SDK
 *
 * Provides programmatic access to the pi-sandbox runtime API.
 * Connects to the daemon via Unix socket or HTTP.
 */

export interface CreateOptions {
  template: string;
  mode: 'fast' | 'compat' | 'secure';
  name?: string;
}

export interface ExecOptions {
  timeoutMs?: number;
  maxOutputBytes?: number;
  cwd?: string;
}

export interface ExecResult {
  exitCode: number;
  durationMs: number;
  stdout: string;
  stderr: string;
  truncated: boolean;
  timedOut: boolean;
}

export interface SandboxInfo {
  id: string;
  name: string;
  template: string;
  mode: string;
  state: string;
  createdAt: string;
  lastUsed: string;
}

export interface ExecStreamEvent {
  type: 'stdout' | 'stderr' | 'done';
  data?: string;
  exitCode?: number;
  durationMs?: number;
  truncated?: boolean;
  timedOut?: boolean;
}

export type ExecStreamHandler = (event: ExecStreamEvent) => void;

export class SandboxClient {
  private socketPath: string;
  private baseUrl: string | null;
  private authToken: string | null;

  constructor(options?: {
    socketPath?: string;
    baseUrl?: string;
    /**
     * Bearer token for remote http daemon contexts (F23/ADR-003).
     * Required when baseUrl is set and points at a remote daemon.
     * Never written to disk by the SDK.
     */
    authToken?: string;
  }) {
    this.socketPath = options?.socketPath || process.env.PI_SOCKET_PATH || '~/.pi/sandboxd.sock';
    this.baseUrl = options?.baseUrl || null;
    this.authToken = options?.authToken || process.env.PI_AUTH_TOKEN || null;
  }

  private async request(method: string, path: string, body?: unknown): Promise<unknown> {
    if (this.baseUrl) {
      return this.httpRequest(method, path, body);
    }
    throw new Error('SDK requires either socketPath or baseUrl');
  }

  private async httpRequest(method: string, path: string, body?: unknown): Promise<unknown> {
    const url = `${this.baseUrl}${path}`;
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
    }
    const opts: RequestInit = { method, headers };
    if (body) opts.body = JSON.stringify(body);
    const res = await fetch(url, opts);
    if (res.status === 401 || res.status === 403) {
      // ADR-003: never fall back to unauthenticated access.
      throw new Error(`Remote auth failed: HTTP ${res.status}. Check the bearer token for this context.`);
    }
    if (!res.ok) throw new Error(`API error: ${res.status} ${res.statusText}`);
    return res.json();
  }

  async create(opts: CreateOptions): Promise<SandboxInfo> {
    return (await this.request('POST', '/v1/sandboxes', {
      name: opts.name,
      template: opts.template,
      mode: opts.mode,
    })) as SandboxInfo;
  }

  async list(): Promise<SandboxInfo[]> {
    return (await this.request('GET', '/v1/sandboxes')) as SandboxInfo[];
  }

  async get(id: string): Promise<SandboxInfo> {
    return (await this.request('GET', `/v1/sandboxes/${id}`)) as SandboxInfo;
  }

  async destroy(id: string): Promise<void> {
    await this.request('DELETE', `/v1/sandboxes/${id}`);
  }

  async destroyAll(): Promise<number> {
    const sandboxes = await this.list();
    let count = 0;
    for (const sb of sandboxes) {
      await this.destroy(sb.id);
      count++;
    }
    return count;
  }

  async exec(id: string, command: string, options?: ExecOptions): Promise<ExecResult> {
    return (await this.request('POST', `/v1/sandboxes/${id}/exec`, {
      command,
      cwd: options?.cwd,
      timeoutMs: options?.timeoutMs,
      maxOutputBytes: options?.maxOutputBytes,
    })) as ExecResult;
  }

  /**
   * Exec with streaming output. Yields stdout/stderr chunks as they arrive.
   * Uses SSE-style streaming from the daemon.
   */
  async *execStream(id: string, command: string, options?: ExecOptions): AsyncGenerator<ExecStreamEvent> {
    const url = `${this.baseUrl}/v1/sandboxes/${id}/exec`;
    const body = JSON.stringify({
      command,
      cwd: options?.cwd,
      timeoutMs: options?.timeoutMs,
      maxOutputBytes: options?.maxOutputBytes,
    });

    const res = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
    });

    if (!res.ok) {
      throw new Error(`API error: ${res.status} ${res.statusText}`);
    }

    // Read response line by line for streaming
    const reader = res.body!.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (!line.trim()) continue;
          try {
            const data = JSON.parse(line);
            yield {
              type: data.type || 'stdout',
              data: data.data,
              exitCode: data.exitCode,
              durationMs: data.durationMs,
              truncated: data.truncated,
              timedOut: data.timedOut,
            };
          } catch {
            // Non-JSON line is raw output
            yield { type: 'stdout', data: line };
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }

  async clone(id: string, url: string): Promise<void> {
    await this.request('POST', `/v1/sandboxes/${id}/clone`, { url });
  }

  async diff(id: string): Promise<string> {
    return (await this.request('GET', `/v1/sandboxes/${id}/diff`)) as string;
  }

  async patch(id: string): Promise<string> {
    return (await this.request('GET', `/v1/sandboxes/${id}/patch`)) as string;
  }

  async filesRead(id: string, path: string): Promise<string> {
    return (await this.request('GET', `/v1/sandboxes/${id}/files/read?path=${encodeURIComponent(path)}`)) as string;
  }

  async filesWrite(id: string, path: string, content: string): Promise<void> {
    await this.request('POST', `/v1/sandboxes/${id}/files/write`, { path, content });
  }

  async logs(id: string): Promise<ExecResult[]> {
    return (await this.request('GET', `/v1/sandboxes/${id}/logs`)) as ExecResult[];
  }

  async artifactsList(id: string): Promise<unknown[]> {
    return (await this.request('GET', `/v1/sandboxes/${id}/artifacts`)) as unknown[];
  }

  async artifactsPull(id: string, destination: string): Promise<void> {
    await this.request('POST', `/v1/sandboxes/${id}/artifacts/pull`, { destination });
  }

  async artifactsPack(id: string, output: string): Promise<void> {
    await this.request('POST', `/v1/sandboxes/${id}/artifacts/pack`, { output });
  }

  async snapshotCreate(id: string, name: string): Promise<void> {
    await this.request('POST', `/v1/sandboxes/${id}/snapshot/create`, { name });
  }

  async snapshotList(id: string): Promise<unknown[]> {
    return (await this.request('GET', `/v1/sandboxes/${id}/snapshot/list`)) as unknown[];
  }

  async snapshotRollback(id: string, name: string): Promise<void> {
    await this.request('POST', `/v1/sandboxes/${id}/snapshot/rollback`, { name });
  }

  async snapshotDelete(id: string, name: string): Promise<void> {
    await this.request('POST', `/v1/sandboxes/${id}/snapshot/delete`, { name });
  }
}

export function createClient(options?: {
  socketPath?: string;
  baseUrl?: string;
  authToken?: string;
}): SandboxClient {
  return new SandboxClient(options);
}
