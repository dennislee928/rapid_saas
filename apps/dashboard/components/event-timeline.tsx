import type { EventStatus, TimelineEvent } from "@/lib/types";

const statusClass: Record<EventStatus, string> = {
  queued: "border-blue-300/50 text-blue-200",
  delivered: "border-lichen-300/50 text-lichen-100",
  failed: "border-ember-400/50 text-ember-100",
  dropped: "border-stone-400/50 text-stone-300",
  dlq: "border-red-400/60 text-red-200"
};

export function EventTimeline({ events }: { events: TimelineEvent[] }) {
  return (
    <section className="rounded-3xl border border-white/10 bg-black/20 p-5">
      <div className="flex items-center justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-stone-500">Live tail</p>
          <h2 className="mt-1 font-display text-3xl font-black text-white">Last events</h2>
        </div>
        <span className="rounded-full border border-lichen-300/30 bg-lichen-300/10 px-3 py-1 text-xs font-bold text-lichen-100">polls every 5s</span>
      </div>
      <div className="mt-6 space-y-3">
        {events.map((event) => (
          <article key={event.id} className="grid gap-3 rounded-2xl border border-white/10 bg-white/[0.05] p-4 md:grid-cols-[92px_1fr_auto] md:items-center">
            <time className="font-mono text-xs text-stone-500">{event.receivedAt}</time>
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <span className={`rounded-full border px-2 py-1 text-xs font-bold uppercase ${statusClass[event.status]}`}>{event.status}</span>
                <span className="text-xs uppercase tracking-[0.2em] text-stone-500">{event.severity}</span>
                <span className="text-xs text-stone-500">{event.source}</span>
              </div>
              <p className="mt-2 text-sm text-stone-200">{event.message}</p>
              <p className="mt-1 text-xs text-stone-500">{event.endpoint}</p>
            </div>
            <p className="font-mono text-xs text-stone-400">{event.latencyMs ? `${event.latencyMs}ms` : "no fanout"}</p>
          </article>
        ))}
      </div>
    </section>
  );
}
