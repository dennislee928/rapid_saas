import { EndpointList } from "@/components/endpoint-list";
import { EventTimeline } from "@/components/event-timeline";
import { AppShell } from "@/components/app-shell";
import { endpoints, timelineEvents } from "@/lib/mock-data";

export default function DashboardPage() {
  return (
    <AppShell eyebrow="Operations console" title="Endpoints, signals, and quota in the same breath.">
      <div className="grid gap-6 xl:grid-cols-[1fr_0.9fr]">
        <EndpointList endpoints={endpoints} />
        <EventTimeline events={timelineEvents} />
      </div>
    </AppShell>
  );
}
