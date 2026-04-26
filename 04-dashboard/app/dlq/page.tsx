import { AppShell } from "@/components/app-shell";
import { DlqTable } from "@/components/dlq-table";
import { getDlqData } from "@/lib/api";

export default async function DlqPage() {
  const { dlqItems, quota } = await getDlqData();

  return (
    <AppShell eyebrow="Retry controls" title="Dead letters stay visible until an operator moves them." quota={quota}>
      <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
        <DlqTable items={dlqItems} />
        <aside className="rounded-3xl border border-white/10 bg-white/[0.06] p-5">
          <p className="text-xs uppercase tracking-[0.3em] text-stone-500">Replay semantics</p>
          <h2 className="mt-2 font-display text-3xl font-black text-white">Safe by default</h2>
          <p className="mt-4 text-sm leading-6 text-stone-300">
            MVP replay should refire the rendered outbound payload, not rerun changed rules. Add a v1 toggle only after warning operators that rule behavior may differ.
          </p>
          <div className="mt-5 rounded-2xl border border-ember-400/20 bg-ember-500/10 p-4 text-sm text-ember-50">
            Placeholder buttons are intentionally inert until the Fly API exposes scoped replay endpoints.
          </div>
        </aside>
      </div>
    </AppShell>
  );
}
