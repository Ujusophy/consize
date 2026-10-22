import type { Money } from "@/schemas";

export function formatMoney(m: Money) {
  const display = new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: m.currency,
    maximumFractionDigits: 0,
  }).format(m.amount_minor / 100);
  const basisLabel = m.basis === "verified_operational" ? "verified" : m.basis.replaceAll("_", " ");
  return { display, basisLabel };
}

export function titleCase(s: string) {
  return s.replaceAll("_", " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

// "approval_required" -> "Approval required" (sentence case, unlike titleCase)
export function sentenceCase(s: string) {
  const t = s.replaceAll("_", " ");
  return t.charAt(0).toUpperCase() + t.slice(1);
}

// Relative time; callers show the exact timestamp via title/dateTime.
export function relativeTime(iso: string, now = Date.now()) {
  const minutes = Math.round((now - new Date(iso).getTime()) / 60_000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.round(hours / 24)}d ago`;
}

// Compact money for tight spaces: $28.1k/mo style.
export function formatMoneyCompact(m: { amount_minor: number; currency: string }) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: m.currency,
    notation: "compact",
    maximumFractionDigits: 1,
  })
    .format(m.amount_minor / 100)
    .toLowerCase();
}

/** Turn an evaluation timestamp into "4m ago", "1h ago", or "2d ago".
 * Passing the current time in makes the formatting easy to check with fixed examples.
 */
export function formatRelativeTime(timestamp: string | null | undefined, now: number): string {
  // Missing or invalid dates mean unknown, rather than an invented evaluation age.
  if (!timestamp) return "—";
  const evaluatedAt = Date.parse(timestamp);
  if (!Number.isFinite(evaluatedAt) || !Number.isFinite(now)) return "—";

  // Small differences between server and browser clocks should not show negative ages.
  const minutes = Math.floor(Math.max(0, now - evaluatedAt) / 60_000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}
