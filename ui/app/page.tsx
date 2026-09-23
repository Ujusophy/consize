"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import {
  Activity,
  AlertTriangle,
  ArrowRight,
  Bell,
  Calendar,
  Check,
  CheckCircle2,
  ChevronDown,
  Cloud,
  Database,
  Download,
  Eye,
  FileText,
  Folder,
  Gauge,
  Grid2X2,
  Link,
  ListChecks,
  Lock,
  Network,
  Play,
  RotateCcw,
  Search,
  Server,
  Settings,
  Shield,
  Sparkles,
  Zap
} from "lucide-react";
import { ThemeToggle } from "./theme-toggle";

const API_BASE = process.env.NEXT_PUBLIC_CONSIZE_API_BASE_URL ?? "http://127.0.0.1:8080";

type ApiResource = {
  id: string;
  type: string;
  provider: string;
  name: string;
  environment: string;
  owner: string;
  region: string;
  criticality: string;
  metadata?: Record<string, unknown>;
};

type ApiRecommendation = {
  id: number;
  resource_id: string;
  plugin_id: string;
  algorithm_id?: string;
  action_type: string;
  title: string;
  summary: string;
  estimated_savings_monthly: number;
  confidence: string;
  risk: string;
  evidence?: string[];
  policy_id: string;
  status: string;
  current?: Record<string, unknown>;
  proposed?: Record<string, unknown>;
  parameters?: { confidence_assessment?: { profile: string; required_history: string; observed_history: string; coverage_ratio: number; reasons: string[] }; verification_plan?: { checks?: string[]; metrics_plugin_id?: string } };
};

type ApiPluginStatus = {
  manifest: {
    id: string;
    display_name: string;
    category: string;
    can_mutate_infrastructure: boolean;
    requires_approval: boolean;
  };
  health: {
    status: string;
    message: string;
  };
};

type ApiAction = {
  plan?: { preflight?: ApiPreflightCheck[] };
  id: number;
  recommendation_id?: number;
  resource_id: string;
  plugin_id: string;
  mode: string;
  result: string;
  message: string;
  created_at: string;
  policy_decision?: { policy_id: string; decision: string; reasons: string[] };
};

type ApiPreflightCheck = { id: string; status: string; message: string; details?: Record<string, unknown> };

type ApiDashboard = {
  verification?: { checks?: Array<{ signal: string; statistic: string; mode: string; threshold: number }> };
  jobs?: Array<{ id: number; state: string; plan?: { preflight?: ApiPreflightCheck[] }; next_run: string; window_start: string; deadline: string; last_error?: string; verification: { rollback_on_failure: boolean; rollback_on_timeout: boolean; checks?: Array<{ signal: string; statistic: string; mode: string; threshold: number }> }; verification_result?: { status: string; reasons: string[] } }>;
  stats: {
    projected_savings_monthly: number;
    verified_savings_monthly: number;
    pending_approvals: number;
    rollback_rate: number;
    open_recommendations: number;
    executed_actions: number;
  };
  resources: ApiResource[];
  recommendations: ApiRecommendation[];
  plugins: ApiPluginStatus[];
  actions: ApiAction[];
};

const nav = [
  { label: "Overview", icon: Grid2X2, active: true },
  { label: "Recommendations", icon: Sparkles },
  { label: "Resources", icon: Server },
  { label: "Policies", icon: Shield },
  { label: "Plugins", icon: Network },
  { label: "Approvals", icon: CheckCircle2, badge: "3" },
  { label: "Audit", icon: FileText },
  { label: "Reports", icon: Activity }
];

const metrics = [
  {
    title: "Projected Savings",
    value: "$148,250",
    suffix: "/ mo",
    icon: Activity,
    foot: ["+14.2%", "vs last month", "84 open recs"]
  },
  {
    title: "Verified Savings",
    value: "$92,400",
    suffix: "/ mo",
    icon: Shield,
    foot: ["Post-action Data...", "34 verified"]
  },
  {
    title: "Pending Approvals",
    value: "7",
    suffix: "Actions",
    warning: "($28.1k/mo)",
    icon: AlertTriangle,
    foot: ["Requires SRE approval", "prod-guard"],
    tone: "warn"
  },
  {
    title: "Rollback Rate",
    value: "0.0%",
    icon: RotateCcw,
    foot: ["142 GitOps/auto ...", "0 rollbacks"]
  }
];

