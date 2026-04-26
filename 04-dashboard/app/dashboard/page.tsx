import { EndpointList } from "@/components/endpoint-list";
import { EventTimeline } from "@/components/event-timeline";
import { AppShell } from "@/components/app-shell";
import { getDashboardData } from "@/lib/api";

export default async function DashboardPage() {
  const { endpoints, timelineEvents, quota } = await getDashboardData();

  return (
    <AppShell eyebrow="Operations console" title="Endpoints, signals, and quota in the same breath." quota={quota}>
      <div className="grid gap-6 xl:grid-cols-[1fr_0.9fr]">
        <EndpointList endpoints={endpoints} />
        <EventTimeline events={timelineEvents} />
      </div>
    </AppShell>
  );
}
