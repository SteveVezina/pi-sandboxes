import React, { useCallback, useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  Activity,
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
  Wrench
} from "lucide-react";
import { ConnectionState, ExecResult, PiDaemonClient, SandboxInfo } from "./api";
import "./styles.css";

const DEFAULT_DAEMON_URL = "http://127.0.0.1:7777";
const DAEMON_URL_STORAGE_KEY = "pi.gui.daemonUrl.v2";
const ALLOWED_FOLDERS_STORAGE_KEY = "pi.gui.allowedFolders.v1";
const templates = ["base", "node-python", "go", "rust", "polyglot"];
const modes = ["fast", "compat", "secure", "microvm"];

const navItems = [
  { label: "Dashboard", icon: Command, active: true },
  { label: "Sessions", icon: MonitorPlay },
  { label: "Templates", icon: Layers3 },
  { label: "Contexts", icon: Network },
  { label: "Policies", icon: ShieldCheck },
  { label: "Settings", icon: Settings }
];

function App() {
  const [daemonUrl, setDaemonUrl] = useState(() => localStorage.getItem(DAEMON_URL_STORAGE_KEY) || DEFAULT_DAEMON_URL);
  const [connection, setConnection] = useState<ConnectionState>("checking");
  const [health, setHealth] = useState<string>("checking");
  const [sessions, setSessions] = useState<SandboxInfo[]>([]);
  const [selectedId, setSelectedId] = useState<string>("");
  const [error, setError] = useState<string>("");
  const [isBusy, setIsBusy] = useState(false);
  const [newName, setNewName] = useState("gui-session");
  const [newTemplate, setNewTemplate] = useState("node-python");
  const [newMode, setNewMode] = useState("fast");
  const [projectFolder, setProjectFolder] = useState("/Users/svezina/Projects/playground-perso/pi-sandboxes");
  const [allowedFolders, setAllowedFolders] = useState<string[]>(() => {
    try {
      return JSON.parse(localStorage.getItem(ALLOWED_FOLDERS_STORAGE_KEY) || "[]") as string[];
    } catch {
      return [];
    }
  });
  const [workspaceMode, setWorkspaceMode] = useState<"copy" | "overlay" | "bind">("copy");
  const [command, setCommand] = useState("pwd");
  const [execResult, setExecResult] = useState<ExecResult | null>(null);
  const [operationOutput, setOperationOutput] = useState<string>("");

  const client = useMemo(() => new PiDaemonClient(daemonUrl), [daemonUrl]);
  const selectedSession = sessions.find((session) => session.id === selectedId) || sessions[0];
  const folderIsAllowed = allowedFolders.includes(projectFolder);

  const refresh = useCallback(async () => {
    setError("");
    try {
      setConnection("checking");
      const daemonHealth = await client.health();
      const sandboxList = await client.listSandboxes();
      setHealth(daemonHealth.status || "ok");
      setSessions(sandboxList);
      setConnection("connected");
      setSelectedId((current) => current || sandboxList[0]?.id || "");
    } catch (err) {
      setConnection("disconnected");
      setHealth("offline");
      setSessions([]);
      setError(err instanceof Error ? err.message : "Unable to connect to daemon");
    }
  }, [client]);

  useEffect(() => {
    localStorage.setItem(DAEMON_URL_STORAGE_KEY, daemonUrl);
  }, [daemonUrl]);

  useEffect(() => {
    localStorage.setItem(ALLOWED_FOLDERS_STORAGE_KEY, JSON.stringify(allowedFolders));
  }, [allowedFolders]);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(timer);
  }, [refresh]);

  async function createSession() {
    if (!folderIsAllowed) {
      setError("Authorize the project folder before creating a GUI-launched session.");
      return;
    }
    setIsBusy(true);
    setError("");
    try {
      const created = await client.createSandbox({
        name: newName.trim() || "gui-session",
        template: newTemplate,
        mode: newMode,
        workspace: {
          mode: workspaceMode,
          source: projectFolder,
          maxSize: "5Gi"
        }
      });
      setSelectedId(created.id);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to create sandbox");
    } finally {
      setIsBusy(false);
    }
  }

  function authorizeFolder() {
    const folder = projectFolder.trim();
    if (!folder) {
      setError("Enter a project folder before authorizing it.");
      return;
    }
    setAllowedFolders((current) => current.includes(folder) ? current : [...current, folder]);
    setError("");
  }

  function removeAllowedFolder(folder: string) {
    setAllowedFolders((current) => current.filter((item) => item !== folder));
  }

  async function destroySelected() {
    if (!selectedSession) return;
    setIsBusy(true);
    setError("");
    try {
      await client.destroySandbox(selectedSession.id);
      setSelectedId("");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to destroy sandbox");
    } finally {
      setIsBusy(false);
    }
  }

  async function runCommand() {
    if (!selectedSession || !command.trim()) return;
    setIsBusy(true);
    setError("");
    setExecResult(null);
    setOperationOutput("");
    try {
      const result = await client.exec(selectedSession.id, command.trim());
      setExecResult(result);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to run command");
    } finally {
      setIsBusy(false);
    }
  }

  async function runOperation(label: string, operation: (id: string) => Promise<unknown>) {
    if (!selectedSession) return;
    setIsBusy(true);
    setError("");
    setExecResult(null);
    try {
      const result = await operation(selectedSession.id);
      setOperationOutput(`${label}\n${formatUnknown(result)}`);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : `${label} failed`);
    } finally {
      setIsBusy(false);
    }
  }

  return (
    <main className="app-shell">
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
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <button className={item.active ? "nav-item active" : "nav-item"} key={item.label}>
                <Icon size={19} />
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>

        <section className="engine-card" aria-label="Daemon connection">
          <div className="engine-topline">
            <Gauge size={17} />
            <span>Local HTTP daemon</span>
          </div>
          <strong className={connection === "connected" ? "connected-text" : "offline-text"}>
            {connection === "checking" ? <Loader2 className="spin" size={15} /> : <CircleDot size={13} />}
            {connectionLabel(connection)}
          </strong>
          <input
            aria-label="Daemon URL"
            className="daemon-input"
            value={daemonUrl}
            onChange={(event) => setDaemonUrl(event.target.value)}
          />
          <button className="secondary-action" onClick={() => void refresh()}>
            Re-check
          </button>
        </section>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <h1>Dashboard</h1>
            <p>{connectionLabel(connection)} · daemon {health} · {daemonUrl}</p>
          </div>
          <div className="topbar-actions">
            <button className="icon-button" aria-label="Open policy status" title="Policy status">
              <ShieldCheck size={20} />
            </button>
            <button className="primary-action" onClick={() => void createSession()} disabled={isBusy || connection !== "connected" || !folderIsAllowed}>
              {isBusy ? <Loader2 className="spin" size={20} /> : <Play size={20} />}
              New sandbox
            </button>
          </div>
        </header>

        {error ? <div className="error-banner">{error}</div> : null}

        <div className="content-grid">
          <section className="hero-panel">
            <div>
              <span className="eyebrow">Live daemon workbench</span>
              <h2>What should this sandbox work on?</h2>
              <p>Create a warm isolated session, run commands, inspect diffs, and export artifacts through the real daemon API.</p>
            </div>
            <button className="primary-action large" onClick={() => void createSession()} disabled={isBusy || connection !== "connected" || !folderIsAllowed}>
              <Plus size={21} />
              Create session
            </button>
          </section>

          <section className="onboarding-panel">
            <div className="section-heading">
              <h3>Start workbench</h3>
              <span>Connection</span>
            </div>
            <button className="choice-row selected" onClick={() => setDaemonUrl(DEFAULT_DAEMON_URL)}>
              <HardDrive size={22} />
              <span>
                <strong>Use local HTTP daemon</strong>
                <small>Start with `pi-sandboxd --http-port 7777`</small>
              </span>
              <ChevronRight size={20} />
            </button>
            <button className="choice-row" title="Remote context selection will use F22/F23 context APIs in the next slice">
              <Network size={22} />
              <span>
                <strong>Connect remote context</strong>
                <small>Uses authenticated HTTP endpoint when configured</small>
              </span>
              <ChevronRight size={20} />
            </button>
          </section>

          <section className="authorization-panel">
            <div className="section-heading">
              <h3>Workspace authorization</h3>
              <span>Default {workspaceMode} mode</span>
            </div>
            <label className="field-label" htmlFor="project-folder">Project folder</label>
            <div className="folder-picker">
              <div className="folder-value" id="project-folder">
                <Folder size={19} />
                <input value={projectFolder} onChange={(event) => setProjectFolder(event.target.value)} />
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
              {allowedFolders.length === 0 ? (
                <span>No folders authorized.</span>
              ) : allowedFolders.map((folder) => (
                <div className="allowed-folder" key={folder}>
                  <span>{folder}</span>
                  <button onClick={() => removeAllowedFolder(folder)}>Remove</button>
                </div>
              ))}
            </div>
          </section>

          <section className="create-panel">
            <div className="section-heading">
              <h3>Create sandbox</h3>
              <span>POST /v1/sandboxes</span>
            </div>
            <label className="field-label" htmlFor="sandbox-name">Name</label>
            <input id="sandbox-name" value={newName} onChange={(event) => setNewName(event.target.value)} />
            <div className="split-fields">
              <label>
                Template
                <select value={newTemplate} onChange={(event) => setNewTemplate(event.target.value)}>
                  {templates.map((template) => <option key={template}>{template}</option>)}
                </select>
              </label>
              <label>
                Mode
                <select value={newMode} onChange={(event) => setNewMode(event.target.value)}>
                  {modes.map((mode) => <option key={mode}>{mode}</option>)}
                </select>
              </label>
            </div>
          </section>

          <section className="sessions-panel">
            <div className="section-heading">
              <h3>Active sessions</h3>
              <button onClick={() => void refresh()}>Refresh</button>
            </div>
            <div className="session-list">
              {sessions.length === 0 ? (
                <div className="empty-state">No sessions returned by the daemon.</div>
              ) : sessions.map((session) => (
                <button
                  className={selectedSession?.id === session.id ? "session-row selected" : "session-row"}
                  key={session.id}
                  onClick={() => setSelectedId(session.id)}
                >
                  <div className="session-icon">
                    <SquareTerminal size={20} />
                  </div>
                  <div className="session-main">
                    <strong>{session.name}</strong>
                    <span>
                      {session.id} · {session.template || "template"} · {session.mode || "mode"} · {formatTime(session.last_used || session.updated_at || session.created_at)}
                    </span>
                    <span>{session.workspace_mode || "copy"} · {session.workspace || "daemon workspace"}</span>
                  </div>
                  <div className={`state-pill ${session.state}`}>
                    <CircleDot size={12} />
                    {session.state}
                  </div>
                  <ChevronRight size={19} />
                </button>
              ))}
            </div>
          </section>

          <section className="operations-panel">
            <div className="section-heading">
              <h3>Session operations</h3>
              <span>{selectedSession?.id || "none"}</span>
            </div>
            <div className="command-row">
              <input value={command} onChange={(event) => setCommand(event.target.value)} aria-label="Command" />
              <button onClick={() => void runCommand()} disabled={!selectedSession || isBusy}>
                <Play size={18} />
                Exec
              </button>
            </div>
            <div className="operation-grid">
              <button onClick={() => void runOperation("History", (id) => client.logs(id))} disabled={!selectedSession || isBusy}>
                <History size={19} />
                History
              </button>
              <button onClick={() => void runOperation("Diff", (id) => client.diff(id))} disabled={!selectedSession || isBusy}>
                <ListChecks size={19} />
                Diff
              </button>
              <button onClick={() => void runOperation("Artifacts", (id) => client.artifacts(id))} disabled={!selectedSession || isBusy}>
                <Database size={19} />
                Artifacts
              </button>
              <button onClick={() => void runOperation("Snapshots", (id) => client.snapshots(id))} disabled={!selectedSession || isBusy}>
                <RotateCcw size={19} />
                Snapshot
              </button>
              <button onClick={() => void runOperation("Logs", (id) => client.logs(id))} disabled={!selectedSession || isBusy}>
                <Clock3 size={19} />
                Logs
              </button>
              <button onClick={() => void destroySelected()} disabled={!selectedSession || isBusy}>
                <Trash2 size={19} />
                Destroy
              </button>
            </div>
            {execResult ? (
              <pre className="terminal-output">{formatExec(execResult)}</pre>
            ) : null}
            {operationOutput ? (
              <pre className="terminal-output">{operationOutput}</pre>
            ) : null}
          </section>

          <section className="diagnostics-panel">
            <div className="section-heading">
              <h3>Diagnostics</h3>
              <span>Live</span>
            </div>
            <div className="metric-row">
              <Activity size={18} />
              <span>daemon health</span>
              <strong>{health}</strong>
            </div>
            <div className="metric-row">
              <KeyRound size={18} />
              <span>connection</span>
              <strong>{connectionLabel(connection)}</strong>
            </div>
            <div className="metric-row">
              <Wrench size={18} />
              <span>support bundle</span>
              <button disabled>Export</button>
            </div>
          </section>
        </div>
      </section>
    </main>
  );
}

function connectionLabel(state: ConnectionState) {
  if (state === "connected") return "Connected";
  if (state === "checking") return "Checking";
  return "Disconnected";
}

function formatTime(value?: string) {
  if (!value) return "no activity";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function formatExec(result: ExecResult) {
  const exitCode = result.exitCode ?? result.exit_code ?? "unknown";
  const duration = result.durationMs ?? result.duration_ms ?? "unknown";
  const timedOut = result.timedOut ?? result.timed_out ?? false;
  return [
    `$ exit=${exitCode} duration=${duration}ms timed_out=${timedOut} truncated=${Boolean(result.truncated)}`,
    result.stdout || "",
    result.stderr ? `stderr:\n${result.stderr}` : ""
  ].filter(Boolean).join("\n");
}

function formatUnknown(value: unknown) {
  if (typeof value === "string") return value;
  return JSON.stringify(value, null, 2);
}

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
