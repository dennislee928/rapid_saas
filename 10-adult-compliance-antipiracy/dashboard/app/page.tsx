const stats = [
  { label: "Verified sessions", value: "128" },
  { label: "Abandoned flows", value: "9" },
  { label: "Fail-open events", value: "0" }
];

const reclaimItems = ["12 new matches awaiting review", "4 takedown drafts ready", "0 auto-send rules enabled"];

export default function DashboardPage() {
  return (
    <main className="shell">
      <section className="hero">
        <p className="eyebrow">Aegis Trust MVP</p>
        <h1>GateKeep compliance gateway and Reclaim anti-piracy operations</h1>
        <p>
          Operational review surface for age-gate sessions, anti-piracy matches, and takedown drafts. Regulatory copy shown here is placeholder product language, not legal advice or compliance certification.
        </p>
      </section>

      <section className="grid">
        <article className="card accent">
          <span className="label">GateKeep</span>
          <h2>Age-gate proxy</h2>
          <p>JWT cookie verification, hosted verify flow, callback minting, CSP injection, and bot screening.</p>
          <dl>
            {stats.map((stat) => (
              <div key={stat.label}>
                <dt>{stat.label}</dt>
                <dd>{stat.value}</dd>
              </div>
            ))}
          </dl>
        </article>

        <article className="card">
          <span className="label">Reclaim</span>
          <h2>Anti-piracy queue</h2>
          <p>Assets are hashed, originals are scheduled for deletion, and matches require human approval.</p>
          <ul>
            {reclaimItems.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </article>

        <article className="card">
          <span className="label">Human Review</span>
          <h2>DMCA draft preview</h2>
          <p>
            Notices are generated as drafts. A tenant reviewer must confirm rights ownership, match accuracy, and contact details before any external send.
          </p>
          <button type="button">Open review queue</button>
        </article>
      </section>
    </main>
  );
}
