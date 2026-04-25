import Link from "next/link";
import { Activity, Braces, Gauge, RotateCcw, Shield } from "lucide-react";
import { AuthStatus } from "./auth-status";
import { QuotaMeter } from "./quota-meter";
import { quota } from "@/lib/mock-data";

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: Activity },
  { href: "/rules", label: "Rules JSON", icon: Braces },
  { href: "/dlq", label: "DLQ / Replay", icon: RotateCcw }
];

export function AppShell({ children, eyebrow, title }: { children: React.ReactNode; eyebrow: string; title: string }) {
  return (
    <div className="min-h-screen bg-ink-950 text-stone-100">
      <div className="pointer-events-none fixed inset-0 bg-[radial-gradient(circle_at_15%_10%,rgba(255,104,31,0.24),transparent_32%),radial-gradient(circle_at_80%_0%,rgba(168,207,132,0.16),transparent_30%),linear-gradient(135deg,rgba(255,255,255,0.05)_0,transparent_40%)]" />
      <div className="relative mx-auto grid min-h-screen w-full max-w-7xl gap-6 px-4 py-4 lg:grid-cols-[280px_1fr] lg:px-6">
        <aside className="rounded-[2rem] border border-white/10 bg-ink-900/80 p-5 shadow-panel backdrop-blur">
          <Link href="/" className="flex items-center gap-3">
            <span className="rounded-2xl bg-ember-500 p-3 text-ink-950 shadow-glow">
              <Shield aria-hidden className="size-5" />
            </span>
            <span>
              <span className="block font-display text-xl font-black tracking-tight">RouterOps</span>
              <span className="block text-xs uppercase tracking-[0.3em] text-stone-500">Security webhooks</span>
            </span>
          </Link>

          <nav className="mt-8 space-y-2">
            {navItems.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className="group flex items-center gap-3 rounded-2xl px-3 py-3 text-sm text-stone-300 transition hover:bg-white/10 hover:text-white"
              >
                <item.icon aria-hidden className="size-4 text-ember-400 transition group-hover:scale-110" />
                {item.label}
              </Link>
            ))}
          </nav>

          <div className="mt-8 space-y-4">
            <QuotaMeter quota={quota} />
            <div className="rounded-2xl border border-ember-400/20 bg-ember-500/10 p-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-ember-50">
                <Gauge aria-hidden className="size-4" />
                Edge ACK target
              </div>
              <p className="mt-2 text-3xl font-black tracking-tight text-white">&lt;100ms</p>
              <p className="mt-1 text-xs text-stone-400">Cloudflare Worker accepts fast, queue handles burst pressure.</p>
            </div>
            <AuthStatus />
          </div>
        </aside>

        <main className="rounded-[2rem] border border-white/10 bg-stone-950/40 p-5 shadow-panel backdrop-blur lg:p-8">
          <header className="mb-8">
            <p className="text-xs font-bold uppercase tracking-[0.4em] text-lichen-300">{eyebrow}</p>
            <h1 className="mt-3 max-w-3xl font-display text-4xl font-black tracking-tight text-white md:text-6xl">{title}</h1>
          </header>
          {children}
        </main>
      </div>
    </div>
  );
}
