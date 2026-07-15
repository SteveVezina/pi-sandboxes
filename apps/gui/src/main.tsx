import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  Boxes,
  ChevronRight,
  CircleDot,
  Clock3,
  Command,
  Database,
  Folder,
  Gauge,
  HardDrive,
  History,
  KeyRound,
  Layers3,
  ListChecks,
  Loader2,
  MonitorPlay,
  Network,
  Play,
  Plus,
  RotateCcw,
  Settings,
  ShieldCheck,
  SquareTerminal,
  Trash2,
  Wrench,
  FileText,
  AlertTriangle,
  ArrowLeft,
  CheckCircle2,
  XCircle,
  Copy,
  Download,
  X
} from "lucide-react";
import {
  ConnectionState,
  ContextInfo,
  DiffResult,
  DoctorResult,
  ExecResult,
  GuiLogEntry,
  LogEntry,
  PatchResult,
  PiDaemonClient,
  RuntimeBackend,
  RuntimeInfo,
  SandboxInfo,
  SnapshotMeta,
  SupportBundle,
  SystemStatus
} from "./api";
import "./styles.css";

// ─── Constants ───────────────────────────────────────────────────────────────

const DEFAULT_DAEMON_URL = "http://127.0.0.1:7777";
const DAEMON_URL_STORAGE_KEY = "pi.gui.daemonUrl.v2";
const ALLOWED_FOLDERS_STORAGE_KEY = "pi.gui.allowedFolders.v1";
const GUI_DEFAULTS_STORAGE_KEY = "pi.gui.defaults.v1";
const TEMPLATES = ["base", "node", "python", "go", "rust", "node-python", "polyglot"];
const MODES = ["fast", "compat", "secure", "microvm"];
const NETWORK_MODES = ["restricted", "none", "open"];

const NAV_ITEMS = [
  { label: "Dashboard", icon: Command, view: "dashboard" },
  { label: "Sessions", icon: MonitorPlay, view: "sessions" },
  { label: "Templates", icon: Layers3, view: "templates" },
  { label: "Contexts", icon: Network, view: "contexts" },
  { label: "Policies", icon: ShieldCheck, view: "policies" },
  { label: "Settings", icon: Settings, view: "settings" }
];

// ─── Types ───────────────────────────────────────────────────────────────────

type TabId = "exec" | "history" | "logs" | "diff" | "patch" | "artifacts" | "snapshots";

interface GUIDefaults {
  activeContext: string;
  template: string;
  mode: string;
  network: string;
}

interface SessionTabState {
  tab: TabId;
  content?: unknown;
  loading: boolean;
  error: string | null;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function connectionLabel(state: ConnectionState): string {
  if (state === "connected") return "Connected";
  if (state === "checking") return "Checking";
  return "Disconnected";
}

function formatTime(value?: string): string {
  if (!value) return "no activity";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(1)}s`;
  const m = Math.floor(s / 60);
  const sec = Math.floor(s % 60);
  return `${m}m ${sec}s`;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "Ki", "Mi", "Gi"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}B`;
}

function loadGUIDefaults(): GUIDefaults {
  const fallback: GUIDefaults = {
    activeContext: "local",
    template: "node-python",
    mode: "fast",
    network: "restricted"
  };
  try {
    return { ...fallback, ...JSON.parse(localStorage.getItem(GUI_DEFAULTS_STORAGE_KEY) || "{}") };
  } catch {
    return fallback;
  }
}

function redactGuiLogMessage(message: string): string {
  return message.replace(/\/Users\/[^/\s]+/g, "~");
}

// ─── Onboarding / Login View ─────────────────────────────────────────────────

