/**
 * Pi Sandbox TypeScript SDK
 *
 * Provides programmatic access to the pi-sandbox runtime API.
 * Connects to the daemon via Unix socket or HTTP.
 */

export interface CreateOptions {
  template: string;
  mode: 'fast' | 'compat';
  name?: string;
}

export interface ExecOptions {
  timeoutMs?: number;
  maxOutputBytes?: number;
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

export class SandboxClient {
  private socketPath: string;
  private baseUrl: string | null;

  constructor(options?: { socketPath?: string; baseUrl?: string }) {
    this.socketPath = options?.socketPath || process.env.PI_SOCKET_PATH || '~/.pi/sandboxd.sock';
    this.baseUrl = options?.baseUrl || null;
  }

  private async request(method: string, path: string, body?: unknown): Promise<unknown> {
    if (this.baseUrl) {
      return this.httpRequest(method, path, body);
    }
    throw new Error('SDK requires either socketPath or baseUrl');
  }

  private async httpRequest(method: string, path: string, body?: unknown): Promise<unknown> {
    const url = `${this.baseUrl}${path}`;
    const opts: RequestInit = { method, headers: { 'Content-Type': 'application/json' } };
    if (body) opts.body = JSON.stringify(body);
    const res = await fetch(url, opts);
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

  async exec(id: string, command: string, options?: ExecOptions): Promise<ExecResult> {
    return (await this.request('POST', `/v1/sandboxes/${id}/exec`, {
      command,
      timeoutMs: options?.timeoutMs,
      maxOutputBytes: options?.maxOutputBytes,
    })) as ExecResult;
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
    return (await this.request('GET', `/v1/sandboxes/${id}/files/read`, { path })) as string;
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
}

export function createClient(options?: { socketPath?: string; baseUrl?: string }): SandboxClient {
  return new SandboxClient(options);
}
