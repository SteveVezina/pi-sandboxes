export type ConnectionState = "checking" | "connected" | "disconnected";

export type SandboxInfo = {
  id: string;
  name: string;
  template: string;
  mode: string;
  state: string;
  created_at?: string;
  updated_at?: string;
  ttl_seconds?: number;
  last_used?: string;
  workspace?: string;
  workspace_mode?: string;
};

export type CreateSandboxInput = {
  name: string;
  template: string;
  mode: string;
  workspace?: {
    mode: string;
    source: string;
    maxSize?: string;
  };
};

export type ExecResult = {
  exitCode?: number;
  exit_code?: number;
  durationMs?: number;
  duration_ms?: number;
  stdout?: string;
  stderr?: string;
  truncated?: boolean;
  timedOut?: boolean;
  timed_out?: boolean;
};

export type ExecStreamEvent = {
  type: "stdout" | "stderr" | "done";
  data?: string;
  exitCode?: number;
  durationMs?: number;
  truncated?: boolean;
  timedOut?: boolean;
};

export type DaemonHealth = {
  status: string;
};

export type SystemStatus = {
  daemon: string;
  active_sandboxes: number;
  total_sandboxes: number;
  pi_home: string;
  config_path: string;
};

export type RuntimeInfo = {
  available: string[];
  best: string;
};

export type ContextInfo = {
  name: string;
  target: string;
  transport: string;
  auth_type: string;
};

export type ContextsResponse = {
  active: string;
  contexts: ContextInfo[];
};

export class PiDaemonClient {
  readonly baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
  }

  async health(): Promise<DaemonHealth> {
    return this.request<DaemonHealth>("GET", "/health");
  }

  async systemStatus(): Promise<SystemStatus> {
    return this.request<SystemStatus>("GET", "/v1/system/status");
  }

  async doctor(): Promise<unknown> {
    return this.request("GET", "/v1/system/doctor");
  }

  async runtimes(): Promise<RuntimeInfo> {
    return this.request<RuntimeInfo>("GET", "/v1/system/runtimes");
  }

  async supportBundle(): Promise<unknown> {
    return this.request("GET", "/v1/support-bundle");
  }

  async contexts(): Promise<ContextsResponse> {
    return this.request<ContextsResponse>("GET", "/v1/contexts");
  }

  async useContext(name: string): Promise<{ active: string }> {
    return this.request<{ active: string }>("POST", "/v1/contexts/use", { name });
  }

  async listSandboxes(): Promise<SandboxInfo[]> {
    const shallow = await this.request<Array<Pick<SandboxInfo, "id" | "name" | "state">>>("GET", "/v1/sandboxes");
    const hydrated = await Promise.all(
      shallow.map(async (sandbox) => {
        try {
          return await this.getSandbox(sandbox.id);
        } catch {
          return sandbox as SandboxInfo;
        }
      })
    );
    return hydrated;
  }

  async getSandbox(id: string): Promise<SandboxInfo> {
    return this.request<SandboxInfo>("GET", `/v1/sandboxes/${encodeURIComponent(id)}`);
  }

  async createSandbox(input: CreateSandboxInput): Promise<SandboxInfo> {
    const created = await this.request<{ id: string }>("POST", "/v1/sandboxes", input);
    return this.getSandbox(created.id);
  }

  async destroySandbox(id: string): Promise<void> {
    await this.request("DELETE", `/v1/sandboxes/${encodeURIComponent(id)}`);
  }

  async exec(id: string, command: string): Promise<ExecResult> {
    return this.request<ExecResult>("POST", `/v1/sandboxes/${encodeURIComponent(id)}/exec`, {
      command
    });
  }

  async *execStream(id: string, command: string): AsyncGenerator<ExecStreamEvent> {
    const response = await fetch(`${this.baseUrl}/v1/sandboxes/${encodeURIComponent(id)}/exec?stream=true`, {
      method: "POST",
      headers: {
        "Accept": "application/x-ndjson",
        "Content-Type": "application/json"
      },
      body: JSON.stringify({ command })
    });

    if (!response.ok || !response.body) {
      const message = await response.text();
      throw new Error(message || `streaming exec failed with HTTP ${response.status}`);
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";
      for (const line of lines) {
        if (!line.trim()) continue;
        yield JSON.parse(line) as ExecStreamEvent;
      }
    }

    if (buffer.trim()) {
      yield JSON.parse(buffer) as ExecStreamEvent;
    }
  }

  async logs(id: string): Promise<unknown> {
    return this.request("GET", `/v1/sandboxes/${encodeURIComponent(id)}/logs`);
  }

  async diff(id: string): Promise<unknown> {
    return this.request("GET", `/v1/sandboxes/${encodeURIComponent(id)}/diff`);
  }

  async patch(id: string): Promise<unknown> {
    return this.request("GET", `/v1/sandboxes/${encodeURIComponent(id)}/patch`);
  }

  async artifacts(id: string): Promise<unknown> {
    return this.request("GET", `/v1/sandboxes/${encodeURIComponent(id)}/artifacts/list`);
  }

  async artifactPull(id: string, destination: string): Promise<unknown> {
    return this.request("POST", `/v1/sandboxes/${encodeURIComponent(id)}/artifacts/pull`, { destination });
  }

  async snapshots(id: string): Promise<unknown> {
    return this.request("GET", `/v1/sandboxes/${encodeURIComponent(id)}/snapshot/list`);
  }

  async snapshotCreate(id: string, name: string): Promise<unknown> {
    return this.request("POST", `/v1/sandboxes/${encodeURIComponent(id)}/snapshot/create`, { name });
  }

  async snapshotRollback(id: string, name: string): Promise<unknown> {
    return this.request("POST", `/v1/sandboxes/${encodeURIComponent(id)}/snapshot/rollback`, { name });
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers: {
        "Accept": "application/json",
        "Content-Type": "application/json"
      },
      body: body === undefined ? undefined : JSON.stringify(body)
    });

    if (!response.ok) {
      const message = await response.text();
      throw new Error(message || `${method} ${path} failed with HTTP ${response.status}`);
    }

    if (response.status === 204) {
      return undefined as T;
    }

    return response.json() as Promise<T>;
  }
}