function OnboardingView({
  onConnectLocal,
  onConnectRemote,
  onDismiss,
  lastError,
  isChecking
}: {
  onConnectLocal: (url: string) => void;
  onConnectRemote: (url: string, bearerToken: string) => void;
  onDismiss: () => void;
  lastError: string | null;
  isChecking: boolean;
}) {
  const [daemonUrlInput, setDaemonUrlInput] = useState(DEFAULT_DAEMON_URL);
  const [remoteUrl, setRemoteUrl] = useState("");
  const [remoteAuth, setRemoteAuth] = useState("");
  const [remoteConnecting, setRemoteConnecting] = useState(false);
  const [remoteError, setRemoteError] = useState<string | null>(null);

  const handleLocalConnect = async () => {
    const url = daemonUrlInput.trim() || DEFAULT_DAEMON_URL;
    try {
      const testClient = new PiDaemonClient(url);
      await testClient.health();
      localStorage.setItem(DAEMON_URL_STORAGE_KEY, url);
      onConnectLocal(url);
    } catch (err) {
      setRemoteError(err instanceof Error ? err.message : "Connection failed");
    }
  };

  const handleRemoteConnect = async () => {
    const url = remoteUrl.trim();
    const bearerToken = remoteAuth.trim();
    if (!url) return;
    if (!bearerToken) {
      setRemoteError("Remote HTTP daemon connections require a bearer token.");
      return;
    }
    setRemoteConnecting(true);
    setRemoteError(null);
    try {
      const testClient = new PiDaemonClient(url, bearerToken);
      await testClient.health();
      localStorage.setItem(DAEMON_URL_STORAGE_KEY, url);
      onConnectRemote(url, bearerToken);
    } catch (err) {
      setRemoteError(err instanceof Error ? err.message : "Remote connection failed");
    } finally {
      setRemoteConnecting(false);
    }
  };

  return (
    <div className="onboarding-screen">
      <div className="onboarding-card">
        <div className="onboarding-header">
          <div className="onboarding-logo">
            <Boxes size={48} strokeWidth={1.8} />
          </div>
          <h1>PI Sandbox Workbench</h1>
          <p className="onboarding-subtitle">
            Create and manage isolated sandbox sessions for your coding projects.
          </p>
        </div>

        {lastError && <div className="error-banner">{lastError}</div>}
        {remoteError && <div className="error-banner">{remoteError}</div>}

        <div className="onboarding-choices">
          {/* Local daemon */}
          <div className="onboarding-choice">
            <div className="choice-icon">
              <HardDrive size={28} />
            </div>
            <div className="choice-content">
              <h3>Local daemon</h3>
              <p>Connect to a pi-sandboxd running on this machine.</p>
              <div className="choice-input-row">
                <input
                  value={daemonUrlInput}
                  onChange={(e) => setDaemonUrlInput(e.target.value)}
                  placeholder="http://127.0.0.1:7777"
                />
                <button
                  className="primary-action"
                  onClick={handleLocalConnect}
                  disabled={isChecking}
                >
                  {isChecking ? (
                    <>
                      <Loader2 className="spin" size={16} />
                      Connecting
                    </>
                  ) : (
                    <>
                      <Play size={16} />
                      Connect
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>

          {/* Remote context */}
          <div className="onboarding-choice">
            <div className="choice-icon">
              <Network size={28} />
            </div>
            <div className="choice-content">
              <h3>Remote daemon</h3>
              <p>Connect to a pi-sandboxd on a remote workstation.</p>
              <div className="choice-input-row remote-input-row">
                <input
                  value={remoteUrl}
                  onChange={(e) => setRemoteUrl(e.target.value)}
                  placeholder="http://remote-host:7777"
                />
                <input
                  value={remoteAuth}
                  onChange={(e) => setRemoteAuth(e.target.value)}
                  placeholder="Bearer token"
                  type="password"
                />
                <button
                  className="primary-action"
                  onClick={handleRemoteConnect}
                  disabled={remoteConnecting || !remoteUrl.trim() || !remoteAuth.trim()}
                >
                  {remoteConnecting ? (
                    <>
                      <Loader2 className="spin" size={16} />
                      Connecting
                    </>
                  ) : (
                    <>
                      <Play size={16} />
                      Connect
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div className="onboarding-footer">
          <button className="skip-button" onClick={onDismiss}>Continue without connection</button>
        </div>
      </div>
    </div>
  );
}

// ─── Sidebar ─────────────────────────────────────────────────────────────────

function Sidebar({
  connection,
  health,
  daemonUrl,
  setDaemonUrl,
  refresh,
  activeView,
  setActiveView,
  contexts,
  activeContext,
  onSelectContext
}: {
  connection: ConnectionState;
  health: string;
  daemonUrl: string;
  setDaemonUrl: (url: string) => void;
  refresh: () => void;
  activeView: string;
  setActiveView: (view: string) => void;
  contexts: ContextInfo[];
  activeContext: string;
  onSelectContext: (name: string) => void;
}) {
  return (
    <aside className="sidebar" aria-label="Workbench navigation">
      <div className="brand">
        <div className="brand-mark">
          <Boxes size={25} strokeWidth={2.2} />
        </div>
        <div>
          <strong>PI Sandbox</strong>
          <span>Workbench</span>
        </div>
      </div>

      <nav className="nav-list">
        {NAV_ITEMS.map((item) => {
          const Icon = item.icon;
          return (
            <button
              className={activeView === item.view ? "nav-item active" : "nav-item"}
              key={item.view}
              onClick={() => setActiveView(item.view)}
            >
              <Icon size={19} />
              <span>{item.label}</span>
            </button>
          );
        })}
      </nav>

      <section className="engine-card" aria-label="Daemon connection">
        <div className="engine-topline">
          <Gauge size={17} />
          <span>{activeContext === "local" ? "Local HTTP daemon" : "Remote context"}</span>
        </div>
        <strong className={connection === "connected" ? "connected-text" : "offline-text"}>
          {connection === "checking" ? (
            <Loader2 className="spin" size={15} />
          ) : (
            <CircleDot size={13} />
          )}
          {connectionLabel(connection)}
        </strong>
        <input
          aria-label="Daemon URL"
          className="daemon-input"
          value={daemonUrl}
          onChange={(event) => setDaemonUrl(event.target.value)}
        />
        <button className="secondary-action" onClick={refresh}>
          Re-check
        </button>
        {contexts.length > 0 && (
          <div className="context-select-row">
            <label>
              Active context
              <select
                value={activeContext}
                onChange={(event) => onSelectContext(event.target.value)}
                disabled={connection !== "connected"}
              >
                {contexts.map((ctx) => (
                  <option key={ctx.name} value={ctx.name}>
                    {ctx.name} ({ctx.transport})
                  </option>
                ))}
              </select>
            </label>
          </div>
        )}
      </section>
    </aside>
  );
}

// ─── Dashboard View ──────────────────────────────────────────────────────────

function DashboardView({
  connection,
  health,
  sessions,
  systemStatus,
  runtimeInfo,
  onSelectSession,
  onCreateSession
}: {
  connection: ConnectionState;
  health: string;
  sessions: SandboxInfo[];
  systemStatus: SystemStatus | null;
  runtimeInfo: RuntimeInfo | null;
  onSelectSession: (id: string) => void;
  onCreateSession: () => void;
}) {
  const activeSessions = sessions.filter((s) => s.state === "WARM" || s.state === "EXECUTING");
  const availableBackends = runtimeInfo?.available.length ?? 0;
  const recentSessions = [...sessions].sort((a, b) => {
    const aTime = new Date(a.last_used || a.updated_at || a.created_at || 0).getTime();
    const bTime = new Date(b.last_used || b.updated_at || b.created_at || 0).getTime();
    return bTime - aTime;
  }).slice(0, 5);

  return (
    <div className="content-grid">
      <section className="hero-panel">
        <div>
          <span className="eyebrow">Live daemon workbench</span>
          <h2>What should this sandbox work on?</h2>
          <p>
            Create a warm isolated session, run commands, inspect diffs, and export artifacts
            through the real daemon API.
          </p>
        </div>
        <button className="primary-action large" onClick={onCreateSession} disabled={connection !== "connected"}>
          <Plus size={21} />
          Create session
        </button>
      </section>

      <section className="status-rail" aria-label="Daemon summary">
        <div>
          <span>Connection</span>
          <strong>{connectionLabel(connection)}</strong>
        </div>
        <div>
          <span>Runtime</span>
          <strong>{runtimeInfo?.best || "unknown"}</strong>
        </div>
        <div>
          <span>Backends</span>
          <strong>{availableBackends}</strong>
        </div>
        <div>
          <span>Sessions</span>
          <strong>{systemStatus?.active_sandboxes ?? activeSessions.length}</strong>
        </div>
      </section>

      <section className="onboarding-panel">
        <div className="section-heading">
          <h3>Start workbench</h3>
          <span>Connection</span>
        </div>
        <div className="metric-row">
          <HardDrive size={18} />
          <span>Local daemon</span>
          <strong>{health}</strong>
        </div>
        <div className="metric-row">
          <MonitorPlay size={18} />
          <span>Active sessions</span>
          <strong>{activeSessions.length}</strong>
        </div>
        <div className="metric-row">
          <Gauge size={18} />
          <span>Best runtime</span>
          <strong>{runtimeInfo?.best || "unknown"}</strong>
        </div>
      </section>

      <section className="sessions-panel dashboard-sessions-panel">
        <div className="section-heading">
          <h3>Active sessions</h3>
          <span>{sessions.length} total</span>
        </div>
        <div className="session-list">
          {sessions.length === 0 ? (
            <div className="empty-state">No sessions returned by the daemon.</div>
          ) : sessions.map((session) => (
            <button
              className={session.state === "WARM" || session.state === "EXECUTING" ? "session-row active" : "session-row"}
              key={session.id}
              onClick={() => onSelectSession(session.id)}
            >
              <div className="session-icon">
                <SquareTerminal size={20} />
              </div>
              <div className="session-main">
                <strong>{session.name}</strong>
                <span>
                  {session.id.slice(0, 8)} · {session.template} · {session.mode}
                </span>
                <span>
                  {session.workspace_mode} · {formatTime(session.last_used || session.updated_at)}
                </span>
              </div>
              <div className={`state-pill ${session.state.toLowerCase()}`}>
                <CircleDot size={12} />
                {session.state}
              </div>
              <ChevronRight size={18} />
            </button>
          ))}
        </div>
      </section>

      <section className="diagnostics-panel">
        <div className="section-heading">
          <h3>Quick diagnostics</h3>
          <span>Live</span>
        </div>
        <div className="metric-row">
          <KeyRound size={18} />
          <span>Connection</span>
          <strong>{connectionLabel(connection)}</strong>
        </div>
        <div className="metric-row">
          <Gauge size={18} />
          <span>Daemon health</span>
          <strong>{health}</strong>
        </div>
        <div className="metric-row">
          <MonitorPlay size={18} />
          <span>Active sandboxes</span>
          <strong>{systemStatus?.active_sandboxes ?? sessions.length}</strong>
        </div>
        <div className="metric-row">
          <HardDrive size={18} />
          <span>Total sandboxes</span>
          <strong>{systemStatus?.total_sandboxes ?? sessions.length}</strong>
        </div>
        <div className="metric-row">
          <Wrench size={18} />
          <span>Support bundle</span>
          <span className="muted-text">Export from Settings</span>
        </div>
      </section>

      {recentSessions.length > 0 && (
        <section className="recent-sessions-panel">
          <div className="section-heading">
            <h3>Recent sessions</h3>
          </div>
          <div className="session-list">
            {recentSessions.map((session) => (
              <button
                className="session-row"
                key={session.id}
                onClick={() => onSelectSession(session.id)}
              >
                <div className="session-icon">
                  <Clock3 size={20} />
                </div>
                <div className="session-main">
                  <strong>{session.name}</strong>
                  <span>
                    {session.id.slice(0, 8)} · {session.template} · {formatTime(session.last_used)}
                  </span>
                </div>
                <div className={`state-pill ${session.state.toLowerCase()}`}>
                  <CircleDot size={12} />
                  {session.state}
                </div>
                <ChevronRight size={18} />
              </button>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

// ─── Session Detail View ─────────────────────────────────────────────────────

function SessionDetailView({
  session,
  client,
  defaultNetwork,
  onRefresh,
  onBack
}: {
  session: SandboxInfo;
  client: PiDaemonClient;
  defaultNetwork: string;
  onRefresh: () => void;
  onBack: () => void;
}) {
  const [tab, setTab] = useState<TabId>("exec");
  const [command, setCommand] = useState("pwd");
  const [execResult, setExecResult] = useState<ExecResult | null>(null);
  const [execStreamText, setExecStreamText] = useState("");
  const [isBusy, setIsBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tabs, setTabs] = useState<Record<string, SessionTabState>>({
    exec: { tab: "exec", loading: false, error: null },
    history: { tab: "history", loading: false, error: null, content: [] },
    logs: { tab: "logs", loading: false, error: null, content: [] },
    diff: { tab: "diff", loading: false, error: null },
    patch: { tab: "patch", loading: false, error: null },
    artifacts: { tab: "artifacts", loading: false, error: null, content: [] },
    snapshots: { tab: "snapshots", loading: false, error: null, content: [] }
  });
  const [artifactDestination, setArtifactDestination] = useState("/tmp/pi-gui-artifacts");
  const [snapshotName, setSnapshotName] = useState("gui-checkpoint");
  const [repoUrl, setRepoUrl] = useState("");
  const [executionNetwork, setExecutionNetwork] = useState(defaultNetwork || "restricted");
  const [cloneResult, setCloneResult] = useState<string | null>(null);
  const [artifactPullOutput, setArtifactPullOutput] = useState<string | null>(null);
  const [artifactPackOutput, setArtifactPackOutput] = useState<string | null>(null);
  const [deletingSnapshot, setDeletingSnapshot] = useState<string | null>(null);

  const loadTab = useCallback(
    async (tabId: TabId) => {
      setTabs((prev) => ({ ...prev, [tabId]: { ...prev[tabId], loading: true, error: null } }));
      try {
        let content: unknown;
        switch (tabId) {
          case "history": {
            const resp = await client.logsHistory(session.id);
            content = resp.entries;
            break;
          }
          case "logs": {
            const resp = await client.logs(session.id);
            content = resp.entries;
            break;
          }
          case "diff": {
            const resp = await client.diff(session.id);
            content = resp;
            break;
          }
          case "patch": {
            const resp = await client.patch(session.id);
            content = resp;
            break;
          }
          case "artifacts": {
            const resp = await client.artifacts(session.id);
            content = resp.files;
            break;
          }
          case "snapshots": {
            const resp = await client.snapshots(session.id);
            content = resp.snapshots;
            break;
          }
          default:
            content = null;
        }
        setTabs((prev) => ({
          ...prev,
          [tabId]: { ...prev[tabId], loading: false, content, error: null }
        }));
      } catch (err) {
        setTabs((prev) => ({
          ...prev,
          [tabId]: {
            ...prev[tabId],
            loading: false,
            error: err instanceof Error ? err.message : "Load failed"
          }
        }));
      }
    },
    [session.id, client]
  );

  useEffect(() => {
    setExecutionNetwork(defaultNetwork || "restricted");
  }, [defaultNetwork]);

  useEffect(() => {
    if (tab === "history" || tab === "logs") {
      void loadTab(tab);
    }
    if (tab === "diff") {
      void loadTab("diff");
    }
    if (tab === "patch") {
      void loadTab("patch");
    }
    if (tab === "artifacts") {
      void loadTab("artifacts");
    }
    if (tab === "snapshots") {
      void loadTab("snapshots");
    }
  }, [tab, session.id, loadTab]);

  const runCommand = useCallback(async () => {
    if (!command.trim()) return;
    setIsBusy(true);
    setError(null);
    setExecResult(null);
    setExecStreamText("");
    try {
      let stdout = "";
      let stderr = "";
      for await (const event of client.execStream(session.id, command.trim(), { network: executionNetwork })) {
        if (event.type === "stdout") {
          stdout += event.data || "";
          setExecStreamText(stdout + (stderr ? `\n--- stderr ---\n${stderr}` : ""));
        } else if (event.type === "stderr") {
          stderr += event.data || "";
          setExecStreamText(stdout + (stderr ? `\n--- stderr ---\n${stderr}` : ""));
        } else if (event.type === "done") {
          setExecResult({
            exitCode: event.exitCode ?? -1,
            durationMs: event.durationMs ?? 0,
            truncated: event.truncated ?? false,
            timedOut: event.timedOut ?? false,
            stdout,
            stderr
          });
        }
      }
      await onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to run command");
    } finally {
      setIsBusy(false);
    }
  }, [command, executionNetwork, session.id, client, onRefresh]);

  const cloneRepo = useCallback(async () => {
    if (!repoUrl.trim()) return;
    setIsBusy(true);
    setError(null);
    setCloneResult(null);
    try {
      const result = await client.cloneSandbox(session.id, repoUrl.trim());
      setCloneResult(`Cloned to session: ${result.id}`);
      await onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Clone failed");
    } finally {
      setIsBusy(false);
    }
  }, [repoUrl, session.id, client, onRefresh]);

  const pullArtifacts = useCallback(async () => {
    setIsBusy(true);
    setError(null);
    setArtifactPullOutput(null);
    try {
      await client.artifactPull(session.id, artifactDestination);
      setArtifactPullOutput(`Pulled to ${artifactDestination}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Pull failed");
    } finally {
      setIsBusy(false);
    }
  }, [session.id, client, artifactDestination]);

  const packArtifacts = useCallback(async () => {
    setIsBusy(true);
    setError(null);
    setArtifactPackOutput(null);
    try {
      const output = `/tmp/artifacts-${session.id.slice(0, 8)}.tar.zst`;
      const result = await client.artifactPack(session.id, output);
      setArtifactPackOutput(`Packed: ${output} (${formatBytes(result.bytes)})`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Pack failed");
    } finally {
      setIsBusy(false);
    }
  }, [session.id, client]);

  const createSnapshot = useCallback(async () => {
    setIsBusy(true);
    setError(null);
    try {
      await client.snapshotCreate(session.id, snapshotName);
      await loadTab("snapshots");
      await onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Snapshot create failed");
    } finally {
      setIsBusy(false);
    }
  }, [session.id, client, snapshotName, loadTab, onRefresh]);

  const rollbackSnapshot = useCallback(async (name = snapshotName) => {
    setIsBusy(true);
    setError(null);
    try {
      await client.snapshotRollback(session.id, name);
      await loadTab("snapshots");
      await onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Rollback failed");
    } finally {
      setIsBusy(false);
    }
  }, [session.id, client, snapshotName, loadTab, onRefresh]);

  const deleteSnapshot = useCallback(
    async (name: string) => {
      setDeletingSnapshot(name);
      try {
        await client.snapshotDelete(session.id, name);
        await loadTab("snapshots");
        await onRefresh();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Delete failed");
      } finally {
        setDeletingSnapshot(null);
      }
    },
    [session.id, client, loadTab, onRefresh]
  );

  const destroySession = useCallback(async () => {
    setIsBusy(true);
    setError(null);
    try {
      await client.destroySandbox(session.id);
      await onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Destroy failed");
    } finally {
      setIsBusy(false);
    }
  }, [session.id, client, onRefresh]);

  const tabItems: { id: TabId; label: string; icon: typeof Command }[] = [
    { id: "exec", label: "Exec", icon: Play },
    { id: "history", label: "History", icon: History },
    { id: "logs", label: "Logs", icon: Clock3 },
    { id: "diff", label: "Diff", icon: ListChecks },
    { id: "patch", label: "Patch", icon: FileText },
    { id: "artifacts", label: "Artifacts", icon: Database },
    { id: "snapshots", label: "Snapshots", icon: RotateCcw }
  ];

  return (
    <div className="session-detail">
      <div className="session-detail-header">
        <button className="back-button" onClick={onBack}>
          <ArrowLeft size={15} />
          Back
        </button>
        <div>
          <h2>
            {session.name}
            <span className="session-id">{session.id.slice(0, 12)}</span>
          </h2>
          <p>
            {session.template} · {session.mode} · {session.workspace_mode}
          </p>
        </div>
        <div className="session-detail-actions">
          <div className={`state-pill ${session.state.toLowerCase()}`}>
            <CircleDot size={12} />
            {session.state}
          </div>
          <button className="danger-action" onClick={destroySession} disabled={isBusy}>
            <Trash2 size={16} />
            Destroy
          </button>
        </div>
      </div>

      <div className="session-tabs">
        {tabItems.map((item) => {
          const Icon = item.icon;
          const isActive = tab === item.id;
          const tabState = tabs[item.id];
          return (
            <button
              className={`session-tab ${isActive ? "active" : ""}`}
              key={item.id}
              onClick={() => setTab(item.id)}
            >
              <Icon size={15} />
              <span>{item.label}</span>
              {tabState?.loading && <Loader2 className="spin" size={13} />}
            </button>
          );
        })}
      </div>

      {error && <div className="error-banner">{error}</div>}

      <div className="session-tab-content">
        {/* ── Exec tab ──────────────────────────────────────────────────── */}
        {tab === "exec" && (
          <div className="exec-panel">
            <div className="exec-input-row">
              <input
                value={command}
                onChange={(e) => setCommand(e.target.value)}
                placeholder="Enter command to execute..."
                onKeyDown={(e) => e.key === "Enter" && runCommand()}
              />
              <button className="primary-action" onClick={runCommand} disabled={isBusy}>
                <Play size={16} />
                Run
              </button>
            </div>
            <label className="exec-network-row">
              Network
              <select
                value={executionNetwork}
                onChange={(e) => setExecutionNetwork(e.target.value)}
              >
                {NETWORK_MODES.map((mode) => (
                  <option key={mode} value={mode}>{mode}</option>
                ))}
              </select>
            </label>
            <div className="clone-input-row">
              <input
                value={repoUrl}
                onChange={(e) => setRepoUrl(e.target.value)}
                placeholder="https://github.com/owner/repo.git"
                onKeyDown={(e) => e.key === "Enter" && cloneRepo()}
              />
              <button onClick={cloneRepo} disabled={isBusy}>
                <Download size={16} />
                Clone
              </button>
            </div>
            {cloneResult && <div className="success-msg">{cloneResult}</div>}
            {execResult && (
              <div className="exec-result">
                <div className="exec-meta">
                  <span>exit={execResult.exitCode}</span>
                  <span>duration={formatDuration(execResult.durationMs || 0)}</span>
                  {execResult.timedOut && <span className="timed-out">timed out</span>}
                  {execResult.truncated && <span className="truncated">truncated</span>}
                </div>
              </div>
            )}
            {execStreamText && <pre className="terminal-output">{execStreamText}</pre>}
          </div>
        )}

        {/* ── History tab ───────────────────────────────────────────────── */}
        {tab === "history" && (
          <div className="history-panel">
            {tabs.history.loading && <div className="loading-msg">Loading history...</div>}
            {tabs.history.error && <div className="error-msg">{tabs.history.error}</div>}
            {!tabs.history.loading && !tabs.history.error && (
              <>
                {(tabs.history.content as LogEntry[] | undefined)?.length === 0 && (
                  <div className="empty-state">No command history yet.</div>
                )}
                {(tabs.history.content as LogEntry[] | undefined)?.map((entry) => (
                  <div className="history-entry" key={entry.sequence}>
                    <div className="history-entry-header">
                      <span className="history-sequence">#{entry.sequence}</span>
                      <span className="history-time">{formatTime(entry.timestamp)}</span>
                      <span className={`history-exit ${entry.exitCode === 0 ? "success" : "failure"}`}>
                        {entry.exitCode === 0 ? (
                          <CheckCircle2 size={14} />
                        ) : (
                          <XCircle size={14} />
                        )}
                        {entry.exitCode}
                      </span>
                      {entry.timedOut && <span className="timed-out">⏱ timed out</span>}
                      {entry.truncated && <span className="truncated">⚠ truncated</span>}
                    </div>
                    <div className="history-command">{entry.command}</div>
                    <div className="history-duration">{formatDuration(entry.durationMs)}</div>
                  </div>
                ))}
              </>
            )}
          </div>
        )}

        {/* ── Logs tab ──────────────────────────────────────────────────── */}
        {tab === "logs" && (
          <div className="logs-panel">
            {tabs.logs.loading && <div className="loading-msg">Loading logs...</div>}
            {tabs.logs.error && <div className="error-msg">{tabs.logs.error}</div>}
            {!tabs.logs.loading && !tabs.logs.error && (
              <>
                {(tabs.logs.content as LogEntry[] | undefined)?.length === 0 && (
                  <div className="empty-state">No log entries yet.</div>
                )}
                {(tabs.logs.content as LogEntry[] | undefined)?.map((entry) => (
                  <div className="log-entry" key={entry.sequence}>
                    <div className="log-entry-header">
                      <span className="log-sequence">#{entry.sequence}</span>
                      <span className="log-time">{formatTime(entry.timestamp)}</span>
                    </div>
                    <div className="log-command">{entry.command}</div>
                    <div className="log-meta">
                      <span>exit={entry.exitCode}</span>
                      <span>{formatDuration(entry.durationMs)}</span>
                    </div>
                  </div>
                ))}
              </>
            )}
          </div>
        )}

        {/* ── Diff tab ──────────────────────────────────────────────────── */}
        {tab === "diff" && (
          <div className="diff-panel">
            {tabs.diff.loading && <div className="loading-msg">Computing diff...</div>}
            {tabs.diff.error && <div className="error-msg">{tabs.diff.error}</div>}
            {!tabs.diff.loading && !tabs.diff.error && (
              <>
                {((tabs.diff.content as DiffResult | undefined)?.diff || "").trim() === "" ? (
                  <div className="empty-state">No workspace changes detected.</div>
                ) : (
                  <>
                    <pre className="terminal-output diff-output">
                      {(tabs.diff.content as DiffResult | undefined)?.diff}
                    </pre>
                    <div className="diff-meta">
                      {(tabs.diff.content as DiffResult | undefined)?.timed_out && (
                        <span className="timed-out">timed out</span>
                      )}
                      <span>{formatDuration((tabs.diff.content as DiffResult | undefined)?.duration_ms || 0)}</span>
                    </div>
                  </>
                )}
              </>
            )}
          </div>
        )}

        {/* ── Patch tab ─────────────────────────────────────────────────── */}
        {tab === "patch" && (
          <div className="patch-panel">
            {tabs.patch.loading && <div className="loading-msg">Exporting patch...</div>}
            {tabs.patch.error && <div className="error-msg">{tabs.patch.error}</div>}
            {!tabs.patch.loading && !tabs.patch.error && (
              <>
                {((tabs.patch.content as PatchResult | undefined)?.patch || "").trim() === "" ? (
                  <div className="empty-state">No patch to export.</div>
                ) : (
                  <>
                    <pre className="terminal-output patch-output">
                      {(tabs.patch.content as PatchResult | undefined)?.patch}
                    </pre>
                    <button
                      className="secondary-action"
                      onClick={() => {
                        const text = (tabs.patch.content as PatchResult | undefined)?.patch;
                        if (text) {
                          navigator.clipboard.writeText(text);
                        }
                      }}
                    >
                      <Copy size={14} />
                      Copy patch
                    </button>
                  </>
                )}
              </>
            )}
          </div>
        )}

        {/* ── Artifacts tab ─────────────────────────────────────────────── */}
        {tab === "artifacts" && (
          <div className="artifacts-panel">
            {tabs.artifacts.loading && <div className="loading-msg">Listing artifacts...</div>}
            {tabs.artifacts.error && <div className="error-msg">{tabs.artifacts.error}</div>}
            {!tabs.artifacts.loading && !tabs.artifacts.error && (
              <>
                {(tabs.artifacts.content as string[] | undefined)?.length === 0 && (
                  <div className="empty-state">No artifacts found.</div>
                )}
                <div className="artifact-list">
                  {(tabs.artifacts.content as string[] | undefined)?.map((file) => (
                    <div className="artifact-item" key={file}>
                      <FileText size={16} />
                      <span>{file}</span>
                    </div>
                  ))}
                </div>
                <div className="artifact-actions">
                  <label>
                    Destination
                    <input
                      value={artifactDestination}
                      onChange={(e) => setArtifactDestination(e.target.value)}
                    />
                  </label>
                  <button onClick={pullArtifacts} disabled={isBusy}>
                    <Download size={14} />
                    Pull
                  </button>
                  <button onClick={packArtifacts} disabled={isBusy}>
                    Pack
                  </button>
                </div>
                {artifactPullOutput && <div className="success-msg">{artifactPullOutput}</div>}
                {artifactPackOutput && <div className="success-msg">{artifactPackOutput}</div>}
              </>
            )}
          </div>
        )}

        {/* ── Snapshots tab ─────────────────────────────────────────────── */}
        {tab === "snapshots" && (
          <div className="snapshots-panel">
            {tabs.snapshots.loading && <div className="loading-msg">Loading snapshots...</div>}
            {tabs.snapshots.error && <div className="error-msg">{tabs.snapshots.error}</div>}
            {!tabs.snapshots.loading && !tabs.snapshots.error && (
              <>
                <div className="snapshot-create-row">
                  <input
                    value={snapshotName}
                    onChange={(e) => setSnapshotName(e.target.value)}
                    placeholder="Snapshot name..."
                  />
                  <button onClick={createSnapshot} disabled={isBusy}>
                    <Plus size={14} />
                    Create
                  </button>
                </div>
                <div className="snapshot-list">
                  {(tabs.snapshots.content as SnapshotMeta[] | undefined)?.length === 0 && (
                    <div className="empty-state">No snapshots yet.</div>
                  )}
                  {(tabs.snapshots.content as SnapshotMeta[] | undefined)?.map((snap) => (
                    <div className="snapshot-item" key={snap.name}>
                      <div className="snapshot-info">
                        <strong>{snap.name}</strong>
                        <span>{formatTime(snap.createdAt)}</span>
                        <span>{formatBytes(snap.sizeBytes)}</span>
                        <span>{snap.method}</span>
                      </div>
                      <div className="snapshot-actions">
                        <button
                          onClick={() => void rollbackSnapshot(snap.name)}
                          disabled={isBusy}
                        >
                          <RotateCcw size={14} />
                          Rollback
                        </button>
                        <button
                          onClick={() => deleteSnapshot(snap.name)}
                          disabled={isBusy || deletingSnapshot === snap.name}
                          className="danger-action"
                        >
                          {deletingSnapshot === snap.name ? (
                            <Loader2 className="spin" size={14} />
                          ) : (
                            <Trash2 size={14} />
                          )}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Templates View ──────────────────────────────────────────────────────────

function TemplatesView({ defaults }: { defaults: GUIDefaults }) {
  const templateDescriptions: Record<string, string> = {
    base: "Minimal POSIX environment with coreutils, git, curl, jq, ripgrep",
    node: "Node.js LTS + npm + pnpm + corepack",
    python: "Python 3.x + uv + pip + venv",
    go: "Go stable toolchain + GOMODCACHE + GOCACHE",
    rust: "rustc + cargo + rustup",
    "node-python": "Node.js + Python + pnpm + uv + pip",
    polyglot: "All toolchains in one image"
  };

  return (
    <div className="templates-grid">
      {TEMPLATES.map((template) => (
        <div className="template-card" key={template}>
          <div className="template-header">
            <Layers3 size={22} />
            <strong>{template}</strong>
            {template === defaults.template && (
              <span className="template-default-badge">default</span>
            )}
          </div>
          <p className="template-desc">{templateDescriptions[template] || ""}</p>
        </div>
      ))}
    </div>
  );
}

// ─── Contexts View ───────────────────────────────────────────────────────────

function ContextsView({
  contexts,
  activeContext,
  onSelectContext,
  daemonUrl,
  setDaemonUrl
}: {
  contexts: ContextInfo[];
  activeContext: string;
  onSelectContext: (name: string) => void;
  daemonUrl: string;
  setDaemonUrl: (url: string) => void;
}) {
  return (
    <div className="contexts-list">
      {contexts.length === 0 ? (
        <div className="empty-state">No contexts configured.</div>
      ) : (
        contexts.map((ctx) => (
          <div
            className={`context-card ${ctx.name === activeContext ? "active" : ""}`}
            key={ctx.name}
          >
            <div className="context-info">
              <Network size={20} />
              <div>
                <strong>{ctx.name}</strong>
                <span>{ctx.target}</span>
                <span className="context-transport">{ctx.transport}</span>
                <span className="context-auth">auth: {ctx.auth_type}</span>
              </div>
            </div>
            <div className="context-actions">
              {ctx.name === activeContext && (
                <span className="active-badge">Active</span>
              )}
              {ctx.name !== activeContext && (
                <button onClick={() => onSelectContext(ctx.name)}>Set active</button>
              )}
            </div>
          </div>
        ))
      )}
      <div className="context-card">
        <div className="context-info">
          <HardDrive size={20} />
          <div>
            <strong>Local daemon</strong>
            <span>{daemonUrl}</span>
            <span className="context-transport">http</span>
            <span className="context-auth">no auth</span>
          </div>
        </div>
        <div className="context-actions">
          {daemonUrl === DEFAULT_DAEMON_URL && (
            <span className="active-badge">Direct</span>
          )}
        </div>
      </div>
    </div>
  );
}

// ─── Policies View ───────────────────────────────────────────────────────────

function PoliciesView({
  defaults,
  allowedFolders,
  doctorResult,
  systemStatus,
  runtimeInfo
}: {
  defaults: GUIDefaults;
  allowedFolders: string[];
  doctorResult: DoctorResult | null;
  systemStatus: SystemStatus | null;
  runtimeInfo: RuntimeInfo | null;
}) {
  const policyRows = [
    {
      label: "Default network",
      value: defaults.network,
      detail: "Applied to GUI command execution requests"
    },
    {
      label: "Workspace default",
      value: "copy",
      detail: "Host folders are omitted until explicitly selected and authorized"
    },
    {
      label: "Secrets",
      value: "not mounted",
      detail: "SSH keys, cloud config, Kubernetes config, and Docker socket stay out by default"
    },
    {
      label: "Support bundle",
      value: systemStatus?.support_redacted ? "redacted" : "unknown",
      detail: "Daemon support payload reports redaction status"
    }
  ];

  return (
    <div className="policies-layout">
      <section className="policy-summary-card">
        <div className="policy-summary-icon">
          <ShieldCheck size={26} />
        </div>
        <div>
          <span className="eyebrow">Daemon-enforced guardrails</span>
          <h2>Policy remains authoritative in the daemon.</h2>
          <p>
            GUI preferences shape requests, but daemon policy decides what is accepted for
            workspaces, network access, diagnostics, and support export.
          </p>
        </div>
      </section>

      <section className="policy-card">
        <div className="section-heading">
          <h3>Effective request defaults</h3>
          <span>{runtimeInfo?.best || "runtime unknown"}</span>
        </div>
        <div className="policy-row-list">
          {policyRows.map((row) => (
            <div className="policy-row" key={row.label}>
              <div>
                <strong>{row.label}</strong>
                <span>{row.detail}</span>
              </div>
              <code>{row.value}</code>
            </div>
          ))}
        </div>
      </section>

      <section className="policy-card">
        <div className="section-heading">
          <h3>Allowed folders</h3>
          <span>GUI preference</span>
        </div>
        {allowedFolders.length === 0 ? (
          <div className="empty-state">No folders authorized for GUI-launched workspace access.</div>
        ) : (
          <div className="allowed-folders-settings">
            {allowedFolders.map((folder) => (
              <div className="allowed-folder-item" key={folder}>
                <Folder size={14} />
                <span>{folder}</span>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="policy-card">
        <div className="section-heading">
          <h3>Doctor policy signals</h3>
          <span>{doctorResult?.passed ? "Passing" : doctorResult ? "Needs attention" : "Unknown"}</span>
        </div>
        {!doctorResult ? (
          <div className="empty-state">Daemon diagnostics have not been loaded yet.</div>
        ) : doctorResult.issues.length === 0 ? (
          <div className="doctor-status passed">
            <CheckCircle2 size={18} />
            <span>No doctor issues reported.</span>
          </div>
        ) : (
          <div className="doctor-result">
            {doctorResult.issues.map((issue, i) => (
              <div className={`doctor-issue ${issue.level}`} key={i}>
                <div className="doctor-issue-header">
                  {issue.level === "error" ? (
                    <XCircle size={14} className="issue-error" />
                  ) : issue.level === "warning" ? (
                    <AlertTriangle size={14} className="issue-warning" />
                  ) : (
                    <CircleDot size={14} className="issue-info" />
                  )}
                  <span>{issue.category}</span>
                </div>
                <div className="doctor-issue-msg">{issue.message}</div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

// ─── Settings View ───────────────────────────────────────────────────────────

function SettingsView({
  defaults,
  setDefaults,
  allowedFolders,
  setAllowedFolders,
  doctorResult,
  runtimeInfo,
  supportBundle,
  exportSupportBundle,
  isBusy
}: {
  defaults: GUIDefaults;
  setDefaults: (d: Partial<GUIDefaults>) => void;
  allowedFolders: string[];
  setAllowedFolders: (updater: string[] | ((prev: string[]) => string[])) => void;
  doctorResult: DoctorResult | null;
  supportBundle: SupportBundle | null;
  runtimeInfo: RuntimeInfo | null;
  exportSupportBundle: () => void;
  isBusy: boolean;
}) {
  const [showBundle, setShowBundle] = useState(false);

  return (
    <div className="settings-grid-layout">
      {/* Defaults */}
      <section className="settings-section">
        <h3>Defaults</h3>
        <div className="settings-form">
          <label>
            Template
            <select
              value={defaults.template}
              onChange={(e) => setDefaults({ template: e.target.value })}
            >
              {TEMPLATES.map((t) => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
          </label>
          <label>
            Runtime mode
            <select
              value={defaults.mode}
              onChange={(e) => setDefaults({ mode: e.target.value })}
            >
              {MODES.map((m) => (
                <option key={m} value={m}>{m}</option>
              ))}
            </select>
          </label>
          <label>
            Network mode
            <select
              value={defaults.network}
              onChange={(e) => setDefaults({ network: e.target.value })}
            >
              {NETWORK_MODES.map((n) => (
                <option key={n} value={n}>{n}</option>
              ))}
            </select>
          </label>
        </div>
      </section>

      {/* Allowed folders */}
      <section className="settings-section">
        <h3>Allowed folders</h3>
        <div className="allowed-folders-settings">
          {allowedFolders.length === 0 ? (
            <span className="muted-text">No folders authorized.</span>
          ) : (
            allowedFolders.map((folder) => (
              <div className="allowed-folder-item" key={folder}>
                <Folder size={14} />
                <span>{folder}</span>
                <button onClick={() => setAllowedFolders((f) => f.filter((x) => x !== folder))}>
                  <X size={14} />
                </button>
              </div>
            ))
          )}
        </div>
      </section>

      {/* Diagnostics */}
      <section className="settings-section">
        <h3>Diagnostics</h3>
        {doctorResult && (
          <div className="doctor-result">
            <div className={`doctor-status ${doctorResult.passed ? "passed" : "failed"}`}>
              {doctorResult.passed ? (
                <CheckCircle2 size={18} />
              ) : (
                <AlertTriangle size={18} />
              )}
              <span>{doctorResult.passed ? "All checks passed" : `${doctorResult.issues.length} issue(s)`}</span>
            </div>
            {doctorResult.issues.map((issue, i) => (
              <div className={`doctor-issue ${issue.level}`} key={i}>
                <div className="doctor-issue-header">
                  {issue.level === "error" ? (
                    <XCircle size={14} className="issue-error" />
                  ) : issue.level === "warning" ? (
                    <AlertTriangle size={14} className="issue-warning" />
                  ) : (
                    <CircleDot size={14} className="issue-info" />
                  )}
                  <span>{issue.category}</span>
                </div>
                <div className="doctor-issue-msg">{issue.message}</div>
                {issue.recommendation && (
                  <div className="doctor-issue-rec">→ {issue.recommendation}</div>
                )}
              </div>
            ))}
          </div>
        )}
        <button
          className="secondary-action"
          onClick={() => {
            exportSupportBundle();
            setShowBundle(true);
          }}
          disabled={isBusy}
        >
          <Wrench size={16} />
          Export support bundle
        </button>
        {supportBundle && !showBundle && (
          <button className="secondary-action quiet-action" onClick={() => setShowBundle(true)}>
            View latest bundle
          </button>
        )}
        {showBundle && supportBundle && (
          <div className="support-bundle-viewer">
            <button className="close-bundle" onClick={() => setShowBundle(false)}>
              <X size={16} />
            </button>
            <pre>{JSON.stringify(supportBundle, null, 2)}</pre>
          </div>
        )}
      </section>

      {/* Runtime management */}
      <section className="settings-section">
        <h3>Runtime backends</h3>
        <div className="runtime-list">
          {(!runtimeInfo || runtimeInfo.backends.length === 0) ? (
            <span className="muted-text">No runtime information available.</span>
          ) : (
            runtimeInfo.backends.map((rt: RuntimeBackend) => (
              <div className={`runtime-card ${rt.available ? "available" : "unavailable"}`} key={rt.name}>
                <div className="runtime-header">
                  <div className="runtime-status">
                    {rt.available ? (
                      <CheckCircle2 size={18} className="runtime-ok" />
                    ) : (
                      <XCircle size={18} className="runtime-fail" />
                    )}
                    <strong>{rt.name}</strong>
                  </div>
                  {rt.available && (
                    <span className="security-badge">Security level {rt.security_level}</span>
                  )}
                </div>
                <p className="runtime-desc">{rt.description}</p>
                {rt.available && rt.name === runtimeInfo.best && (
                  <span className="best-runtime-badge">Best available</span>
                )}
              </div>
            ))
          )}
        </div>
        <div className="runtime-summary">
          <span>Best available: <strong>{runtimeInfo?.best || "unknown"}</strong></span>
          <span>Available: <strong>{runtimeInfo?.available.length || 0}</strong></span>
        </div>
      </section>
    </div>
  );
}

// ─── App ─────────────────────────────────────────────────────────────────────

function App() {
  const [daemonUrl, setDaemonUrl] = useState(() => localStorage.getItem(DAEMON_URL_STORAGE_KEY) || DEFAULT_DAEMON_URL);
  const [daemonBearerToken, setDaemonBearerToken] = useState("");
  const [connection, setConnection] = useState<ConnectionState>("checking");
  const [health, setHealth] = useState<string>("checking");
  const [sessions, setSessions] = useState<SandboxInfo[]>([]);
  const [selectedId, setSelectedId] = useState<string>("");
  const [error, setError] = useState<string>("");
  const [isBusy, setIsBusy] = useState(false);
  const [newName, setNewName] = useState("gui-session");
  const [defaults, setDefaults] = useState(() => loadGUIDefaults());
  const [newTemplate, setNewTemplate] = useState(defaults.template);
  const [newMode, setNewMode] = useState(defaults.mode);
  const [projectFolder, setProjectFolder] = useState("");
  const [allowedFolders, setAllowedFolders] = useState<string[]>(() => {
    try {
      return JSON.parse(localStorage.getItem(ALLOWED_FOLDERS_STORAGE_KEY) || "[]");
    } catch {
      return [];
    }
  });
  const [workspaceMode, setWorkspaceMode] = useState<"copy" | "overlay" | "bind">("copy");
  const [activeView, setActiveView] = useState("dashboard");
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [runtimeInfo, setRuntimeInfo] = useState<RuntimeInfo | null>(null);
  const [contexts, setContexts] = useState<ContextInfo[]>([]);
  const [activeContext, setActiveContext] = useState(defaults.activeContext);
  const [doctorResult, setDoctorResult] = useState<DoctorResult | null>(null);
  const [supportBundle, setSupportBundle] = useState<SupportBundle | null>(null);
  const guiLogsRef = useRef<GuiLogEntry[]>([]);

  // Track whether the user has dismissed the onboarding (skip or connect).
  const [onboardingDismissed, setOnboardingDismissed] = useState(false);
  const [lastError, setLastError] = useState<string | null>(null);

  const client = useMemo(() => new PiDaemonClient(daemonUrl, daemonBearerToken), [daemonUrl, daemonBearerToken]);
  const selectedSession = sessions.find((s) => s.id === selectedId);
  const projectFolderSource = projectFolder.trim();
  const folderIsAllowed = projectFolderSource ? allowedFolders.includes(projectFolderSource) : true;

  function recordGuiLog(level: GuiLogEntry["level"], message: string): GuiLogEntry[] {
    const entry: GuiLogEntry = {
      timestamp: new Date().toISOString(),
      level,
      message: redactGuiLogMessage(message)
    };
    guiLogsRef.current = [...guiLogsRef.current, entry].slice(-200);
    return guiLogsRef.current;
  }

  const refresh = useCallback(async () => {
    setError("");
    setLastError(null);
    try {
      setConnection("checking");
      const daemonHealth = await client.health();
      const [sandboxList, status, runtimes, contextResponse, doctor] = await Promise.all([
        client.listSandboxes(),
        client.systemStatus().catch(() => null),
        client.runtimes().catch(() => null),
        client.contexts().catch(() => null),
        client.doctor().catch(() => null)
      ]);
      setHealth(daemonHealth.status || "ok");
      setSessions(sandboxList);
      setSystemStatus(status);
      setRuntimeInfo(runtimes);
      setDoctorResult(doctor);
      if (contextResponse) {
        setContexts(contextResponse.contexts);
        setActiveContext(contextResponse.active);
        setDefaults((current) => ({ ...current, activeContext: contextResponse.active }));
      }
      setConnection("connected");
      setOnboardingDismissed(true);
      recordGuiLog("info", `Connected to daemon at ${daemonUrl}; sessions=${sandboxList.length}`);
      if (!selectedId && sandboxList.length > 0) {
        setSelectedId(sandboxList[0].id);
      }
    } catch (err) {
      setConnection("disconnected");
      setHealth("offline");
      setSessions([]);
      const msg = err instanceof Error ? err.message : "Unable to connect to daemon";
      setError(msg);
      setLastError(msg);
      recordGuiLog("error", `Connection failed for ${daemonUrl}: ${msg}`);
    }
  }, [client, daemonUrl, selectedId]);

  useEffect(() => {
    localStorage.setItem(DAEMON_URL_STORAGE_KEY, daemonUrl);
  }, [daemonUrl]);

  useEffect(() => {
    localStorage.setItem(ALLOWED_FOLDERS_STORAGE_KEY, JSON.stringify(allowedFolders));
  }, [allowedFolders]);

  useEffect(() => {
    localStorage.setItem(GUI_DEFAULTS_STORAGE_KEY, JSON.stringify(defaults));
  }, [defaults]);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(timer);
  }, [refresh]);

  async function createSession() {
    if (!folderIsAllowed && projectFolderSource) {
      setError("Authorize the project folder before creating a GUI-launched session.");
      return;
    }
    setIsBusy(true);
    setError("");
    try {
      const workspace = projectFolderSource
        ? {
            mode: workspaceMode,
            source: projectFolderSource,
            maxSize: "5Gi"
          }
        : undefined;
      const created = await client.createSandbox({
        name: newName.trim() || "gui-session",
        template: newTemplate,
        mode: newMode,
        workspace,
        ttlSeconds: defaults.mode === "microvm" ? 3600 : 7200
      });
      setSelectedId(created.id);
      recordGuiLog("info", `Created sandbox ${created.id} (${created.name || newName})`);
      await refresh();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Unable to create sandbox";
      setError(msg);
      recordGuiLog("error", `Create sandbox failed: ${msg}`);
    } finally {
      setIsBusy(false);
    }
  }

  function authorizeFolder() {
    const folder = projectFolderSource;
    if (!folder) {
      setError("Enter a project folder before authorizing it.");
      return;
    }
    setAllowedFolders((current) => (current.includes(folder) ? current : [...current, folder]));
    setError("");
  }

  function removeAllowedFolder(folder: string) {
    setAllowedFolders((current) => current.filter((item) => item !== folder));
  }

  async function selectContext(name: string) {
    const selectedContext = contexts.find((context) => context.name === name);
    if (selectedContext?.transport === "http" && selectedContext.auth_type === "bearer-token") {
      const msg = "HTTP remote contexts require bearer-token auth. Connect through Remote daemon and provide a token for this GUI session.";
      setError(msg);
      recordGuiLog("warning", `Blocked unauthenticated HTTP context switch for ${name}`);
      return;
    }

    setIsBusy(true);
    setError("");
    try {
      const response = await client.useContext(name);
      const selected = contexts.find((context) => context.name === response.active);
      setActiveContext(response.active);
      setDefaults((current) => ({ ...current, activeContext: response.active }));
      if (selected?.transport === "http" && selected.target.startsWith("http")) {
        setDaemonUrl(selected.target);
        setDaemonBearerToken("");
      } else if (selected?.name === "local") {
        setDaemonUrl(DEFAULT_DAEMON_URL);
        setDaemonBearerToken("");
      }
      recordGuiLog("info", `Switched active context to ${response.active}`);
      await refresh();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Unable to switch context";
      setError(msg);
      recordGuiLog("error", `Context switch failed: ${msg}`);
    } finally {
      setIsBusy(false);
    }
  }

  async function exportSupportBundle() {
    setIsBusy(true);
    setError("");
    try {
      const bundle = await client.supportBundle();
      const logs = recordGuiLog("info", "Support bundle exported from GUI");
      setSupportBundle({ ...bundle, gui_logs: logs });
      setError("");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Unable to export support bundle";
      setError(msg);
      recordGuiLog("error", `Support bundle export failed: ${msg}`);
    } finally {
      setIsBusy(false);
    }
  }

  function updateDefaults(next: Partial<GUIDefaults>) {
    setDefaults((current) => {
      const updated = { ...current, ...next };
      setNewTemplate(updated.template || current.template);
      setNewMode(updated.mode || current.mode);
      return updated;
    });
  }

  // Show onboarding until the user dismisses it or a connection succeeds.
  const showOnboarding = !onboardingDismissed && connection !== "connected";

  if (showOnboarding) {
    return (
      <OnboardingView
        onConnectLocal={(url) => {
          setDaemonUrl(url);
          setDaemonBearerToken("");
          setOnboardingDismissed(true);
          recordGuiLog("info", `Selected local daemon ${url}`);
        }}
        onConnectRemote={(url, bearerToken) => {
          setDaemonUrl(url);
          setDaemonBearerToken(bearerToken);
          setOnboardingDismissed(true);
          recordGuiLog("info", `Selected authenticated remote daemon ${url}`);
        }}
        onDismiss={() => setOnboardingDismissed(true)}
        lastError={lastError}
        isChecking={connection === "checking"}
      />
    );
  }

  return (
    <main className="app-shell">
      <Sidebar
        connection={connection}
        health={health}
        daemonUrl={daemonUrl}
        setDaemonUrl={(url) => {
          setDaemonUrl(url);
          setDaemonBearerToken("");
        }}
        refresh={refresh}
        activeView={activeView}
        setActiveView={setActiveView}
        contexts={contexts}
        activeContext={activeContext}
        onSelectContext={selectContext}
      />

      <section className="workspace">
        <header className="topbar">
          <div>
            <h1>{activeView.charAt(0).toUpperCase() + activeView.slice(1)}</h1>
            <p>
              {connectionLabel(connection)} · daemon {health} · {daemonUrl}
            </p>
          </div>
          <div className="topbar-actions">
            {(activeView === "dashboard" || activeView === "sessions") && (
              <button
                className="primary-action"
                onClick={() => void createSession()}
                disabled={isBusy || connection !== "connected" || !folderIsAllowed}
              >
                {isBusy ? <Loader2 className="spin" size={20} /> : <Play size={20} />}
                New sandbox
              </button>
            )}
          </div>
        </header>

        {error && <div className="error-banner">{error}</div>}

        {/* ── Dashboard view ─────────────────────────────────────────────── */}
        {activeView === "dashboard" && (
          <DashboardView
            connection={connection}
            health={health}
            sessions={sessions}
            systemStatus={systemStatus}
            runtimeInfo={runtimeInfo}
            onSelectSession={(id) => {
              setSelectedId(id);
              setActiveView("sessions");
            }}
            onCreateSession={() => void createSession()}
          />
        )}

        {/* ── Sessions view ──────────────────────────────────────────────── */}
        {activeView === "sessions" && (
          <div className="content-grid">
            <section className="sessions-panel">
              <div className="section-heading">
                <h3>Sessions</h3>
                <button onClick={refresh}>Refresh</button>
              </div>
              <div className="session-list">
                {sessions.length === 0 ? (
                  <div className="empty-state">No sessions returned by the daemon.</div>
                ) : sessions.map((session) => (
                  <button
                    className={
                      selectedId === session.id
                        ? "session-row selected"
                        : session.state === "WARM" || session.state === "EXECUTING"
                        ? "session-row active"
                        : "session-row"
                    }
                    key={session.id}
                    onClick={() => setSelectedId(session.id)}
                  >
                    <div className="session-icon">
                      <SquareTerminal size={20} />
                    </div>
                    <div className="session-main">
                      <strong>{session.name}</strong>
                      <span>
                        {session.id.slice(0, 8)} · {session.template} · {session.mode}
                      </span>
                      <span>
                        {session.workspace_mode} · {formatTime(session.last_used || session.updated_at)}
                      </span>
                    </div>
                    <div className={`state-pill ${session.state.toLowerCase()}`}>
                      <CircleDot size={12} />
                      {session.state}
                    </div>
                    <ChevronRight size={18} />
                  </button>
                ))}
              </div>
            </section>

            {selectedSession ? (
              <SessionDetailView
                session={selectedSession}
                client={client}
                defaultNetwork={defaults.network}
                onRefresh={refresh}
                onBack={() => setSelectedId("")}
              />
            ) : (
              <div className="empty-state">Select a session to view details.</div>
            )}
          </div>
        )}

        {/* ── Templates view ─────────────────────────────────────────────── */}
        {activeView === "templates" && (
          <TemplatesView defaults={defaults} />
        )}

        {/* ── Contexts view ──────────────────────────────────────────────── */}
        {activeView === "contexts" && (
          <ContextsView
            contexts={contexts}
            activeContext={activeContext}
            onSelectContext={selectContext}
            daemonUrl={daemonUrl}
            setDaemonUrl={(url) => {
              setDaemonUrl(url);
              setDaemonBearerToken("");
            }}
          />
        )}

        {/* ── Settings view ──────────────────────────────────────────────── */}
        {activeView === "settings" && (
          <SettingsView
            defaults={defaults}
            setDefaults={updateDefaults}
            allowedFolders={allowedFolders}
            setAllowedFolders={setAllowedFolders}
            doctorResult={doctorResult}
            runtimeInfo={runtimeInfo}
            supportBundle={supportBundle}
            exportSupportBundle={exportSupportBundle}
            isBusy={isBusy}
          />
        )}

        {/* ── Create session panel (always visible on dashboard) ─────────── */}
        {activeView === "dashboard" && (
          <div className="dashboard-lower-grid">
            <section className="authorization-panel">
              <div className="section-heading">
                <h3>Workspace authorization</h3>
                <span>Default {workspaceMode} mode</span>
              </div>
              <label className="field-label" htmlFor="project-folder">Project folder</label>
              <div className="folder-picker">
                <div className="folder-value" id="project-folder">
                  <Folder size={19} />
                  <input
                    value={projectFolder}
                    onChange={(e) => setProjectFolder(e.target.value)}
                    placeholder="/path/to/project"
                  />
                </div>
                <button onClick={authorizeFolder}>Authorize</button>
              </div>
              <div className="mode-grid" aria-label="Workspace mode">
                {(["copy", "overlay", "bind"] as const).map((mode) => (
                  <button
                    className={workspaceMode === mode ? "mode-option active" : "mode-option"}
                    key={mode}
                    onClick={() => setWorkspaceMode(mode)}
                  >
                    {mode}
                  </button>
                ))}
              </div>
              <div className="allowed-folders">
                <div className={projectFolderSource ? (folderIsAllowed ? "workspace-access-state allowed" : "workspace-access-state pending") : "workspace-access-state neutral"}>
                  {projectFolderSource
                    ? folderIsAllowed
                      ? "This folder is authorized for GUI-launched workspace access."
                      : "This folder will not be sent to the daemon until it is authorized."
                    : "No host folder selected. New sandboxes start without host workspace access."}
                </div>
                {allowedFolders.length === 0 ? (
                  <span>No folders authorized.</span>
                ) : (
                  allowedFolders.map((folder) => (
                    <div className="allowed-folder" key={folder}>
                      <span>{folder}</span>
                      <button onClick={() => removeAllowedFolder(folder)}>Remove</button>
                    </div>
                  ))
                )}
              </div>
            </section>

            <section className="create-panel">
              <div className="section-heading">
                <h3>Create sandbox</h3>
                <span>POST /v1/sandboxes</span>
              </div>
              <label className="field-label" htmlFor="sandbox-name">Name</label>
              <input
                id="sandbox-name"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
              />
              <div className="split-fields">
                <label>
                  Template
                  <select value={newTemplate} onChange={(e) => setNewTemplate(e.target.value)}>
                    {TEMPLATES.map((template) => (
                      <option key={template} value={template}>{template}</option>
                    ))}
                  </select>
                </label>
                <label>
                  Mode
                  <select value={newMode} onChange={(e) => setNewMode(e.target.value)}>
                    {MODES.map((mode) => (
                      <option key={mode} value={mode}>{mode}</option>
                    ))}
                  </select>
                </label>
              </div>
              <button
                className="primary-action create-submit"
                onClick={() => void createSession()}
                disabled={isBusy || connection !== "connected" || !folderIsAllowed}
              >
                {isBusy ? <Loader2 className="spin" size={18} /> : <Plus size={18} />}
                Create sandbox
              </button>
            </section>
          </div>
        )}
      </section>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