const plugins = [
  ["Datadog", "42 signals", "Live", "green"],
  ["Kubecost", "$382k/mo", "Synced", "green"],
  ["K8s Direct", "v1.28", "Ready", "green"],
  ["GitHub GitOps", "infra-1", "PR mode", "blue"],
  ["Slack", "#finops-alerts", "Active", "green"]
];

const recommendations = [
  {
    active: true,
    icon: Server,
    color: "cyan",
    name: "payment-service-deployment",
    meta: "K8s · prod-us-east-1 · c6i.2xlarge",
    owner: "Platform",
    policy: "prod-conservative",
    risk: "Low",
    riskTone: "low",
    savings: "$4,200",
    action: "Create PR",
    actionIcon: ArrowRight
  },
  {
    icon: Database,
    color: "gold",
    name: "orders-aurora-cluster",
    meta: "RDS · db.r6g.4xl→2xl",
    owner: "Checkout",
    policy: "db-safe-change",
    risk: "Medium",
    riskTone: "medium",
    savings: "$12,800",
    action: "Schedule",
    actionIcon: Gauge
  },
  {
    icon: Folder,
    color: "violet",
    name: "analytics-raw-events",
    meta: "S3 · Glacier IR",
    owner: "Data Eng",
    policy: "lifecycle",
    risk: "Low",
    riskTone: "low",
    savings: "$1,950",
    action: "Apply",
    actionIcon: Check
  },
  {
    icon: Grid2X2,
    color: "red",
    name: "ml-infer-cluster",
    meta: "EKS · g5.4xl autoscale",
    owner: "ML Ops",
    policy: "gpu-strict",
    risk: "High",
    riskTone: "high",
    savings: "$8,400",
    action: "Required",
    actionIcon: Lock
  },
  {
    icon: Server,
    color: "green",
    name: "staging-worker-pool",
    meta: "K8s · Non-Prod Limits",
    owner: "Infra Core",
    policy: "non-prod-auto",
    risk: "Low",
    riskTone: "low",
    savings: "$2,640",
    action: "Direct Apply",
    actionIcon: Zap,
    primary: true
  }
];

const audit = [
  {
    icon: Check,
    tone: "ok",
    title: "Optimization applied: staging-worker-pool scaled down by 4 nodes",
    line: "-$1,800/mo impact · Actor: auto-remediate-daemon · Rule: non-prod-auto",
    time: "12m ago"
  },
  {
    icon: ArrowRight,
    tone: "flow",
    title: "GitOps PR generated: #1402 orders-aurora-cluster instance right-sizing",
    line: "Branch: finops/aurora-2xlarge-downscale · Actor: FinOps Bot · Awaiting SRE signoff",
    time: "1h ago"
  },
  {
    icon: Check,
    tone: "ok",
    title: "Verification window passed: checkout-gateway latency & CPU headroom preserved",
    line: "Datadog P99: 42ms (threshold: <65ms) · Zero errors recorded over 30m verification cycle",
    time: "3h ago"
  },
  {
    icon: AlertTriangle,
    tone: "deny",
    title: "Safety Policy Blocked: Direct apply on payment-db denied",
    line: "Violation: Direct mutation disallowed for tier-1 prod database · Guard: production-conservative",
    time: "5h ago"
  }
];

const checks = [
  ["Headroom Safe Buffer", "Retains >30% buffer over peak 14-day P95 load", "Passed", "ok"],
  ["Step-down Throttling Limit", "Max 25% drop per cycle (Requested: 15%)", "Passed", "ok"],
  ["Approval Requirement", "Prod tag triggers required GitOps peer review", "Required", "warn"],
  ["Automatic Rollback Guard", "Triggers rollback if HTTP 5xx error >0.05% in 15m", "Armed", "ok"]
];

function currency(value: number) {
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(value);
}

