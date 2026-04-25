"use client";

import { useMemo, useState } from "react";
import { AlertTriangle, CheckCircle2 } from "lucide-react";
import type { RuleDocument } from "@/lib/types";

export function RulesEditor({ initialRules }: { initialRules: RuleDocument }) {
  const [draft, setDraft] = useState(() => JSON.stringify(initialRules, null, 2));

  const validation = useMemo(() => {
    try {
      const parsed = JSON.parse(draft) as Partial<RuleDocument>;
      if (!parsed.endpoint_id || !Array.isArray(parsed.rules)) {
        return { ok: false, message: "Expected endpoint_id and rules array." };
      }

      return { ok: true, message: `${parsed.rules.length} rules ready to publish.` };
    } catch (error) {
      return { ok: false, message: error instanceof Error ? error.message : "Invalid JSON." };
    }
  }, [draft]);

  return (
    <section className="rounded-3xl border border-white/10 bg-black/25 p-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-stone-500">Raw JSON MVP</p>
          <h2 className="mt-1 font-display text-3xl font-black text-white">Rule chain editor</h2>
        </div>
        <div
          className={`flex items-center gap-2 rounded-full border px-3 py-2 text-xs font-bold ${
            validation.ok ? "border-lichen-300/30 bg-lichen-300/10 text-lichen-100" : "border-ember-400/40 bg-ember-500/10 text-ember-100"
          }`}
        >
          {validation.ok ? <CheckCircle2 aria-hidden className="size-4" /> : <AlertTriangle aria-hidden className="size-4" />}
          {validation.message}
        </div>
      </div>
      <textarea
        aria-label="Rules JSON editor"
        className="mt-5 min-h-[560px] w-full resize-y rounded-2xl border border-white/10 bg-ink-950/80 p-5 font-mono text-sm leading-6 text-lichen-100 outline-none ring-ember-400/40 transition placeholder:text-stone-600 focus:border-ember-400/50 focus:ring-4"
        spellCheck={false}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
      />
      <div className="mt-4 flex flex-wrap gap-3">
        <button className="rounded-full bg-ember-500 px-5 py-3 text-sm font-black text-ink-950 shadow-glow transition hover:-translate-y-0.5">
          Publish rules placeholder
        </button>
        <button className="rounded-full border border-white/15 px-5 py-3 text-sm font-bold text-stone-200 transition hover:bg-white/10">
          Dry run sample event
        </button>
      </div>
    </section>
  );
}
