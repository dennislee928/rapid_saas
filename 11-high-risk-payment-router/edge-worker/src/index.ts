import { Hono } from "hono";
import type { Context } from "hono";
import { rejectRawCardData, sha256Hex, verifyMerchantSignature, type IdempotencyRecord } from "./security";

type Env = {
  IDEMPOTENCY: KVNamespace;
  ORCHESTRATOR_URL: string;
  ROUTEKIT_HMAC_SECRET: string;
  ORCHESTRATOR_SHARED_SECRET: string;
};

const app = new Hono<{ Bindings: Env }>();
type AppContext = Context<{ Bindings: Env }>;

app.get("/healthz", (c) => c.json({ status: "ok", component: "edge-worker" }));

app.post("/charges", async (c) => forwardWrite(c));
app.post("/charges/:id/capture", async (c) => forwardWrite(c));
app.post("/refunds", async (c) => forwardWrite(c));
app.post("/webhooks/:psp", async (c) => {
  const body = await c.req.text();
  if (!c.req.header("x-routekit-sandbox-event-id")) {
    return c.json({ error: "missing webhook event id" }, 422);
  }
  return proxy(c, body);
});

async function forwardWrite(c: AppContext): Promise<Response> {
  const idemKey = c.req.header("idempotency-key");
  if (!idemKey) return c.json({ error: "Idempotency-Key is required" }, 400);

  const body = await c.req.text();
  let parsed: unknown;
  try {
    parsed = body ? JSON.parse(body) : {};
  } catch {
    return c.json({ error: "invalid JSON body" }, 400);
  }

  const rawCardError = rejectRawCardData(parsed);
  if (rawCardError) return c.json({ error: rawCardError }, 422);

  const validSignature = await verifyMerchantSignature(c.env.ROUTEKIT_HMAC_SECRET, c.req.raw, body);
  if (!validSignature) return c.json({ error: "invalid request signature" }, 401);

  const bodyHash = await sha256Hex(body);
  const kvKey = `idem:${idemKey}`;
  const cached = await c.env.IDEMPOTENCY.get(kvKey, "json") as IdempotencyRecord | null;
  if (cached) {
    if (cached.bodyHash !== bodyHash) return c.json({ error: "idempotency key body mismatch" }, 422);
    return new Response(cached.responseBody, {
      status: cached.responseStatus,
      headers: { "content-type": "application/json", "x-routekit-idempotent-replay": "true" }
    });
  }

  const response = await proxy(c, body);
  const responseBody = await response.clone().text();
  if (response.status < 500) {
    await c.env.IDEMPOTENCY.put(kvKey, JSON.stringify({
      bodyHash,
      responseStatus: response.status,
      responseBody
    }), { expirationTtl: 86400 });
  }
  return new Response(responseBody, response);
}

async function proxy(c: AppContext, body: string): Promise<Response> {
  const url = new URL(c.req.url);
  const upstream = new URL(url.pathname, c.env.ORCHESTRATOR_URL);
  const headers = new Headers(c.req.raw.headers);
  headers.set("x-routekit-edge-auth", c.env.ORCHESTRATOR_SHARED_SECRET);
  return fetch(upstream, {
    method: c.req.method,
    headers,
    body
  });
}

export default app;
