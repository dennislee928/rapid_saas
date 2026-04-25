import { compactNumber, percent } from "@/lib/format";

export function QuotaMeter({ quota }: { quota: { used: number; limit: number; resetLabel: string } }) {
  const usage = percent(quota.used, quota.limit);

  return (
    <section className="rounded-2xl border border-white/10 bg-white/[0.06] p-4">
      <div className="flex items-baseline justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-stone-500">Monthly quota</p>
          <p className="mt-1 text-2xl font-black text-white">{usage}%</p>
        </div>
        <p className="text-right text-xs text-stone-400">{quota.resetLabel}</p>
      </div>
      <div className="mt-4 h-3 overflow-hidden rounded-full bg-black/30">
        <div className="h-full rounded-full bg-gradient-to-r from-lichen-300 via-ember-400 to-ember-700" style={{ width: `${usage}%` }} />
      </div>
      <p className="mt-3 text-xs text-stone-400">
        {compactNumber(quota.used)} of {compactNumber(quota.limit)} events processed
      </p>
    </section>
  );
}
