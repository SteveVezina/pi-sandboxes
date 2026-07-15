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
import { listen } from "@tauri-apps/api/event";
import {
  isPermissionGranted,
  requestPermission,
  sendNotification
} from "@tauri-apps/plugin-notification";
import {
  ConnectionState,
  ContextInput,
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
import { Button } from "@/components/ui/button";
import "./styles.css";

// ─── Constants ───────────────────────────────────────────────────────────────

const DEFAULT_DAEMON_URL = "http://127.0.0.1:7777";
const DAEMON_URL_STORAGE_KEY = "pi.gui.daemonUrl.v2";
const ALLOWED_FOLDERS_STORAGE_KEY = "pi.gui.allowedFolders.v1";
const GUI_DEFAULTS_STORAGE_KEY = "pi.gui.defaults.v1";
const TEMPLATES = ["base", "node", "python", "go", "rust", "node-python", "polyglot"];
const DEFAULT_RUNTIME_MODE = "compat";
const MODES = ["microvm", "secure", "fast", "compat"];
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
type NotificationPermissionState = "checking" | "granted" | "denied" | "default" | "unsupported";
type WorkspaceMode = "copy" | "overlay" | "bind";

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

interface CreateSandboxDraft {
  name: string;
  template: string;
  mode: string;
  network: string;
  ttlSeconds: number;
  workspaceSource: string;
  workspaceMode: WorkspaceMode;
  workspaceMaxSize: string;
  repoUrl: string;
  rememberDefaults: boolean;
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
    mode: DEFAULT_RUNTIME_MODE,
    network: "restricted"
  };
  try {
    return { ...fallback, ...JSON.parse(localStorage.getItem(GUI_DEFAULTS_STORAGE_KEY) || "{}") };
  } catch {
    return fallback;
  }
}

function availableRuntimeModes(runtimeInfo: RuntimeInfo | null): string[] {
  if (!runtimeInfo?.available?.length) return MODES;
  const available = new Set(runtimeInfo.available);
  return MODES.filter((mode) => available.has(mode));
}

function bestRuntimeMode(runtimeInfo: RuntimeInfo | null, current?: string): string {
  const modes = availableRuntimeModes(runtimeInfo);
  if (current && modes.includes(current)) return current;
  if (runtimeInfo?.best && modes.includes(runtimeInfo.best)) return runtimeInfo.best;
  return modes[0] || DEFAULT_RUNTIME_MODE;
}

function sessionState(session: SandboxInfo): string {
  return (session.state || "").toUpperCase();
}

function canRunSession(session: SandboxInfo): boolean {
  return sessionState(session) === "WARM";
}

function canDestroySession(session: SandboxInfo): boolean {
  const state = sessionState(session);
  return state === "WARM" || state === "EXECUTING";
}

function sessionGateMessage(session: SandboxInfo): string {
  const state = sessionState(session);
  if (state === "WARM") return "Ready for commands and workspace operations.";
  if (state === "EXECUTING") return "Command is running. Workspace operations unlock when the session returns to WARM.";
  if (state === "CREATING") return "Session is still being created. Controls unlock when the daemon reports WARM.";
  if (state === "DESTROYING") return "Session is being destroyed. Controls are locked.";
  if (state === "DESTROYED") return "Session is destroyed. This view is read-only.";
  return "Session is not ready for mutating operations.";
}

function redactGuiLogMessage(message: string): string {
  return message.replace(/\/Users\/[^/\s]+/g, "~");
}

function isTauriRuntime(): boolean {
  return "__TAURI_INTERNALS__" in window;
}

async function readNotificationPermission(): Promise<NotificationPermissionState> {
  if (!isTauriRuntime()) return "unsupported";
  try {
    return (await isPermissionGranted()) ? "granted" : "default";
  } catch {
    return "unsupported";
  }
}

