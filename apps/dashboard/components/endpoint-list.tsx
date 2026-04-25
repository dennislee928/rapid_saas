import { Copy, RadioTower } from "lucide-react";
import type { Endpoint, EndpointStatus } from "@/lib/types";
import { compactNumber } from "@/lib/format";

const statusClass: Record<EndpointStatus, string> = {
  healthy: "bg-lichen-300 text-ink-950",
  muted: "bg-stone-500 text-white",
  failing: "bg-ember-500 text-ink-950"
};

export function EndpointList({ endpoints }: { endpoints: Endpoint[] }) {
  return (
    <section className="grid gap-4">
      {endpoints.map((endpoint) => (
        <article key={endpoint.id} className="rounded-3xl border border-white/10 bg-white/[0.07] p-5 transition hover:-translate-y-1 hover:bg-white/[0.1]">
          <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div>
              <div className="flex flex-wrap items-center gap-3">
                <h2 className="font-display text-2xl font-black text-white">{endpoint.name}</h2>
                <span className={`rounded-full px-3 py-1 text-xs font-black uppercase tracking-wide ${statusClass[endpoint.status]}`}>
                  {endpoint.status}
                </span>
              </div>
              <p className="mt-2 text-sm text-stone-400">{endpoint.preset} preset · last seen {endpoint.lastSeen}</p>
            </div>
            <div className="grid grid-cols-2 gap-3 text-right">
              <Metric label="24h events" value={compactNumber(endpoint.events24h)} />
              <Metric label="p95 ACK" value={`${endpoint.p95AckMs}ms`} />
            </div>
          </div>
          <div className="mt-5 flex items-center gap-3 rounded-2xl border border-white/10 bg-black/25 px-4 py-3 font-mono text-xs text-stone-300">
            <RadioTower aria-hidden className="size-4 shrink-0 text-ember-400" />
            <span className="truncate">{endpoint.ingressUrl}</span>
            <Copy aria-hidden className="ml-auto size-4 shrink-0 text-stone-500" />
          </div>
        </article>
      ))}
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-black/20 px-4 py-3">
      <p className="text-xs uppercase tracking-[0.2em] text-stone-500">{label}</p>
      <p className="mt-1 text-xl font-black text-white">{value}</p>
    </div>
  );
}