function titleCase(value: string) {
  return value
    .replace(/[-_.]/g, " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function formatChange(value: unknown, key: string, resource: unknown) {
  if (value === undefined || value === null) return "Unavailable";
  if (resource === "memory" && ["request", "limit"].includes(key) && typeof value === "number") {
    if (value === 0 && key === "limit") return "No limit";
    return `${(value / 1048576).toLocaleString(undefined, { maximumFractionDigits: 2 })} MiB`;
  }
  return String(value);
}

function metadataValue(resource: ApiResource | undefined, key: string) {
  const value = resource?.metadata?.[key];
  return typeof value === "string" ? value : "";
}

function resourceIcon(type: string) {
  if (type.includes("rds") || type.includes("database")) return Database;
  if (type.includes("storage") || type.includes("bucket")) return Folder;
  if (type.includes("kubernetes")) return Server;
  return Grid2X2;
}

function riskTone(risk: string) {
  const normalized = risk.toLowerCase();
  if (normalized.includes("high")) return "high";
  if (normalized.includes("medium")) return "medium";
  return "low";
}

function timeAgo(raw: string) {
  if (!raw) return "just now";
  const then = new Date(raw).getTime();
  const seconds = Math.max(1, Math.floor((Date.now() - then) / 1000));
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export default function Page() {
  const applyDialog = useRef<HTMLDialogElement>(null);
  const [confirmApply, setConfirmApply] = useState<ApiRecommendation | null>(null);
  const [selectedID, setSelectedID] = useState<number | null>(null);
  const [dashboard, setDashboard] = useState<ApiDashboard | null>(null);
  const [apiError, setApiError] = useState("");
  const [actionError, setActionError] = useState("");
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [openMenu, setOpenMenu] = useState<"cloud" | "date" | null>(null);
  const [cloudScope, setCloudScope] = useState("us-east-1 / multi-cloud");
  const [dateRange, setDateRange] = useState("Last 30 days");
  useEffect(() => { if (confirmApply && applyDialog.current && !applyDialog.current.open) applyDialog.current.showModal(); }, [confirmApply]);

  async function loadDashboard() {
    try {
      const response = await fetch(`${API_BASE}/api/dashboard`, { cache: "no-store" });
      if (!response.ok) throw new Error(await response.text());
      setDashboard(await response.json());
      setApiError("");
    } catch (error) {
      setApiError(error instanceof Error ? error.message : "API unavailable");
    }
  }

  useEffect(() => {
    void loadDashboard();
    const timer = window.setInterval(() => void loadDashboard(), 15000);
    return () => window.clearInterval(timer);
  }, []);

  async function runAction(rec: ApiRecommendation, operation: "plan" | "execute" | "recover") {
    setActionError("");
    setBusyAction(`${rec.id}:${operation}`);
    try {
      const response = await fetch(`${API_BASE}/api/recommendations/${rec.id}/${operation}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "ui@consize.local", mode: operation === "plan" ? "dry_run" : "approved" })
      });
      if (!response.ok) { const error = await response.json(); throw new Error(error.error || "Action rejected"); }
      await loadDashboard();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Action failed");
    } finally {
      setBusyAction(null);
    }
  }

  async function runPolicyEvaluation() {
    setActionError("");
    const resource = dashboard?.resources[0];
    if (!resource) {
      setActionError("No resource is registered yet. Register a resource before generating a recommendation.");
      return;
    }
    setBusyAction("policy:evaluate");
    try {
      const response = await fetch(`${API_BASE}/api/recommendations/generate?resource_id=${encodeURIComponent(resource.id)}`, {
        method: "POST"
      });
      if (!response.ok) throw new Error(await response.text());
      await loadDashboard();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : "Recommendation generation failed");
    } finally {
      setBusyAction(null);
    }
  }

  function exportSummary() {
    const payload = JSON.stringify(dashboard ?? { status: "api_unavailable", fallback: true }, null, 2);
    const blob = new Blob([payload], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "consize-dashboard-summary.json";
    anchor.click();
    URL.revokeObjectURL(url);
  }

  function requireBackend(action: string) {
    setApiError(`${action} needs the Consize API. Start it with: go run ./cmd/consize serve -addr 127.0.0.1:8080`);
  }

  const resourceByID = useMemo(() => {
    const index = new Map<string, ApiResource>();
    dashboard?.resources.forEach((resource) => index.set(resource.id, resource));
    return index;
  }, [dashboard]);

  const liveMetrics = dashboard
    ? [
        {
          title: "Projected Savings",
          value: `$${currency(dashboard.stats.projected_savings_monthly)}`,
          suffix: "/ mo",
          icon: Activity,
          foot: ["Open recommendations", `${dashboard.stats.open_recommendations} active`]
        },
        {
          title: "Verified Savings",
          value: `$${currency(dashboard.stats.verified_savings_monthly)}`,
          suffix: "/ mo",
          icon: Shield,
          foot: ["Executed actions", `${dashboard.stats.executed_actions} applied`]
        },
        {
          title: "Pending Approvals",
          value: `${dashboard.stats.pending_approvals}`,
          suffix: "Actions",
          warning: "",
          icon: AlertTriangle,
          foot: ["Policy gated", "approval path"],
          tone: "warn"
        },
        {
          title: "Rollback Rate",
          value: `${dashboard.stats.rollback_rate.toFixed(1)}%`,
          icon: RotateCcw,
          foot: ["Post-action checks", "metrics-based"]
        }
      ]
    : metrics.map((metric) => ({ ...metric, value: "—", warning: "", foot: ["API unavailable", "Waiting for data"] }));

  const livePlugins = dashboard
    ? dashboard.plugins.map((item) => [
        item.manifest.display_name || item.manifest.id,
        item.health.message || titleCase(item.manifest.category),
        item.health.status === "healthy" ? "Healthy" : titleCase(item.health.status),
        item.health.status === "healthy" ? "green" : "blue"
      ])
    : [];

  const liveRecommendations = dashboard
    ? dashboard.recommendations.map((rec, index) => {
        const res = resourceByID.get(rec.resource_id);
        const tone = riskTone(rec.risk);
        const namespace = metadataValue(res, "namespace");
        const meta = [res?.provider || "resource", res?.environment, namespace || res?.region].filter(Boolean).join(" · ");
        return {
          id: rec.id,
          active: index === 0,
          icon: resourceIcon(res?.type ?? ""),
          color: res?.provider === "aws" ? "gold" : "cyan",
          name: res?.name || rec.title,
          meta,
          owner: titleCase(res?.owner || "unassigned"),
          policy: rec.policy_id || "default",
          risk: titleCase(rec.risk || "low"),
          riskTone: tone,
          savings: `$${currency(rec.estimated_savings_monthly)}`,
          action: rec.status === "planned" ? "Apply" : rec.status === "pending" ? "Review Change" : titleCase(rec.status),
          actionIcon: rec.status === "planned" ? Check : ArrowRight,
          primary: rec.status === "planned",
          raw: rec
        };
      })
    : [];

  const selected = liveRecommendations.find((rec) => "raw" in rec && rec.raw.id === selectedID) ?? liveRecommendations[0];
  const selectedRaw = selected && "raw" in selected ? selected.raw : undefined;
  const selectedResource = selectedRaw ? resourceByID.get(selectedRaw.resource_id) : undefined;
  const selectedJob = dashboard?.jobs?.find((job) => job.id === selectedRaw?.id);
  const preflightFor = (id: number): ApiPreflightCheck[] => dashboard?.jobs?.find((job) => job.id === id)?.plan?.preflight || dashboard?.actions.filter((action) => action.recommendation_id === id && action.plan?.preflight).sort((a, b) => b.id - a.id)[0]?.plan?.preflight || [];
  const selectedPreflight = selectedRaw ? preflightFor(selectedRaw.id) : [];
  const selectedPolicy = dashboard?.actions.filter((action) => action.recommendation_id === selectedRaw?.id && action.policy_decision).at(-1)?.policy_decision;
  const liveAudit = dashboard
    ? dashboard.actions.filter((item) => !["requested", "planned", "prepared", "applying"].includes(item.result)).slice(-6).reverse().map((item) => ({
        icon: item.result === "failed" ? AlertTriangle : item.result === "executed" ? Check : ArrowRight,
        tone: item.result === "failed" ? "deny" : item.result === "executed" ? "ok" : "flow",
        title: titleCase(`${item.result}: ${item.plugin_id}`),
        line: `${item.message} · Mode: ${item.mode} · Resource: ${item.resource_id}`,
        time: timeAgo(item.created_at)
      }))
    : [];

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brandMark">
            <img alt="Consize" src="/consize_dashboard_logo.png" />
          </div>
          <div>
            <div className="brandLine">
              <span>Consize</span>
              <b>0.3 alpha</b>
            </div>
            <small>Control Plane</small>
          </div>
        </div>

        <nav className="nav" aria-label="Primary navigation">
          <p>Operational Queue</p>
          {nav.map((item) => {
            const Icon = item.icon;
            return (
              <a
                aria-label={item.label}
                className={`navItem ${item.active ? "active" : ""}`}
                data-label={item.label}
                href="#"
                key={item.label}
                title={item.label}
              >
                <Icon size={18} />
                <span>{item.label}</span>
                {item.badge ? <b>{item.badge}</b> : null}
              </a>
            );
          })}
        </nav>

        <div className="sidebarFooter">
          <button className="workspace">
            <Database size={18} />
            <span>
              <small>Workspace</small>
              <strong>Acme Corp Infra</strong>
            </span>
            <ChevronDown size={16} />
          </button>
          <div className="profile">
            <div className="avatar">FO</div>
            <span>
              <strong>FinOps Lead / SRE</strong>
              <small>Active Guard</small>
            </span>
            <Settings size={17} />
          </div>
        </div>
      </aside>

      <section className="main">
        <header className="topbar">
          <label className="search">
            <Search size={18} />
            <input placeholder="Search resources, policies, PRs..." />
            <kbd>⌘K</kbd>
          </label>
          <div className="topbarControls">
            <button className={`health ${dashboard ? "" : "disconnected"}`}><span />{dashboard ? `${dashboard.plugins.length} plugins connected` : "API disconnected"}</button>
            <div className="menuWrap">
              <button
                aria-expanded={openMenu === "cloud"}
                className="select"
                onClick={() => setOpenMenu(openMenu === "cloud" ? null : "cloud")}
              >
                <Cloud size={17} /><span>{cloudScope}</span><ChevronDown size={16} />
              </button>
              {openMenu === "cloud" ? (
                <div className="dropdown">
                  {["us-east-1 / multi-cloud", "AWS production", "Kubernetes clusters", "All providers"].map((item) => (
                    <button key={item} onClick={() => { setCloudScope(item); setOpenMenu(null); }}>{item}</button>
                  ))}
                </div>
              ) : null}
            </div>
            <div className="menuWrap">
              <button
                aria-expanded={openMenu === "date"}
                className="select date"
                onClick={() => setOpenMenu(openMenu === "date" ? null : "date")}
              >
                <Calendar size={17} /><span>{dateRange}</span><ChevronDown size={16} />
              </button>
              {openMenu === "date" ? (
                <div className="dropdown right">
                  {["Last 24 hours", "Last 7 days", "Last 30 days", "This quarter"].map((item) => (
                    <button key={item} onClick={() => { setDateRange(item); setOpenMenu(null); }}>{item}</button>
                  ))}
                </div>
              ) : null}
            </div>
            <ThemeToggle />
            <button className="bell"><Bell size={18} /><i /></button>
            <button className="user">FL</button>
          </div>
        </header>

        <main className="content">
          <section className="hero">
            <div>
              <div className="titleLine">
                <h1>Optimization Overview</h1>
                <span><i />Policy Guard Active</span>
              </div>
              <p>Real-time governed resource optimization, policy evaluation, and post-action verification.</p>
            </div>
            <div className="actions">
              <button onClick={exportSummary}><Download size={17} />Export Summary</button>
              <button className="primary" disabled={busyAction === "policy:evaluate"} onClick={() => void runPolicyEvaluation()}>
                <Play size={17} />{busyAction === "policy:evaluate" ? "Evaluating" : "Run Policy Evaluation"}
              </button>
            </div>
          </section>

          {(actionError || apiError) && <div className="actionError" role="alert"><AlertTriangle size={18} /><span>{actionError || apiError}</span><button onClick={() => { setActionError(""); void loadDashboard(); }}>Dismiss</button></div>}
          <section className="metrics">
            {liveMetrics.map((metric) => {
              const Icon = metric.icon;
              return (
                <article className={`metric ${metric.tone ?? ""}`} key={metric.title}>
                  <div className="metricTop">
                    <span>{metric.title}</span>
                    <b><Icon size={18} /></b>
                  </div>
                  <h2>{metric.value}<small>{metric.suffix}</small>{"warning" in metric && metric.warning ? <em>{metric.warning}</em> : null}</h2>
                  <div className="metricFoot">
                    {metric.foot.map((item, index) => (
                      <span className={index === 0 || item.includes("verified") || item.includes("rollbacks") ? "accent" : ""} key={item}>{item}</span>
                    ))}
                  </div>
                </article>
              );
            })}
          </section>

          <section className="plane">
            <div className="planeHead">
              <h2><Link size={17} />Connected integrations</h2>
              <span>{dashboard ? `${dashboard.plugins.length} connected` : "Awaiting API connection"}</span>
            </div>
            <div className="pluginStrip">
              {livePlugins.length ? livePlugins.map(([name, meta, state, tone]) => (
                <div className={`plugin ${tone}`} key={name}>
                  <i />
                  <span>
                    <strong>{name}</strong>
                    <small>{meta}</small>
                  </span>
                  <b>{state}</b>
                </div>
              )) : (
                <div className="plugin blue">
                  <i />
                  <span>
                    <strong>No plugins configured</strong>
                    <small>Start API with a Consize config to connect Kubernetes or Prometheus</small>
                  </span>
                  <b>Setup</b>
                </div>
              )}
            </div>
          </section>

          <div className="dashboardGrid">
            <div className="leftStack">
              <section className="panel recommendationPanel">
                <div className="panelHead recHead">
                  <h2><ListChecks size={18} />Top Prioritized<br />Recommendations</h2>
                  <div className="tabs">
                    <button>{dashboard?.recommendations.length ?? 0}<br />items</button>
                    <button className="active">All ({dashboard?.recommendations.length ?? 0})</button>
                    <button>Production</button>
                    <button>High Impact</button>
                    <button>Auto</button>
                  </div>
                </div>

                <div className="table">
                  <div className="row head">
                    <span>Resource / Target</span><span>Owner / Policy</span><span>Risk</span><span>Est. Savings</span><span>Action</span>
                  </div>
                  {liveRecommendations.length ? liveRecommendations.map((rec) => {
                    const Icon = rec.icon;
                    const Action = rec.actionIcon;
                    return (
                      <div className={`row ${rec === selected ? "selected" : ""}`} key={rec.raw.id}
                        onClick={() => "raw" in rec && setSelectedID(rec.raw.id)}>
                        <span className="target"><b className={rec.color}><Icon size={18} /></b><span><button className="recommendationSelect" aria-pressed={rec === selected} onClick={() => "raw" in rec && setSelectedID(rec.raw.id)}>{rec.name}</button><small>{rec.meta}</small></span></span>
                        <span><strong>{rec.owner}</strong><small>{rec.policy}</small></span>
                        <span><em className={`risk ${rec.riskTone}`}>{rec.risk}</em></span>
                        <span><strong className="money">{rec.savings}<small>/mo</small></strong></span>
                        <span>
                          <button
                            className={`rowBtn ${rec.primary ? "apply" : ""} ${rec.riskTone === "medium" ? "gold" : ""} ${rec.riskTone === "high" ? "locked" : ""}`}
                            disabled={"raw" in rec && (apiError !== "" || !["pending", "planned"].includes(rec.raw.status) || busyAction === `${rec.raw.id}:${rec.primary ? "execute" : "plan"}` || (rec.primary && (rec.raw.confidence === "low" || preflightFor(rec.raw.id).some((check) => check.status !== "passed"))))}
                            onClick={() => { setSelectedID(rec.raw.id); if (rec.primary) setConfirmApply(rec.raw); else void runAction(rec.raw, "plan"); }}
                          >
                            <Action size={15} />
                            {busyAction === ("raw" in rec ? `${rec.raw.id}:${rec.primary ? "execute" : "plan"}` : "") ? "Running" : rec.action}
                          </button>
                        </span>
                      </div>
                    );
                  }) : (
                    <div className="row empty">
                      <span>No recommendations yet</span><span>Run policy evaluation after metrics are connected.</span><span /><span /><span />
                    </div>
                  )}
                </div>
                <div className="panelFoot">
                  <span>{apiError ? "Backend API unavailable" : `Showing ${liveRecommendations.length} prioritized items`}</span>
                  <a href="#">View full recommendation matrix <ArrowRight size={16} /></a>
                </div>
              </section>

              <section className="panel auditPanel">
                <div className="panelHead">
                  <h2><FileText size={18} />Operational Verification & Audit Log</h2>
                  <span className="runState">Post-action metrics checks</span>
                </div>
                {liveAudit.map((item) => {
                  const Icon = item.icon;
                  return (
                    <div className="auditItem" key={item.title}>
                      <b className={item.tone}><Icon size={20} /></b>
                      <span>
                        <strong>{item.title}</strong>
                        <p>{item.line}</p>
                      </span>
                      <time>{item.time}</time>
                    </div>
                  );
                })}
                {!liveAudit.length && <div className="panelFoot">No applied changes or verification results yet.</div>}
              </section>
            </div>

            <aside className="rightStack">
              <section className="panel evalPanel">
                <div className="evalTop"><span><i />Selected recommendation</span></div>
                {selectedRaw ? <>
                  <h3>{selectedRaw.title}</h3>
                  <p>{selectedResource?.name || selectedRaw.resource_id} · {selectedResource?.provider || "Unknown provider"}{metadataValue(selectedResource, "namespace") ? ` · ${metadataValue(selectedResource, "namespace")}` : ""}</p>
                  <p>{selectedRaw.summary}</p>
                  <dl className="recommendationFacts">
                    <div><dt>Status</dt><dd>{selectedRaw.status === "planned" ? "Reviewed · not applied" : titleCase(selectedRaw.status)}</dd></div>
                    <div><dt>Confidence</dt><dd>{titleCase(selectedRaw.confidence)}</dd></div>
                    <div><dt>Risk</dt><dd>{titleCase(selectedRaw.risk)}</dd></div>
                    <div><dt>Action plugin</dt><dd>{selectedRaw.plugin_id}</dd></div>
                    <div><dt>Policy</dt><dd>{selectedPolicy?.policy_id || selectedRaw.policy_id || "Not evaluated yet"}</dd></div>
                    {selectedPolicy && <div><dt>Decision</dt><dd>{titleCase(selectedPolicy.decision)}</dd></div>}
                  </dl>
                  <div className="recommendationChange">
                    <h4>Proposed change</h4>
                    {Array.from(new Set([...Object.keys(selectedRaw.current || {}), ...Object.keys(selectedRaw.proposed || {})])).map((key) => (
                      <div key={key}><span>{titleCase(key)}</span><span>{formatChange(selectedRaw.current?.[key], key, selectedRaw.current?.resource)} <ArrowRight size={13} /> {formatChange(selectedRaw.proposed?.[key], key, selectedRaw.current?.resource)}</span></div>
                    ))}
                  </div>
                  <div className="recommendationEvidence">
                    <h4>Evidence</h4>
                    <ul>{(selectedRaw.evidence || []).map((line, index) => <li key={index}>{line}</li>)}</ul>
                    {!selectedRaw.evidence?.length && <p>No evidence supplied.</p>}
                  </div>
                  {selectedRaw.parameters?.verification_plan && <div className="recommendationEvidence">
                    <h4>Post-action verification</h4>
                    <p>Metrics plugin: {selectedRaw.parameters.verification_plan.metrics_plugin_id || "Not specified"}</p>
                    <ul>{selectedRaw.parameters.verification_plan.checks?.map((line) => <li key={line}>{line}</li>)}</ul>
                    <ul>{(selectedJob?.verification.checks || dashboard?.verification?.checks || []).map((check) => <li key={check.signal}>{check.signal.replaceAll("_", " ")}: {check.statistic} must not exceed {check.threshold}{check.mode === "ratio" ? "× baseline" : " (absolute)"}. Required evidence.</li>)}</ul>
                  </div>}
                  {selectedRaw.parameters?.confidence_assessment && <div className="recommendationEvidence">
                    <h4>Confidence assessment</h4>
                    <p>Profile: {selectedRaw.parameters.confidence_assessment.profile}. History: {selectedRaw.parameters.confidence_assessment.observed_history || "Unavailable"}; required: {selectedRaw.parameters.confidence_assessment.required_history}.</p>
                    <p>Coverage: {(selectedRaw.parameters.confidence_assessment.coverage_ratio * 100).toFixed(1)}%</p>
                    <ul>{selectedRaw.parameters.confidence_assessment.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul>
                  </div>}
                  {selectedPreflight.length > 0 && <div className="recommendationEvidence">
                    <h4>Infrastructure preflight</h4>
                    <dl className="recommendationFacts">{selectedPreflight.map((check) => <div key={check.id}><dt>{check.id.toUpperCase()}</dt><dd>{titleCase(check.status)}</dd></div>)}</dl>
                    <ul>{selectedPreflight.map((check) => <li key={check.id}>{check.message}{check.details?.before && check.details?.after ? ` (${String(check.details.before)} → ${String(check.details.after)})` : ""}</li>)}</ul>
                  </div>}
                  {selectedJob && <div className="recommendationEvidence">
                    <h4>Execution & recovery</h4>
                    <p>{titleCase(selectedJob.state)}</p>
                    <p>Rollback on failed checks: {selectedJob.verification.rollback_on_failure ? "Enabled" : "Manual recovery"}. Rollback on verification timeout: {selectedJob.verification.rollback_on_timeout ? "Enabled" : "Manual recovery"}.</p>
                    {selectedJob.window_start && !selectedJob.window_start.startsWith("0001") && <p>Observation starts: {new Date(selectedJob.window_start).toLocaleString()}</p>}
                    {selectedJob.deadline && <p>Deadline: {new Date(selectedJob.deadline).toLocaleString()}</p>}
                    {selectedJob.last_error && <p>{selectedJob.last_error}</p>}
                    {selectedJob.verification_result && <p>Metrics result: {selectedJob.verification_result.status}. {selectedJob.verification_result.reasons?.join("; ")}</p>}
                  </div>}
                  {selectedJob?.state === "manual_intervention" && <button className="ghost" onClick={() => setConfirmApply(selectedRaw)}>Review & Retry Recovery</button>}
                  {selectedRaw.confidence === "low" && <p>Evidence does not meet confidence requirements. Review is available; applying is blocked.</p>}
                  {selectedRaw.algorithm_id === "prometheus-headroom-v1" && !selectedRaw.parameters?.confidence_assessment && <p>This recommendation uses legacy confidence scoring. Generate a new recommendation to see the current assessment. Execution rechecks fresh evidence.</p>}
                  <button className="cta" disabled={apiError !== "" || busyAction !== null || !["pending", "planned"].includes(selectedRaw.status) || (selected?.primary && (selectedRaw.confidence === "low" || selectedPreflight.some((check) => check.status !== "passed")))} onClick={() => selected?.primary ? setConfirmApply(selectedRaw) : void runAction(selectedRaw, "plan")}>
                    <ArrowRight size={17} />{busyAction ? "Running…" : selected?.primary ? "Apply Recommendation" : "Review Change"}
                  </button>
                </> : <p>{apiError ? "Connect the backend to inspect recommendations." : "No recommendation selected. Generate a recommendation to view its details."}</p>}
              </section>
            </aside>
          </div>
          {confirmApply && <dialog ref={applyDialog} className="applyDialog" aria-labelledby="applyTitle" onCancel={(event) => { event.preventDefault(); setConfirmApply(null); }}>
            <h2 id="applyTitle">{confirmApply.status === "manual_intervention" ? "Confirm recovery retry" : "Confirm infrastructure change"}</h2>
            <p>{confirmApply.title}</p>
            {Object.keys(confirmApply.proposed || {}).filter((key) => key !== "resource").map((key) => <p key={key}>{titleCase(key)}: {formatChange(confirmApply.status === "manual_intervention" ? confirmApply.proposed?.[key] : confirmApply.current?.[key], key, confirmApply.current?.resource)} → {formatChange(confirmApply.status === "manual_intervention" ? confirmApply.current?.[key] : confirmApply.proposed?.[key], key, confirmApply.current?.resource)}</p>)}
            <p>{confirmApply.status === "manual_intervention" ? "A persistent worker will retry restoration and verify the restored workload." : "This queues the reviewed change for direct apply. A persistent worker verifies workload health and follows the configured rollback policy. External drift stops recovery for manual review."}</p>
            <button className="ghost" autoFocus onClick={() => setConfirmApply(null)}>Cancel</button>
            {confirmApply.status === "manual_intervention" && <p>This retries restoration of the captured original state. External drift must first be reconciled manually; it will not be overwritten.</p>}
            <button className="cta" onClick={() => { const rec = confirmApply; setConfirmApply(null); void runAction(rec, rec.status === "manual_intervention" ? "recover" : "execute"); }}>{confirmApply.status === "manual_intervention" ? "Confirm Recovery" : "Confirm Apply"}</button>
          </dialog>}
        </main>
      </section>
    </div>
  );
}
