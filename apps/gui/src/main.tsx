import React from "react";
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
  MonitorPlay,
  Network,
  Play,
  Plus,
  RotateCcw,
  Settings,
  ShieldCheck,
  SquareTerminal,
  Wrench
} from "lucide-react";
import "./styles.css";

type SessionState = "warm" | "running" | "idle";

type Session = {
  id: string;
  name: string;
  template: string;
  mode: string;
  workspace: string;
  state: SessionState;
  updated: string;
};

const sessions: Session[] = [
  {
    id: "sbx-41c8",
    name: "sdk streaming polish",
    template: "node-python",
    mode: "fast",
    workspace: "~/Projects/pi-sandboxes",
    state: "running",
    updated: "2 min ago"
  },
  {
    id: "sbx-229a",
    name: "microvm guest smoke",
    template: "rust",
    mode: "microvm",
    workspace: "~/work/pi-runtime",
    state: "warm",
    updated: "18 min ago"
  },
  {
    id: "sbx-804d",
    name: "remote daemon auth checks",
    template: "go",
    mode: "compat",
    workspace: "ssh://lab-box.local",
    state: "idle",
    updated: "Yesterday"
  }
];

const navItems = [
  { label: "Dashboard", icon: Command, active: true },
  { label: "Sessions", icon: MonitorPlay },
  { label: "Templates", icon: Layers3 },
  { label: "Contexts", icon: Network },
  { label: "Policies", icon: ShieldCheck },
  { label: "Settings", icon: Settings }
];

function stateLabel(state: SessionState) {
  if (state === "running") return "running";
  if (state === "warm") return "warm";
  return "idle";
}

function App() {
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
            <span>Local daemon</span>
          </div>
          <strong>
            <CircleDot size={13} />
            Connected
          </strong>
          <code>unix://~/.pi/sandboxd.sock</code>
          <button className="danger-action">Disconnect</button>
        </section>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <h1>Dashboard</h1>
            <p>Connected · v0.1.0 · local · restricted network</p>
          </div>
          <div className="topbar-actions">
            <button className="icon-button" aria-label="Open policy status" title="Policy status">
              <ShieldCheck size={20} />
            </button>
            <button className="primary-action">
              <Play size={20} />
              New sandbox
            </button>
          </div>
        </header>

        <div className="content-grid">
          <section className="hero-panel">
            <div>
              <span className="eyebrow">Ready workspace</span>
              <h2>What should this sandbox work on?</h2>
              <p>Create a warm isolated session, run commands, inspect diffs, and export artifacts without touching your host by default.</p>
            </div>
            <button className="primary-action large">
              <Plus size={21} />
              Create session
            </button>
          </section>

          <section className="onboarding-panel">
            <div className="section-heading">
              <h3>Start workbench</h3>
              <span>Onboarding</span>
            </div>
            <button className="choice-row selected">
              <HardDrive size={22} />
              <span>
                <strong>Use local daemon</strong>
                <small>Best for this workstation</small>
              </span>
              <ChevronRight size={20} />
            </button>
            <button className="choice-row">
              <Network size={22} />
              <span>
                <strong>Connect remote context</strong>
                <small>SSH, Tailscale, or WireGuard daemon</small>
              </span>
              <ChevronRight size={20} />
            </button>
          </section>

          <section className="authorization-panel">
            <div className="section-heading">
              <h3>Workspace authorization</h3>
              <span>Default copy mode</span>
            </div>
            <label className="field-label" htmlFor="project-folder">Project folder</label>
            <div className="folder-picker">
              <div className="folder-value" id="project-folder">
                <Folder size={19} />
                <span>/Users/svezina/Projects/playground-perso/pi-sandboxes</span>
              </div>
              <button>Browse</button>
            </div>
            <div className="mode-grid" aria-label="Workspace mode">
              <button className="mode-option active">copy</button>
              <button className="mode-option">overlay</button>
              <button className="mode-option">bind</button>
            </div>
          </section>

          <section className="sessions-panel">
            <div className="section-heading">
              <h3>Active sessions</h3>
              <button>View all</button>
            </div>
            <div className="session-list">
              {sessions.map((session) => (
                <article className="session-row" key={session.id}>
                  <div className="session-icon">
                    <SquareTerminal size={20} />
                  </div>
                  <div className="session-main">
                    <strong>{session.name}</strong>
                    <span>
                      {session.id} · {session.template} · {session.mode} · {session.updated}
                    </span>
                  </div>
                  <div className={`state-pill ${session.state}`}>
                    <CircleDot size={12} />
                    {stateLabel(session.state)}
                  </div>
                  <ChevronRight size={19} />
                </article>
              ))}
            </div>
          </section>

          <section className="operations-panel">
            <div className="section-heading">
              <h3>Session operations</h3>
              <span>sbx-41c8</span>
            </div>
            <div className="operation-grid">
              <button>
                <Play size={19} />
                Exec
              </button>
              <button>
                <History size={19} />
                History
              </button>
              <button>
                <ListChecks size={19} />
                Diff
              </button>
              <button>
                <Database size={19} />
                Artifacts
              </button>
              <button>
                <RotateCcw size={19} />
                Snapshot
              </button>
              <button>
                <Clock3 size={19} />
                Logs
              </button>
            </div>
          </section>

          <section className="diagnostics-panel">
            <div className="section-heading">
              <h3>Diagnostics</h3>
              <span>Doctor</span>
            </div>
            <div className="metric-row">
              <Activity size={18} />
              <span>fast, compat, secure available</span>
              <strong>3/5</strong>
            </div>
            <div className="metric-row">
              <KeyRound size={18} />
              <span>remote auth</span>
              <strong>ok</strong>
            </div>
            <div className="metric-row">
              <Wrench size={18} />
              <span>support bundle</span>
              <button>Export</button>
            </div>
          </section>
        </div>
      </section>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
