import { Suspense } from "react";
import { OverviewContent } from "@/features/overview/OverviewContent";

// The Overview reads scope and filters from the URL, so its client content
// sits inside Suspense (a Next requirement for useSearchParams).
export default function OverviewPage() {
  return (
    <main className="px-6 pt-4 pb-10 space-y-4 max-w-[1680px] mx-auto">
      <Suspense fallback={<div className="h-96 rounded-btn bg-app-surface border border-app-border animate-pulse" />}>
        <OverviewContent />
      </Suspense>
    </main>
  );
}