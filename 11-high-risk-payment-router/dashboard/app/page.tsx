import { mockPspHealth, mockRules, mockTransactions, mockWebhookDeliveries } from "../lib/mock-data";

export default function Page() {
  return (
    <main className="shell">
      <section className="hero">
        <p className="eyebrow">RouteKit merchant console</p>
        <h1>Route high-risk payments with token-only PSP failover.</h1>
        <p className="lede">
          Manage PSP credentials, routing rules, transaction search, and webhook delivery without ever collecting raw card data.
        </p>
      </section>

      <section className="grid cards">
        {mockPspHealth.map((psp) => (
          <article className="card" key={psp.code}>
            <div className="cardHeader">
              <h2>{psp.label}</h2>
              <span className={`pill ${psp.state}`}>{psp.state}</span>
            </div>
            <p className="metric">{psp.authRate}%</p>
            <p>Auth rate, p95 {psp.p95} ms, region {psp.region}</p>
          </article>
        ))}
      </section>

      <section className="panel">
        <div className="sectionHeader">
          <h2>Routing Rules</h2>
          <button type="button">Add rule</button>
        </div>
        <div className="table">
          {mockRules.map((rule) => (
            <div className="row" key={rule.priority}>
              <span>#{rule.priority}</span>
              <strong>{rule.predicate}</strong>
              <span>{rule.action}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="grid">
        <article className="panel">
          <h2>Transaction Search</h2>
          <input placeholder="Search by transaction id, PSP id, customer ref" />
          {mockTransactions.map((txn) => (
            <div className="miniRow" key={txn.id}>
              <span>{txn.id}</span>
              <strong>{txn.state}</strong>
              <span>{txn.psp}</span>
            </div>
          ))}
        </article>
        <article className="panel">
          <h2>Outbound Webhooks</h2>
          {mockWebhookDeliveries.map((delivery) => (
            <div className="miniRow" key={delivery.id}>
              <span>{delivery.event}</span>
              <strong>{delivery.status}</strong>
              <span>{delivery.nextAttempt}</span>
            </div>
          ))}
        </article>
      </section>
    </main>
  );
}

