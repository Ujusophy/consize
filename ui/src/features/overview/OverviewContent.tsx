"use client";

import { AuditLog } from "./AuditLog";
import { IntegrationsStrip } from "./IntegrationsStrip";
import { MetricsRow } from "./MetricsRow";
import { OverviewHeader } from "./OverviewHeader";
import { RecommendationPanel } from "./RecommendationPanel";

export function OverviewContent() {
  return (
    <>
      <OverviewHeader />
      <MetricsRow />
      <IntegrationsStrip />
      <RecommendationPanel />
      <AuditLog />
    </>
  );
}