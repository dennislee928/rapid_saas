import Link from "next/link";
import { ArrowRight, Braces, RadioTower, RotateCcw } from "lucide-react";

const features = [
  {
    icon: RadioTower,
    title: "Dedicated ingress URLs",
    copy: "Per-endpoint health, HMAC preset visibility, and sub-100ms ACK posture in one view."
  },
  {
    icon: Braces,
    title: "Rules without a workflow canvas",
    copy: "Raw JSONLogic editing keeps the MVP explicit while the engine matures."
  },
  {
    icon: RotateCcw,
    title: "Replay with guardrails",
    copy: "DLQ items are parked, inspected, and replayed intentionally rather than silently retried forever."
  }
];

export default function LandingPage() {
  return (
    <main className="min-h-screen overflow-hidden bg-ink-950 text-white">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_15%,rgba(255,104,31,0.32),transparent_32%),radial-gradient(circle_at_84%_20%,rgba(168,207,132,0.18),transparent_30%)]" />
      <section className="relative mx-auto flex min-h-screen max-w-7xl flex-col px-5 py-8">
        <nav className="flex items-center justify-between">
          <Link href="/" className="font-display text-2xl font-black tracking-tight">
            RouterOps
          </Link>
          <Link href="/dashboard" className="rounded-full border border-white/15 px-4 py-2 text-sm font-bold text-stone-200 transition hover:bg-white/10">
            Open dashboard
          </Link>
        </nav>

        <div className="grid flex-1 items-center gap-10 py-16 lg:grid-cols-[1.05fr_0.95fr]">
          <div>
            <p className="text-xs font-bold uppercase tracking-[0.45em] text-ember-400">Security webhook router</p>
            <h1 className="mt-5 max-w-4xl font-display text-6xl font-black leading-[0.92] tracking-tight md:text-8xl">
              Route noisy alerts with edge-speed control.
            </h1>
            <p className="mt-7 max-w-2xl text-lg leading-8 text-stone-300">
              A focused dashboard for endpoints, delivery timelines, JSON rules, quota pressure, and DLQ replay. Built for Cloudflare Pages with Clerk and Fly API seams ready.
            </p>
            <div className="mt-9 flex flex-wrap gap-3">
              <Link href="/dashboard" className="inline-flex items-center gap-2 rounded-full bg-ember-500 px-6 py-4 font-black text-ink-950 shadow-glow transition hover:-translate-y-1">
                Inspect live routes <ArrowRight aria-hidden className="size-4" />
              </Link>
              <a href="#scope" className="rounded-full border border-white/15 px-6 py-4 font-bold text-stone-200 transition hover:bg-white/10">
                View MVP scope
              </a>
            </div>
          </div>

          <div className="relative rounded-[2rem] border border-white/10 bg-white/[0.06] p-5 shadow-panel backdrop-blur">
            <div className="rounded-[1.5rem] border border-ember-400/20 bg-black/35 p-5">
              <p className="font-mono text-xs text-lichen-300">POST /w/ep_01HV9_falcon</p>
              <div className="mt-6 grid gap-3">
                {["verify signature", "enqueue raw payload", "evaluate JSONLogic", "fan out to destinations"].map((step, index) => (
                  <div key={step} className="flex items-center justify-between rounded-2xl border border-white/10 bg-white/[0.05] p-4">
                    <span className="font-bold">{step}</span>
                    <span className="font-mono text-xs text-stone-500">0{index + 1}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div id="scope" className="grid gap-4 pb-8 md:grid-cols-3">
          {features.map((feature) => (
            <article key={feature.title} className="rounded-3xl border border-white/10 bg-white/[0.06] p-5 backdrop-blur">
              <feature.icon aria-hidden className="size-6 text-ember-400" />
              <h2 className="mt-4 font-display text-2xl font-black">{feature.title}</h2>
              <p className="mt-3 text-sm leading-6 text-stone-400">{feature.copy}</p>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
