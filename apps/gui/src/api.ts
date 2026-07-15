// ─── Types ───────────────────────────────────────────────────────────────────

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
  ttlSeconds?: number;
};

export type ExecOptions = {
  network?: string;
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
  support_redacted?: boolean;
};

export type DoctorIssue = {
  category: string;
  message: string;
  recommendation: string;
  level: "error" | "warning" | "info";
};

export type DoctorResult = {
  passed: boolean;
  issues: DoctorIssue[];
};

// Raw daemon doctor response uses PascalCase.
export type DoctorIssueRaw = {
  Severity: string;
  Message: string;
  Recommendation: string;
};

export type DoctorResultRaw = {
  Passed: boolean;
  Issues: DoctorIssueRaw[];
};

export type RuntimeBackend = {
  name: string;
  available: boolean;
  security_level: number;
  description: string;
};

export type RuntimeInfo = {
  available: string[];
  best: string;
  backends: RuntimeBackend[];
};

export type ContextInfo = {
  name: string;
  target: string;
  transport: string;
  auth_type: string;
  token_env?: string;
  ssh_user?: string;
  ssh_host?: string;
};

export type ContextsResponse = {
  active: string;
  contexts: ContextInfo[];
};

export type ContextInput = {
  name: string;
  target: string;
  transport: string;
  auth_type: string;
  token_env?: string;
  ssh_user?: string;
  ssh_host?: string;
};

export type LogEntry = {
  sequence: number;
  timestamp: string;
  command: string;
  exitCode: number;
  durationMs: number;
  timedOut: boolean;
  truncated: boolean;
  stdoutPath: string;
  stderrPath: string;
};

export type LogsResponse = {
  id: string;
  count: number;
  entries: LogEntry[];
};

export type DiffResult = {
  id: string;
  name: string;
  diff: string;
  timed_out: boolean;
  duration_ms: number;
};

export type PatchResult = {
  id: string;
  name: string;
  patch: string;
  timed_out: boolean;
  duration_ms: number;
};

export type ArtifactsResponse = {
  id: string;
  files: string[];
};

export type SnapshotMeta = {
  name: string;
  sandboxId: string;
  createdAt: string;
  sizeBytes: number;
  method: string;
  workspaceId: string;
};

// Raw daemon snapshot meta uses PascalCase.
export type SnapshotMetaRaw = {
  Name?: string;
  SandboxID?: string;
  CreatedAt?: string;
  SizeBytes?: number;
  Method?: string;
  WorkspaceID?: string;
};

export type SnapshotsResponse = {
  id: string;
  action: string;
  snapshots: SnapshotMeta[];
};

export type GuiLogEntry = {
  timestamp: string;
  level: "info" | "warning" | "error";
  message: string;
};

export type SupportBundle = {
  version: { component: string };
  diagnostics: DoctorResult;
  runtimes: { available: string[]; best: string };
  sessions: { count: number; ids: string[] };
  config: { path: string; pi_home: string };
  gui_logs?: GuiLogEntry[];
  redacted: boolean;
};

// ─── API Client ──────────────────────────────────────────────────────────────

export class PiDaemonClient {
  readonly baseUrl: string;
  readonly bearerToken: string;

  constructor(baseUrl: string, bearerToken = "") {
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.bearerToken = bearerToken;
  }

  async health(): Promise<DaemonHealth> {
    return this.request<DaemonHealth>("GET", "/health");
  }

  async systemStatus(): Promise<SystemStatus> {
    return this.request<SystemStatus>("GET", "/v1/system/status");
  }

  async doctor(): Promise<DoctorResult> {
    const raw = await this.request<DoctorResultRaw>("GET", "/v1/system/doctor");
    // Transform PascalCase daemon response to camelCase.
    const issues: DoctorIssue[] = (raw.Issues || []).map((i) => ({
      category: i.Severity,
      message: i.Message,
      recommendation: i.Recommendation,
      level: (i.Severity === "error" ? "error" : i.Severity === "warning" ? "warning" : "info") as "error" | "warning" | "info"
    }));
    return {
      passed: raw.Passed ?? issues.filter((i) => i.level === "error").length === 0,
      issues
    };
  }

  async runtimes(): Promise<RuntimeInfo> {
    return this.request<RuntimeInfo>("GET", "/v1/system/runtimes");
  }

  async supportBundle(): Promise<SupportBundle> {
    return this.request<SupportBundle>("GET", "/v1/support-bundle");
  }

  async contexts(): Promise<ContextsResponse> {
    return this.request<ContextsResponse>("GET", "/v1/contexts");
  }

  async getContext(name: string): Promise<ContextInfo> {
    return this.request<ContextInfo>("GET", `/v1/contexts/${encodeURIComponent(name)}`);
  }

  async createContext(input: ContextInput): Promise<ContextInfo> {
    return this.request<ContextInfo>("POST", "/v1/contexts", input);
  }

  async updateContext(name: string, input: ContextInput): Promise<ContextInfo> {
    return this.request<ContextInfo>("PUT", `/v1/contexts/${encodeURIComponent(name)}`, input);
  }

  async deleteContext(name: string): Promise<{ deleted: string; active: string }> {
    return this.request<{ deleted: string; active: string }>("DELETE", `/v1/contexts/${encodeURIComponent(name)}`);
  }

  async useContext(name: string): Promise<{ active: string }> {
    return this.request<{ active: string }>("POST", "/v1/contexts/use", { name });
  }

