import { RotateCcw } from "lucide-react";
import type { DlqItem } from "@/lib/types";

export function DlqTable({ items }: { items: DlqItem[] }) {
  return (
    <section className="rounded-3xl border border-white/10 bg-black/25 p-5">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-stone-500">Dead letters</p>
          <h2 className="mt-1 font-display text-3xl font-black text-white">Parked deliveries</h2>
        </div>
        <button className="rounded-full border border-ember-400/30 bg-ember-500/10 px-5 py-3 text-sm font-black text-ember-100 transition hover:bg-ember-500/20">
          Bulk replay placeholder
        </button>
      </div>
      <div className="mt-6 overflow-hidden rounded-2xl border border-white/10">
        {items.map((item) => (
          <article key={item.id} className="grid gap-4 border-b border-white/10 bg-white/[0.04] p-4 last:border-b-0 md:grid-cols-[1fr_160px_160px] md:items-center">
            <div>
              <p className="font-mono text-xs text-stone-500">{item.id}</p>
              <h3 className="mt-1 font-bold text-white">{item.endpoint}</h3>
              <p className="mt-1 text-sm text-stone-400">
                {item.destination} · {item.reason}
              </p>
            </div>
            <div className="text-sm text-stone-300">
              <p>{item.failedAt}</p>
              <p className="text-xs text-stone-500">{item.attempts} attempts</p>
            </div>
            <button className="inline-flex items-center justify-center gap-2 rounded-full bg-lichen-300 px-4 py-3 text-sm font-black text-ink-950 transition hover:-translate-y-0.5">
              <RotateCcw aria-hidden className="size-4" />
              Replay
            </button>
          </article>
        ))}
      </div>
    </section>
  );
}