async function ensureNotificationPermission(): Promise<boolean> {
  if (!isTauriRuntime()) return false;
  try {
    if (await isPermissionGranted()) return true;
    return (await requestPermission()) === "granted";
  } catch {
    return false;
  }
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
  onSelectSession
}: {
  connection: ConnectionState;
  health: string;
  sessions: SandboxInfo[];
  systemStatus: SystemStatus | null;
  runtimeInfo: RuntimeInfo | null;
  onSelectSession: (id: string) => void;
}) {
  const readySessions = sessions.filter((s) => s.state === "WARM" || s.state === "EXECUTING");
  const warmSessions = sessions.filter((s) => s.state === "WARM");
  const executingSessions = sessions.filter((s) => s.state === "EXECUTING");
  const availableBackends = runtimeInfo?.available.length ?? 0;
  const recentSessions = [...sessions].sort((a, b) => {
    const aTime = new Date(a.last_used || a.updated_at || a.created_at || 0).getTime();
    const bTime = new Date(b.last_used || b.updated_at || b.created_at || 0).getTime();
    return bTime - aTime;
  }).slice(0, 5);

  return (
    <div className="content-grid dashboard-grid">
      <section className="hero-panel">
        <div>
          <span className="eyebrow">Dashboard</span>
          <h2>Sandbox control</h2>
          <p>
            {warmSessions.length} ready · {executingSessions.length} executing · {sessions.length} total · best runtime {runtimeInfo?.best || "unknown"}
          </p>
        </div>
      </section>

      <section className="status-rail" aria-label="Daemon summary">
        <div>
          <KeyRound size={17} />
          <span>Connection</span>
          <strong>{connectionLabel(connection)}</strong>
        </div>
        <div>
          <Gauge size={17} />
          <span>Runtime</span>
          <strong>{runtimeInfo?.best || "unknown"}</strong>
        </div>
        <div>
          <Database size={17} />
          <span>Backends</span>
          <strong>{availableBackends}</strong>
        </div>
        <div>
          <MonitorPlay size={17} />
          <span>Sessions</span>
          <strong>{systemStatus?.active_sandboxes ?? readySessions.length}</strong>
        </div>
      </section>

      <section className="onboarding-panel">
        <div className="section-heading">
          <h3>Daemon pulse</h3>
          <span>{health}</span>
        </div>
        <div className="metric-row">
          <HardDrive size={18} />
          <span>Local daemon</span>
          <strong>{health}</strong>
        </div>
        <div className="metric-row">
          <MonitorPlay size={18} />
          <span>Ready sessions</span>
          <strong>{readySessions.length}</strong>
        </div>
        <div className="metric-row">
          <Gauge size={18} />
          <span>Best runtime</span>
          <strong>{runtimeInfo?.best || "unknown"}</strong>
        </div>
      </section>

      <section className="sessions-panel dashboard-sessions-panel">
        <div className="section-heading">
          <h3>Ready sessions</h3>
          <span>{warmSessions.length} warm · {executingSessions.length} executing · {sessions.length} total</span>
        </div>
        <div className="session-list">
          {readySessions.length === 0 ? (
            <div className="empty-state dashboard-empty">
              <MonitorPlay size={22} />
              <strong>No live sessions</strong>
              <span>Create a sandbox to start a warm workbench session.</span>
            </div>
          ) : readySessions.map((session) => (
            <button
              className="session-row active"
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
            <span>Last activity</span>
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
  const isSessionReady = canRunSession(session);
  const canDestroy = canDestroySession(session);
  const controlsLocked = isBusy || !isSessionReady;

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
    if (!isSessionReady) {
      setError(sessionGateMessage(session));
      return;
    }
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
  }, [command, executionNetwork, isSessionReady, session, client, onRefresh]);

  const cloneRepo = useCallback(async () => {
    if (!repoUrl.trim()) return;
    if (!isSessionReady) {
      setError(sessionGateMessage(session));
      return;
    }
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
  }, [repoUrl, isSessionReady, session, client, onRefresh]);

  const pullArtifacts = useCallback(async () => {
    if (!isSessionReady) {
      setError(sessionGateMessage(session));
      return;
    }
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
  }, [isSessionReady, session, client, artifactDestination]);

  const packArtifacts = useCallback(async () => {
    if (!isSessionReady) {
      setError(sessionGateMessage(session));
      return;
    }
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
  }, [isSessionReady, session, client]);

  const createSnapshot = useCallback(async () => {
    if (!isSessionReady) {
      setError(sessionGateMessage(session));
      return;
    }
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
  }, [isSessionReady, session, client, snapshotName, loadTab, onRefresh]);

  const rollbackSnapshot = useCallback(async (name = snapshotName) => {
    if (!isSessionReady) {
      setError(sessionGateMessage(session));
      return;
    }
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
  }, [isSessionReady, session, client, snapshotName, loadTab, onRefresh]);

  const deleteSnapshot = useCallback(
    async (name: string) => {
      if (!isSessionReady) {
        setError(sessionGateMessage(session));
        return;
      }
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
    [isSessionReady, session, client, loadTab, onRefresh]
  );

  const destroySession = useCallback(async () => {
    if (!canDestroy) {
      setError(sessionGateMessage(session));
      return;
    }
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
  }, [canDestroy, session, client, onRefresh]);

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
          <button className="danger-action" onClick={destroySession} disabled={isBusy || !canDestroy}>
            <Trash2 size={16} />
            Destroy
          </button>
        </div>
      </div>

      <div className={`session-state-gate ${isSessionReady ? "ready" : "locked"}`}>
        <CircleDot size={13} />
        <span>{sessionGateMessage(session)}</span>
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
                disabled={!isSessionReady}
                onKeyDown={(e) => e.key === "Enter" && isSessionReady && runCommand()}
              />
              <button className="primary-action" onClick={runCommand} disabled={controlsLocked}>
                <Play size={16} />
                Run
              </button>
            </div>
            <label className="exec-network-row">
              Network
              <select
                value={executionNetwork}
                onChange={(e) => setExecutionNetwork(e.target.value)}
                disabled={!isSessionReady}
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
                disabled={!isSessionReady}
                onKeyDown={(e) => e.key === "Enter" && isSessionReady && cloneRepo()}
              />
              <button onClick={cloneRepo} disabled={controlsLocked}>
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
                      disabled={!isSessionReady}
                    />
                  </label>
                  <button onClick={pullArtifacts} disabled={controlsLocked}>
                    <Download size={14} />
                    Pull
                  </button>
                  <button onClick={packArtifacts} disabled={controlsLocked}>
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
                    disabled={!isSessionReady}
                  />
                  <button onClick={createSnapshot} disabled={controlsLocked}>
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
                          disabled={controlsLocked}
                        >
                          <RotateCcw size={14} />
                          Rollback
                        </button>
                        <button
                          onClick={() => deleteSnapshot(snap.name)}
                          disabled={controlsLocked || deletingSnapshot === snap.name}
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

function CreateSandboxDialog({
  open,
  defaults,
  allowedFolders,
  connection,
  runtimeInfo,
  isBusy,
  error,
  onAuthorizeFolder,
  onRemoveAllowedFolder,
  onCancel,
  onCreate
}: {
  open: boolean;
  defaults: GUIDefaults;
  allowedFolders: string[];
  connection: ConnectionState;
  runtimeInfo: RuntimeInfo | null;
  isBusy: boolean;
  error: string;
  onAuthorizeFolder: (folder: string) => void;
  onRemoveAllowedFolder: (folder: string) => void;
  onCancel: () => void;
  onCreate: (draft: CreateSandboxDraft) => void;
}) {
  const runtimeModes = availableRuntimeModes(runtimeInfo);
  const defaultMode = bestRuntimeMode(runtimeInfo, defaults.mode);
  const runtimeModeKey = `${runtimeInfo?.best || ""}:${runtimeInfo?.available.join("|") || ""}`;
  const [draft, setDraft] = useState<CreateSandboxDraft>({
    name: "gui-session",
    template: defaults.template,
    mode: defaultMode,
    network: defaults.network,
    ttlSeconds: defaultMode === "microvm" ? 3600 : 7200,
    workspaceSource: "",
    workspaceMode: "copy",
    workspaceMaxSize: "5Gi",
    repoUrl: "",
    rememberDefaults: true
  });

  useEffect(() => {
    if (!open) return;
    const mode = bestRuntimeMode(runtimeInfo, defaults.mode);
    setDraft((current) => ({
      ...current,
      template: defaults.template,
      mode,
      network: defaults.network,
      ttlSeconds: mode === "microvm" ? 3600 : 7200
    }));
  }, [open, defaults.template, defaults.mode, defaults.network, runtimeModeKey]);

  if (!open) return null;

  const workspaceSource = draft.workspaceSource.trim();
  const folderIsAllowed = workspaceSource ? allowedFolders.includes(workspaceSource) : true;
  const canCreate = connection === "connected" && !isBusy && folderIsAllowed && draft.name.trim().length > 0;
  const runtimeBackends = runtimeInfo?.backends || [];

  function updateDraft(next: Partial<CreateSandboxDraft>) {
    setDraft((current) => ({ ...current, ...next }));
  }

  return (
    <div className="create-dialog-backdrop" role="presentation" onMouseDown={onCancel}>
      <section
        className="create-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="create-dialog-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="create-dialog-header">
          <div>
            <span className="eyebrow">POST /v1/sandboxes</span>
            <h2 id="create-dialog-title">New sandbox</h2>
            <p>Select every launch parameter before the daemon creates the session.</p>
          </div>
          <button className="icon-button" onClick={onCancel} aria-label="Close create sandbox dialog">
            <X size={18} />
          </button>
        </div>

        {error && <div className="create-dialog-error">{error}</div>}

        <div className="create-dialog-body">
          <section className="create-step create-step-identity">
            <div className="create-step-heading">
              <span>01</span>
              <h3>Identity</h3>
            </div>
            <label>
              Name
              <input
                value={draft.name}
                onChange={(event) => updateDraft({ name: event.target.value })}
                placeholder="gui-session"
              />
            </label>
          </section>

          <section className="create-step">
            <div className="create-step-heading">
              <span>02</span>
              <h3>Runtime</h3>
            </div>
            <div className="create-field-grid">
              <label>
                Template
                <select
                  value={draft.template}
                  onChange={(event) => updateDraft({ template: event.target.value })}
                >
                  {TEMPLATES.map((template) => (
                    <option key={template} value={template}>{template}</option>
                  ))}
                </select>
              </label>
              <label>
                Mode
                <select
                  value={draft.mode}
                  onChange={(event) => updateDraft({
                    mode: event.target.value,
                    ttlSeconds: event.target.value === "microvm" ? 3600 : draft.ttlSeconds
                  })}
                >
                  {runtimeModes.map((mode) => {
                    const backend = runtimeBackends.find((rt) => rt.mode === mode);
                    const label = backend ? `${mode}${backend.mode === runtimeInfo?.best ? " (best)" : ""}` : mode;
                    return <option key={mode} value={mode}>{label}</option>;
                  })}
                </select>
              </label>
              <label>
                Exec network default
                <select
                  value={draft.network}
                  onChange={(event) => updateDraft({ network: event.target.value })}
                >
                  {NETWORK_MODES.map((mode) => (
                    <option key={mode} value={mode}>{mode}</option>
                  ))}
                </select>
              </label>
              <label>
                TTL seconds
                <input
                  type="number"
                  min={300}
                  step={300}
                  value={draft.ttlSeconds}
                  onChange={(event) => updateDraft({ ttlSeconds: Number(event.target.value) || 0 })}
                />
              </label>
            </div>
          </section>

          <section className="create-step">
            <div className="create-step-heading">
              <span>03</span>
              <h3>Workspace</h3>
            </div>
            <label>
              Project folder
              <div className="create-folder-row">
                <input
                  value={draft.workspaceSource}
                  onChange={(event) => updateDraft({ workspaceSource: event.target.value })}
                  placeholder="/path/to/project"
                />
                <button onClick={() => onAuthorizeFolder(workspaceSource)} disabled={!workspaceSource}>
                  <Folder size={15} />
                  Authorize
                </button>
              </div>
            </label>
            <div className="create-mode-grid" aria-label="Workspace mode">
              {(["copy", "overlay", "bind"] as const).map((mode) => (
                <button
                  className={draft.workspaceMode === mode ? "mode-option active" : "mode-option"}
                  key={mode}
                  onClick={() => updateDraft({ workspaceMode: mode })}
                >
                  {mode}
                </button>
              ))}
            </div>
            <div className="create-field-grid create-field-grid-narrow">
              <label>
                Workspace max size
                <input
                  value={draft.workspaceMaxSize}
                  onChange={(event) => updateDraft({ workspaceMaxSize: event.target.value })}
                  placeholder="5Gi"
                />
              </label>
              <label>
                Clone repository after create
                <input
                  value={draft.repoUrl}
                  onChange={(event) => updateDraft({ repoUrl: event.target.value })}
                  placeholder="https://github.com/owner/repo.git"
                />
              </label>
            </div>
            <div className={workspaceSource ? (folderIsAllowed ? "workspace-access-state allowed" : "workspace-access-state pending") : "workspace-access-state neutral"}>
              {workspaceSource
                ? folderIsAllowed
                  ? "This folder is authorized for GUI-launched workspace access."
                  : "Authorize this folder before creating a sandbox with host workspace access."
                : "No host folder selected. The create request will omit workspace access."}
            </div>
            {allowedFolders.length > 0 && (
              <div className="create-allowed-folders">
                {allowedFolders.map((folder) => (
                  <button
                    key={folder}
                    onClick={() => updateDraft({ workspaceSource: folder })}
                    title={folder}
                  >
                    <Folder size={13} />
                    <span>{folder}</span>
                    <X
                      size={13}
                      onClick={(event) => {
                        event.stopPropagation();
                        onRemoveAllowedFolder(folder);
                      }}
                    />
                  </button>
                ))}
              </div>
            )}
          </section>
        </div>

        <div className="create-review-strip">
          <label className="remember-defaults">
            <input
              type="checkbox"
              checked={draft.rememberDefaults}
              onChange={(event) => updateDraft({ rememberDefaults: event.target.checked })}
            />
            Remember template, runtime, and network
          </label>
          <div className="create-review-values">
            <code>{draft.template}</code>
            <code>{draft.mode}</code>
            <code>{draft.network}</code>
            <code>{draft.ttlSeconds}s</code>
            <code>{workspaceSource ? draft.workspaceMode : "no workspace"}</code>
          </div>
          <button className="secondary-action" onClick={onCancel}>Cancel</button>
          <button className="primary-action" onClick={() => onCreate(draft)} disabled={!canCreate}>
            {isBusy ? <Loader2 className="spin" size={18} /> : <Play size={18} />}
            Create sandbox
          </button>
        </div>
      </section>
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
  client,
  onRefresh
}: {
  contexts: ContextInfo[];
  activeContext: string;
  onSelectContext: (name: string) => void;
  daemonUrl: string;
  client: PiDaemonClient;
  onRefresh: () => Promise<void>;
}) {
  const blankContext: ContextInput = {
    name: "",
    target: "https://daemon.example.test",
    transport: "http",
    auth_type: "bearer-token",
    token_env: "",
    ssh_user: "",
    ssh_host: ""
  };
  const [selectedName, setSelectedName] = useState("");
  const [draft, setDraft] = useState<ContextInput>(blankContext);
  const [isSaving, setIsSaving] = useState(false);
  const [contextError, setContextError] = useState<string | null>(null);
  const [contextMessage, setContextMessage] = useState<string | null>(null);

  const selectForEdit = (ctx: ContextInfo) => {
    setSelectedName(ctx.name);
    setDraft({
      name: ctx.name,
      target: ctx.target,
      transport: ctx.transport,
      auth_type: ctx.auth_type,
      token_env: ctx.token_env || "",
      ssh_user: ctx.ssh_user || "",
      ssh_host: ctx.ssh_host || ""
    });
    setContextError(null);
    setContextMessage(null);
  };

  const startNewContext = () => {
    setSelectedName("");
    setDraft(blankContext);
    setContextError(null);
    setContextMessage(null);
  };

  const updateTransport = (transport: string) => {
    const authType = transport === "http" ? "bearer-token" : transport === "ssh" ? "ssh-agent" : "none";
    setDraft((current) => ({
      ...current,
      transport,
      auth_type: authType,
      target:
        transport === "unix"
          ? "unix://~/.pi-box/sandboxd.sock"
          : transport === "ssh"
          ? "ssh://host.example.test"
          : current.target.startsWith("unix://")
          ? "https://daemon.example.test"
          : current.target
    }));
  };

  const saveContext = async () => {
    setIsSaving(true);
    setContextError(null);
    setContextMessage(null);
    try {
      const input: ContextInput = {
        ...draft,
        name: draft.name.trim(),
        target: draft.target.trim(),
        token_env: draft.token_env?.trim(),
        ssh_user: draft.ssh_user?.trim(),
        ssh_host: draft.ssh_host?.trim()
      };
      if (selectedName) {
        await client.updateContext(selectedName, input);
        setContextMessage(`Updated context "${selectedName}".`);
      } else {
        await client.createContext(input);
        setSelectedName(input.name);
        setContextMessage(`Created context "${input.name}".`);
      }
      await onRefresh();
    } catch (err) {
      setContextError(err instanceof Error ? err.message : "Unable to save context");
    } finally {
      setIsSaving(false);
    }
  };

  const deleteContext = async (name: string) => {
    setIsSaving(true);
    setContextError(null);
    setContextMessage(null);
    try {
      await client.deleteContext(name);
      setContextMessage(`Deleted context "${name}".`);
      startNewContext();
      await onRefresh();
    } catch (err) {
      setContextError(err instanceof Error ? err.message : "Unable to delete context");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="contexts-workbench">
      <section className="contexts-list-panel">
        <div className="section-heading">
          <h3>Daemon contexts</h3>
          <button onClick={startNewContext}>
            <Plus size={14} />
            New context
          </button>
        </div>
        <div className="contexts-list">
          {contexts.length === 0 ? (
            <div className="empty-state">No contexts configured.</div>
          ) : (
            contexts.map((ctx) => (
              <div
                className={`context-card ${ctx.name === activeContext ? "active" : ""} ${ctx.name === selectedName ? "selected" : ""}`}
                key={ctx.name}
              >
                <button className="context-info-button" onClick={() => selectForEdit(ctx)}>
                  <Network size={20} />
                  <div>
                    <strong>{ctx.name}</strong>
                    <span>{ctx.target}</span>
                    <span className="context-transport">{ctx.transport}</span>
                    <span className="context-auth">
                      auth: {ctx.auth_type}
                      {ctx.token_env ? ` · token env: ${ctx.token_env}` : ""}
                    </span>
                  </div>
                </button>
                <div className="context-actions">
                  {ctx.name === activeContext && (
                    <span className="active-badge">Active</span>
                  )}
                  {ctx.name !== activeContext && (
                    <button onClick={() => onSelectContext(ctx.name)}>Set active</button>
                  )}
                  {ctx.name !== "local" && (
                    <button className="danger-inline" onClick={() => deleteContext(ctx.name)} disabled={isSaving}>
                      <Trash2 size={14} />
                    </button>
                  )}
                </div>
              </div>
            ))
          )}
          <div className="context-card direct">
            <div className="context-info-static">
              <HardDrive size={20} />
              <div>
                <strong>Direct local URL</strong>
                <span>{daemonUrl}</span>
                <span className="context-transport">http</span>
                <span className="context-auth">renderer connection target</span>
              </div>
            </div>
            <span className="active-badge">Direct</span>
          </div>
        </div>
      </section>

      <section className="context-editor-panel">
        <div className="section-heading">
          <h3>{selectedName ? `Edit ${selectedName}` : "Create context"}</h3>
          <span>{selectedName ? "PUT /v1/contexts/{name}" : "POST /v1/contexts"}</span>
        </div>
        {contextError && <div className="error-msg">{contextError}</div>}
        {contextMessage && <div className="success-msg">{contextMessage}</div>}
        {selectedName === "local" ? (
          <div className="empty-state compact-empty">
            <ShieldCheck size={18} />
            <span>The reserved local context is inspectable but cannot be edited or deleted.</span>
          </div>
        ) : (
          <div className="context-form">
            <label>
              Name
              <input
                value={draft.name}
                onChange={(e) => setDraft((current) => ({ ...current, name: e.target.value }))}
                placeholder="remote-dev"
                disabled={Boolean(selectedName)}
              />
            </label>
            <label>
              Transport
              <select value={draft.transport} onChange={(e) => updateTransport(e.target.value)}>
                <option value="http">http</option>
                <option value="ssh">ssh</option>
                <option value="unix">unix</option>
              </select>
            </label>
            <label className="context-form-wide">
              Target
              <input
                value={draft.target}
                onChange={(e) => setDraft((current) => ({ ...current, target: e.target.value }))}
                placeholder="https://daemon.example.test"
              />
            </label>
            <label>
              Auth type
              <select
                value={draft.auth_type}
                onChange={(e) => setDraft((current) => ({ ...current, auth_type: e.target.value }))}
              >
                <option value="bearer-token">bearer-token</option>
                <option value="ssh-agent">ssh-agent</option>
                <option value="none">none</option>
              </select>
            </label>
            {draft.auth_type === "bearer-token" && (
              <label>
                Token env var
                <input
                  value={draft.token_env || ""}
                  onChange={(e) => setDraft((current) => ({ ...current, token_env: e.target.value }))}
                  placeholder="PI_REMOTE_TOKEN"
                />
              </label>
            )}
            {draft.auth_type === "ssh-agent" && (
              <>
                <label>
                  SSH user
                  <input
                    value={draft.ssh_user || ""}
                    onChange={(e) => setDraft((current) => ({ ...current, ssh_user: e.target.value }))}
                    placeholder="svezina"
                  />
                </label>
                <label>
                  SSH host
                  <input
                    value={draft.ssh_host || ""}
                    onChange={(e) => setDraft((current) => ({ ...current, ssh_host: e.target.value }))}
                    placeholder="gpu-box.local"
                  />
                </label>
              </>
            )}
            <div className="context-security-note context-form-wide">
              <KeyRound size={16} />
              <span>Raw bearer tokens and private keys are not stored. Use an environment variable for HTTP bearer auth or ssh-agent for SSH.</span>
            </div>
            <button className="primary-action context-form-wide" onClick={saveContext} disabled={isSaving}>
              {isSaving ? <Loader2 className="spin" size={16} /> : <CheckCircle2 size={16} />}
              {selectedName ? "Save context" : "Create context"}
            </button>
          </div>
        )}
      </section>
    </div>
  );
}

// ─── Policies View ───────────────────────────────────────────────────────────

function PoliciesView({
  defaults,
  allowedFolders,
  doctorResult,
  systemStatus,
  runtimeInfo,
  connection,
  activeContext,
  sessions,
  onOpenSettings,
  onOpenSessions
}: {
  defaults: GUIDefaults;
  allowedFolders: string[];
  doctorResult: DoctorResult | null;
  systemStatus: SystemStatus | null;
  runtimeInfo: RuntimeInfo | null;
  connection: ConnectionState;
  activeContext: string;
  sessions: SandboxInfo[];
  onOpenSettings: () => void;
  onOpenSessions: () => void;
}) {
  const activeSessions = sessions.filter((s) => s.state === "WARM" || s.state === "EXECUTING");
  const doctorErrors = doctorResult?.issues.filter((issue) => issue.level === "error").length ?? 0;
  const doctorWarnings = doctorResult?.issues.filter((issue) => issue.level === "warning").length ?? 0;

  const enforcementRows = [
    {
      control: "Network execution",
      requested: defaults.network,
      enforced: "daemon network policy",
      source: "POST /v1/sandboxes/{id}/exec",
      action: "Change default",
      onAction: onOpenSettings
    },
    {
      control: "Workspace access",
      requested: allowedFolders.length === 0 ? "none authorized" : `${allowedFolders.length} folder(s)`,
      enforced: "explicit GUI grant + daemon create",
      source: "POST /v1/sandboxes",
      action: "Manage grants",
      onAction: onOpenSettings
    },
    {
      control: "Secrets",
      requested: "not requested",
      enforced: "not mounted by default",
      source: "runtime policy",
      action: "Fixed guardrail"
    },
    {
      control: "Support bundle",
      requested: "export on demand",
      enforced: systemStatus?.support_redacted ? "redacted" : "redaction unknown",
      source: "GET /v1/support-bundle",
      action: "Export",
      onAction: onOpenSettings
    }
  ];

  const apiCoverage = [
    {
      group: "Session lifecycle",
      wired: "create, list, inspect, destroy",
      surface: "Dashboard, Sessions",
      status: "wired"
    },
    {
      group: "Session operations",
      wired: "exec stream, clone, diff, patch, artifacts, snapshots, logs",
      surface: "Session detail",
      status: "wired"
    },
    {
      group: "Daemon context",
      wired: "list contexts, switch active context, proxy active remote",
      surface: "Contexts, sidebar",
      status: "wired"
    },
    {
      group: "System diagnostics",
      wired: "status, doctor, runtimes, support bundle",
      surface: "Policies, Settings",
      status: "wired"
    },
    {
      group: "CLI-only gaps",
      wired: "interactive shell, file read/write, template build/update, prune, disk usage",
      surface: "not first-class GUI views yet",
      status: "gap"
    }
  ];

  return (
    <div className="policies-layout">
      <section className="policy-summary-card">
        <div className="policy-summary-icon">
          <ShieldCheck size={26} />
        </div>
        <div>
          <span className="eyebrow">Daemon policy map</span>
          <h2>See what the GUI asks for and what the daemon enforces.</h2>
          <p>
            This view is mostly read-only by design. Change preferences in Settings or
            session controls; the daemon remains the source of truth.
          </p>
        </div>
        <div className="policy-summary-actions">
          <button onClick={onOpenSettings}>
            <Settings size={15} />
            Settings
          </button>
          <button onClick={onOpenSessions}>
            <MonitorPlay size={15} />
            Sessions
          </button>
        </div>
      </section>

      <section className="policy-strip" aria-label="Policy summary">
        <div>
          <span>Connection</span>
          <strong>{connectionLabel(connection)}</strong>
        </div>
        <div>
          <span>Context</span>
          <strong>{activeContext}</strong>
        </div>
        <div>
          <span>Runtime</span>
          <strong>{runtimeInfo?.best || "unknown"}</strong>
        </div>
        <div>
          <span>Doctor</span>
          <strong>{doctorResult ? (doctorErrors > 0 ? `${doctorErrors} error(s)` : doctorWarnings > 0 ? `${doctorWarnings} warning(s)` : "passing") : "unknown"}</strong>
        </div>
        <div>
          <span>Active</span>
          <strong>{systemStatus?.active_sandboxes ?? activeSessions.length} sandbox(es)</strong>
        </div>
      </section>

      <section className="policy-card policy-card-wide">
        <div className="section-heading">
          <h3>Effective enforcement</h3>
          <span>{runtimeInfo?.best || "runtime unknown"}</span>
        </div>
        <div className="policy-table">
          <div className="policy-table-head">
            <span>Area</span>
            <span>GUI request</span>
            <span>Daemon enforcement</span>
            <span>Source</span>
            <span>Action</span>
          </div>
          {enforcementRows.map((row) => (
            <div className="policy-table-row" key={row.control}>
              <strong>{row.control}</strong>
              <code>{row.requested}</code>
              <span>{row.enforced}</span>
              <code>{row.source}</code>
              <div>
                {row.onAction ? (
                  <button onClick={row.onAction}>{row.action}</button>
                ) : (
                  <span className="muted-text">{row.action}</span>
                )}
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="policy-card">
        <div className="section-heading">
          <h3>API and CLI coverage</h3>
          <span>GUI wiring</span>
        </div>
        <div className="coverage-list">
          {apiCoverage.map((item) => (
            <div className={`coverage-row ${item.status}`} key={item.group}>
              <div>
                <strong>{item.group}</strong>
                <span>{item.wired}</span>
              </div>
              <code>{item.surface}</code>
            </div>
          ))}
        </div>
      </section>

      <section className="policy-card">
        <div className="section-heading">
          <h3>Allowed folders</h3>
          <span>{allowedFolders.length} GUI grant(s)</span>
        </div>
        {allowedFolders.length === 0 ? (
          <div className="empty-state compact-empty">
            <Folder size={18} />
            <span>No folders authorized for GUI-launched workspace access.</span>
          </div>
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

      <section className="policy-card policy-card-wide">
        <div className="section-heading">
          <h3>Doctor policy signals</h3>
          <span>{doctorResult?.passed ? "Passing" : doctorResult ? "Needs attention" : "Unknown"}</span>
        </div>
        {!doctorResult ? (
          <div className="empty-state">Daemon diagnostics have not been loaded yet.</div>
        ) : (
          <div className="doctor-signal-list">
            {doctorResult.issues.length === 0 && (
              <div className="doctor-status passed">
                <CheckCircle2 size={18} />
                <span>No doctor issues reported.</span>
              </div>
            )}
            {doctorResult.issues.map((issue, i) => (
              <div className={`doctor-signal ${issue.level}`} key={i}>
                <div className="doctor-signal-severity">
                  {issue.level === "error" ? <XCircle size={15} className="issue-error" /> : null}
                  {issue.level === "warning" ? <AlertTriangle size={15} className="issue-warning" /> : null}
                  {issue.level === "info" ? <CircleDot size={15} className="issue-info" /> : null}
                  <strong>{issue.level}</strong>
                </div>
                <span>{issue.message}</span>
                {issue.recommendation ? <small>{issue.recommendation}</small> : <small />}
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
  const runtimeModes = availableRuntimeModes(runtimeInfo);

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
              {runtimeModes.map((m) => (
                <option key={m} value={m}>{m}{m === runtimeInfo?.best ? " (best)" : ""}</option>
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
              <div className={`runtime-card ${rt.available ? "available" : "unavailable"}`} key={rt.mode}>
                <div className="runtime-header">
                  <div className="runtime-status">
                    {rt.available ? (
                      <CheckCircle2 size={18} className="runtime-ok" />
                    ) : (
                      <XCircle size={18} className="runtime-fail" />
                    )}
                    <strong>{rt.mode}</strong>
                  </div>
                  <span className="security-badge">Isolation tier {rt.isolation_tier} · compat tier {rt.compat_tier}</span>
                </div>
                <p className="runtime-desc">{rt.description}</p>
                {!rt.available && rt.reason && (
                  <p className="runtime-desc runtime-reason">{rt.reason}</p>
                )}
                {rt.available && rt.mode === runtimeInfo.best && (
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

function Topbar({
  activeView,
  connection,
  health,
  daemonUrl,
  isBusy,
  onCreateSession
}: {
  activeView: string;
  connection: ConnectionState;
  health: string;
  daemonUrl: string;
  isBusy: boolean;
  onCreateSession: () => void;
}) {
  const canCreate = connection === "connected" && !isBusy;

  return (
    <header className="topbar">
      <div className="topbar-title">
        <h1>{activeView.charAt(0).toUpperCase() + activeView.slice(1)}</h1>
        <p>
          {connectionLabel(connection)} · daemon {health} · {daemonUrl}
        </p>
      </div>
      <div className="topbar-actions">
        {(activeView === "dashboard" || activeView === "sessions") && (
          <Button
            className="primary-action"
            onClick={onCreateSession}
            disabled={!canCreate}
          >
            {isBusy ? <Loader2 className="spin" size={18} /> : <Play size={18} />}
            New sandbox
          </Button>
        )}
      </div>
    </header>
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
  const [defaults, setDefaults] = useState(() => loadGUIDefaults());
  const [allowedFolders, setAllowedFolders] = useState<string[]>(() => {
    try {
      return JSON.parse(localStorage.getItem(ALLOWED_FOLDERS_STORAGE_KEY) || "[]");
    } catch {
      return [];
    }
  });
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [activeView, setActiveView] = useState("dashboard");
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [runtimeInfo, setRuntimeInfo] = useState<RuntimeInfo | null>(null);
  const [contexts, setContexts] = useState<ContextInfo[]>([]);
  const [activeContext, setActiveContext] = useState(defaults.activeContext);
  const [doctorResult, setDoctorResult] = useState<DoctorResult | null>(null);
  const [supportBundle, setSupportBundle] = useState<SupportBundle | null>(null);
  const [notificationPermission, setNotificationPermission] = useState<NotificationPermissionState>("checking");
  const guiLogsRef = useRef<GuiLogEntry[]>([]);
  const lastConnectionRef = useRef<ConnectionState>("checking");
  const refreshInFlightRef = useRef(false);

  // Track whether the user has dismissed the onboarding (skip or connect).
  const [onboardingDismissed, setOnboardingDismissed] = useState(false);
  const [lastError, setLastError] = useState<string | null>(null);

  const client = useMemo(() => new PiDaemonClient(daemonUrl, daemonBearerToken), [daemonUrl, daemonBearerToken]);
  const selectedSession = sessions.find((s) => s.id === selectedId);

  const notifySystem = useCallback(
    async (title: string, body: string) => {
      if (!(await ensureNotificationPermission())) {
        setNotificationPermission(await readNotificationPermission());
        return;
      }
      try {
        sendNotification({ title, body });
        setNotificationPermission("granted");
      } catch {
        setNotificationPermission("unsupported");
      }
    },
    []
  );

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
    if (refreshInFlightRef.current) return;
    refreshInFlightRef.current = true;
    setError("");
    setLastError(null);
    try {
      if (lastConnectionRef.current !== "connected") {
        setConnection("checking");
      }
      const daemonHealth = await client.health();
      setHealth(daemonHealth.status || "ok");
      setConnection("connected");
      if (lastConnectionRef.current === "disconnected") {
        void notifySystem("PI Sandbox connected", `Daemon is reachable at ${daemonUrl}.`);
      }
      lastConnectionRef.current = "connected";
      setOnboardingDismissed(true);

      const [sandboxList, status, runtimes, contextResponse, doctor] = await Promise.allSettled([
        client.listSandboxes(),
        client.systemStatus(),
        client.runtimes(),
        client.contexts(),
        client.doctor()
      ]);

      if (sandboxList.status === "fulfilled") {
        setSessions(sandboxList.value);
        if (!selectedId && sandboxList.value.length > 0) {
          setSelectedId(sandboxList.value[0].id);
        }
        recordGuiLog("info", `Daemon refresh ok at ${daemonUrl}; sessions=${sandboxList.value.length}`);
      } else {
        recordGuiLog("warning", `Session refresh failed while daemon stayed healthy: ${sandboxList.reason}`);
      }
      if (status.status === "fulfilled") {
        setSystemStatus(status.value);
      }
      if (runtimes.status === "fulfilled") {
        setRuntimeInfo(runtimes.value);
      }
      if (doctor.status === "fulfilled") {
        setDoctorResult(doctor.value);
      }
      if (contextResponse.status === "fulfilled") {
        setContexts(contextResponse.value.contexts);
        setActiveContext(contextResponse.value.active);
        setDefaults((current) => ({ ...current, activeContext: contextResponse.value.active }));
      }
    } catch (err) {
      setConnection("disconnected");
      if (lastConnectionRef.current === "connected") {
        void notifySystem("PI Sandbox disconnected", "The daemon connection dropped.");
      }
      lastConnectionRef.current = "disconnected";
      setHealth("offline");
      setSessions([]);
      const msg = err instanceof Error ? err.message : "Unable to connect to daemon";
      setError(msg);
      setLastError(msg);
      recordGuiLog("error", `Connection failed for ${daemonUrl}: ${msg}`);
    } finally {
      refreshInFlightRef.current = false;
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
    if (!runtimeInfo?.available?.length) return;
    const correctedMode = bestRuntimeMode(runtimeInfo, defaults.mode);
    if (correctedMode !== defaults.mode) {
      setDefaults((current) => ({ ...current, mode: correctedMode }));
      recordGuiLog("info", `Adjusted unavailable runtime default ${defaults.mode} -> ${correctedMode}`);
    }
  }, [runtimeInfo, defaults.mode]);

  useEffect(() => {
    void readNotificationPermission().then(setNotificationPermission);
  }, []);

  useEffect(() => {
    if (!isTauriRuntime()) return;
    let unlisten: (() => void) | undefined;
    void listen<string>("pi://navigate", (event) => {
      if (event.payload) {
        setActiveView(event.payload);
      }
    }).then((stop) => {
      unlisten = stop;
    });
    return () => {
      unlisten?.();
    };
  }, []);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(timer);
  }, [refresh]);

  async function createSession(draft: CreateSandboxDraft) {
    const workspaceSource = draft.workspaceSource.trim();
    const folderIsAllowed = workspaceSource ? allowedFolders.includes(workspaceSource) : true;
    if (!folderIsAllowed && workspaceSource) {
      setError("Authorize the project folder before creating a GUI-launched session.");
      return;
    }
    setIsBusy(true);
    setError("");
    try {
      const workspace = workspaceSource
        ? {
            mode: draft.workspaceMode,
            source: workspaceSource,
            maxSize: draft.workspaceMaxSize.trim() || "5Gi"
          }
        : undefined;
      const created = await client.createSandbox({
        name: draft.name.trim() || "gui-session",
        template: draft.template,
        mode: draft.mode,
        workspace,
        ttlSeconds: draft.ttlSeconds
      });
      if (draft.repoUrl.trim()) {
        await client.cloneSandbox(created.id, draft.repoUrl.trim());
      }
      if (draft.rememberDefaults) {
        setDefaults((current) => ({
          ...current,
          template: draft.template,
          mode: draft.mode,
          network: draft.network
        }));
      }
      setSelectedId(created.id);
      setActiveView("sessions");
      setCreateDialogOpen(false);
      recordGuiLog("info", `Created sandbox ${created.id} (${created.name || draft.name})`);
      void notifySystem("Sandbox created", `${created.name || draft.name} is ready to inspect.`);
      await refresh();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Unable to create sandbox";
      setError(msg);
      recordGuiLog("error", `Create sandbox failed: ${msg}`);
    } finally {
      setIsBusy(false);
    }
  }

  function authorizeFolder(folderInput: string) {
    const folder = folderInput.trim();
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
    setIsBusy(true);
    setError("");
    try {
      const response = await client.useContext(name);
      const selected = contexts.find((context) => context.name === response.active);
      setActiveContext(response.active);
      setDefaults((current) => ({ ...current, activeContext: response.active }));
      if (selected?.name === "local") {
        setDaemonUrl(DEFAULT_DAEMON_URL);
      }
      setDaemonBearerToken("");
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
      void notifySystem("Support bundle exported", "Diagnostics were collected and redacted.");
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
        <Topbar
          activeView={activeView}
          connection={connection}
          health={health}
          daemonUrl={daemonUrl}
          isBusy={isBusy}
          onCreateSession={() => setCreateDialogOpen(true)}
        />

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
          />
        )}

        {/* ── Sessions view ──────────────────────────────────────────────── */}
        {activeView === "sessions" && (
          <div className="sessions-workbench">
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
            client={client}
            onRefresh={refresh}
          />
        )}

        {/* ── Policies view ──────────────────────────────────────────────── */}
        {activeView === "policies" && (
          <PoliciesView
            defaults={defaults}
            allowedFolders={allowedFolders}
            doctorResult={doctorResult}
            systemStatus={systemStatus}
            runtimeInfo={runtimeInfo}
            connection={connection}
            activeContext={activeContext}
            sessions={sessions}
            onOpenSettings={() => setActiveView("settings")}
            onOpenSessions={() => setActiveView("sessions")}
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

        <CreateSandboxDialog
          open={createDialogOpen}
          defaults={defaults}
          allowedFolders={allowedFolders}
          connection={connection}
          runtimeInfo={runtimeInfo}
          isBusy={isBusy}
          error={error}
          onAuthorizeFolder={authorizeFolder}
          onRemoveAllowedFolder={removeAllowedFolder}
          onCancel={() => setCreateDialogOpen(false)}
          onCreate={(draft) => void createSession(draft)}
        />
      </section>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