  // List returns the full list — the daemon already hydrates every field.
  async listSandboxes(): Promise<SandboxInfo[]> {
    return this.request<SandboxInfo[]>("GET", "/v1/sandboxes");
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

  async cloneSandbox(id: string, url: string): Promise<{ id: string }> {
    return this.request<{ id: string }>("POST", `/v1/sandboxes/${encodeURIComponent(id)}/clone`, {
      url
    });
  }

  async exec(id: string, command: string, options: ExecOptions = {}): Promise<ExecResult> {
    return this.request<ExecResult>("POST", `/v1/sandboxes/${encodeURIComponent(id)}/exec`, {
      command,
      network: options.network
    });
  }

  async *execStream(id: string, command: string, options: ExecOptions = {}): AsyncGenerator<ExecStreamEvent> {
    const headers: Record<string, string> = {
      Accept: "application/x-ndjson",
      "Content-Type": "application/json"
    };
    if (this.bearerToken) {
      headers.Authorization = `Bearer ${this.bearerToken}`;
    }

    const response = await fetch(
      `${this.baseUrl}/v1/sandboxes/${encodeURIComponent(id)}/exec?stream=true`,
      {
        method: "POST",
        headers,
        body: JSON.stringify({ command, network: options.network })
      }
    );

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

    // Flush any remaining buffered data (final done event).
    if (buffer.trim()) {
      yield JSON.parse(buffer) as ExecStreamEvent;
    }
  }

  // logs returns the raw list endpoint response.
  async logs(id: string): Promise<LogsResponse> {
    const raw = await this.request<LogsResponse>("GET", `/v1/sandboxes/${encodeURIComponent(id)}/logs/list`);
    // Daemon may return null for empty arrays.
    return { ...raw, entries: raw.entries || [] };
  }

  // logsHistory returns command history summaries.
  async logsHistory(id: string): Promise<LogsResponse> {
    const raw = await this.request<LogsResponse>("GET", `/v1/sandboxes/${encodeURIComponent(id)}/logs/history`);
    // Daemon may return null for empty arrays.
    return { ...raw, entries: raw.entries || [] };
  }

  async diff(id: string): Promise<DiffResult> {
    return this.request<DiffResult>("GET", `/v1/sandboxes/${encodeURIComponent(id)}/diff`);
  }

  async patch(id: string): Promise<PatchResult> {
    return this.request<PatchResult>("GET", `/v1/sandboxes/${encodeURIComponent(id)}/patch`);
  }

  async artifacts(id: string): Promise<ArtifactsResponse> {
    const raw = await this.request<ArtifactsResponse>("GET", `/v1/sandboxes/${encodeURIComponent(id)}/artifacts/list`);
    // Daemon may return null for empty arrays.
    return { ...raw, files: raw.files || [] };
  }

  async artifactPull(id: string, destination: string): Promise<{ action: string; destination: string }> {
    return this.request<{ action: string; destination: string }>(
      "POST",
      `/v1/sandboxes/${encodeURIComponent(id)}/artifacts/pull`,
      { destination }
    );
  }

  async artifactPack(id: string, output: string): Promise<{ action: string; output: string; bytes: number }> {
    return this.request<{ action: string; output: string; bytes: number }>(
      "POST",
      `/v1/sandboxes/${encodeURIComponent(id)}/artifacts/pack`,
      { output }
    );
  }

  async snapshots(id: string): Promise<SnapshotsResponse> {
    // Daemon returns PascalCase snapshot meta; transform to camelCase.
    const raw = await this.request<{ id: string; action: string; snapshots: SnapshotMetaRaw[] | null }>(
      "GET",
      `/v1/sandboxes/${encodeURIComponent(id)}/snapshot/list`
    );
    const snapshots = (raw.snapshots || []).map((s) => ({
      name: s.Name || "",
      sandboxId: s.SandboxID || "",
      createdAt: s.CreatedAt || "",
      sizeBytes: s.SizeBytes ?? 0,
      method: s.Method || "",
      workspaceId: s.WorkspaceID || ""
    }));
    return { id: raw.id, action: raw.action, snapshots };
  }

  async snapshotCreate(id: string, name: string): Promise<{ action: string; name: string }> {
    return this.request<{ action: string; name: string }>(
      "POST",
      `/v1/sandboxes/${encodeURIComponent(id)}/snapshot/create`,
      { name }
    );
  }

  async snapshotRollback(id: string, name: string): Promise<{ action: string; name: string }> {
    return this.request<{ action: string; name: string }>(
      "POST",
      `/v1/sandboxes/${encodeURIComponent(id)}/snapshot/rollback`,
      { name }
    );
  }

  async snapshotDelete(id: string, name: string): Promise<{ action: string; name: string }> {
    return this.request<{ action: string; name: string }>(
      "POST",
      `/v1/sandboxes/${encodeURIComponent(id)}/snapshot/delete`,
      { name }
    );
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const headers: Record<string, string> = {
      Accept: "application/json",
      "Content-Type": "application/json"
    };
    if (this.bearerToken) {
      headers.Authorization = `Bearer ${this.bearerToken}`;
    }

    const response = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body)
    });

    if (!response.ok) {
      let message: string;
      try {
        const errBody = await response.json();
        message = errBody.error || `${method} ${path} failed with HTTP ${response.status}`;
      } catch {
        message = `${method} ${path} failed with HTTP ${response.status}`;
      }
      throw new Error(message);
    }

    if (response.status === 204) {
      return undefined as T;
    }

    return response.json() as Promise<T>;
  }
}
