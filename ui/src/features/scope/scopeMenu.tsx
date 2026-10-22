"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { ChevronDown, Server as Stack, X } from "lucide-react";
import { useDismiss } from "@/lib/useDismiss";
import { useScopeOptions } from "../overview/queries";
import { SCOPE_KEYS, useScope, type Scope, type ScopeKey } from "./useScope";

const FIELDS: { key: ScopeKey; label: string; all: string; list: "providers" | "sources" | "regions" | "resource_types" }[] = [
  { key: "provider", label: "Provider", all: "All connected providers", list: "providers" },
  { key: "source", label: "Connected source", all: "All connected sources", list: "sources" },
  { key: "region", label: "Region / location", all: "All discovered locations", list: "regions" },
  { key: "type", label: "Resource type", all: "All discovered resource types", list: "resource_types" },
];

export function ScopeMenu() {
  const { scope, update } = useScope();
  const { data: options, isError } = useScopeOptions();
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<Scope>({});
  const wrapRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);

  const close = useCallback(() => {
    setOpen(false);
    triggerRef.current?.focus(); // restore focus (WCAG 2.2 / PRD §20)
  }, []);
  useDismiss(wrapRef, open, close);

  useEffect(() => {
    if (open) wrapRef.current?.querySelector("select")?.focus();
  }, [open]);

  function toggle() {
    if (open) return close();
    setDraft(scope); // edit a copy; nothing changes until "Apply scope"
    setOpen(true);
  }

  function apply(next: Scope) {
    update({ provider: next.provider, source: next.source, region: next.region, type: next.type });
    close();
  }

  const labelOf = (key: ScopeKey, value: string) => {
    const f = FIELDS.find((x) => x.key === key)!;
    return options?.[f.list].find((o) => o.value === value)?.label ?? value;
  };
  const chosen = SCOPE_KEYS.filter((k) => scope[k]).map((k) => labelOf(k, scope[k]!));
  const summary =
    chosen.length === 0
      ? "All infrastructure"
      : chosen.length <= 2
        ? chosen.join(" · ")
        : `${chosen.slice(0, 2).join(" · ")} +${chosen.length - 2}`;

  return (
    <div ref={wrapRef} className="relative hidden md:block">
      <button
        ref={triggerRef}
        type="button"
        aria-expanded={open}
        aria-haspopup="dialog"
        onClick={toggle}
        className="flex items-center gap-2 px-3 h-8 min-w-[190px] bg-app-surface border border-app-border hover:border-app-border-strong rounded-btn text-xs text-fg-secondary hover:text-fg transition-colors"
      >
        <Stack className="w-3.5 h-3.5 text-ok-strong shrink-0" aria-hidden />
        <span className="min-w-0 flex-1 truncate text-left">{summary}</span>
        <ChevronDown className="w-3.5 h-3.5 text-fg-muted" aria-hidden />
      </button>

      {open && (
        <div
          role="dialog"
          aria-label="Infrastructure scope"
          className="absolute right-0 top-10 z-50 w-[380px] bg-app-sidebar border border-app-border rounded-btn shadow-2xl p-4"
        >
          <div className="flex items-start justify-between gap-4 pb-3 border-b border-app-border">
            <div>
              <h3 className="text-sm font-semibold text-fg">Infrastructure scope</h3>
              <p className="text-[11px] text-fg-muted mt-1">
                Choose which connected infrastructure appears across this dashboard.
              </p>
            </div>
            <button type="button" aria-label="Close infrastructure scope" onClick={close} className="text-fg-muted hover:text-fg">
              <X className="w-4 h-4" aria-hidden />
            </button>
          </div>

          <div className="grid grid-cols-2 gap-3 py-3">
            {FIELDS.map((f) => (
              <label key={f.key} className="block">
                <span className="block text-[10px] font-medium text-fg-muted mb-1">{f.label}</span>
                <span className="relative block">
                  <select
                    value={draft[f.key] ?? ""}
                    disabled={!options}
                    onChange={(e) => setDraft({ ...draft, [f.key]: e.target.value || undefined })}
                    className="w-full h-9 appearance-none truncate rounded-btn border border-app-border bg-app-field pl-2.5 pr-7 text-xs text-fg-secondary hover:border-app-border-strong focus:border-emerald-500 focus:outline-none focus:ring-1 focus:ring-emerald-500/40 disabled:cursor-not-allowed"
                  >
                    <option value="">{f.all}</option>
                    {options?.[f.list].map((o) => (
                      <option key={o.value} value={o.value}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-fg-muted" aria-hidden />
                </span>
              </label>
            ))}
          </div>

          <p className="text-[10px] text-fg-faint -mt-1 mb-3">
            {isError
              ? "Couldn't load scope options. Try again shortly."
              : "Options are populated from connected sources and discovered resources."}
          </p>
          <p className="text-[10px] text-fg-faint mb-3">
            Scope filters savings and recommendations. Panels that can&apos;t be filtered say so.
          </p>

          <div className="flex items-center justify-between gap-3 pt-3 border-t border-app-border">
            <div className="flex items-center gap-2">
              <button type="button" onClick={() => apply({})} className="h-8 px-2 text-xs text-fg-secondary hover:text-fg">
                Reset
              </button>
              <span aria-hidden className="h-4 w-px bg-app-border" />
              <Link href="/plugins" onClick={close} className="inline-flex h-8 items-center px-2 text-xs font-medium text-info-fg hover:underline">
                Manage connections
              </Link>
            </div>
            <button
              type="button"
              onClick={() => apply(draft)}
              className="h-8 px-4 bg-emerald-600 hover:bg-emerald-500 text-white rounded-btn text-xs font-medium"
            >
              Apply scope
            </button>
          </div>
        </div>
      )}
    </div>
  );
}