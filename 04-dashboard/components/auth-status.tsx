import { ShieldCheck } from "lucide-react";

export function AuthStatus() {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.06] p-4 text-sm text-stone-200">
      <div className="flex items-center gap-3">
        <div className="rounded-xl bg-lichen-300/15 p-2 text-lichen-100">
          <ShieldCheck aria-hidden className="size-4" />
        </div>
        <div>
          <p className="font-semibold text-white">Clerk auth placeholder</p>
          <p className="text-stone-400">Wire `ClerkProvider`, sign-in routes, and tenant provisioning webhook when API keys exist.</p>
        </div>
      </div>
    </div>
  );
}
