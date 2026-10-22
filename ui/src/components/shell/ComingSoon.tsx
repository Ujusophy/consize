import Link from "next/link";
import { ArrowLeft } from "lucide-react";

export function ComingSoon({ title }: { title: string }) {
  return (
    <div className="flex flex-col items-center justify-center text-center py-24 gap-4">
      <h1 className="text-xl font-semibold text-fg">{title}</h1>
      <p className="text-sm text-slate-400 max-w-sm">
        This section isn&apos;t built yet. Overview is where the real, wired-up work lives right now.
      </p>
      <Link
        href="/"
        className="inline-flex items-center gap-2 h-9 px-4 bg-app-surface hover:bg-slate-800 border border-app-border rounded-lg text-sm text-fg transition-colors"
      >
        <ArrowLeft className="w-4 h-4" />
        Back to Overview
      </Link>
    </div>
  );
}