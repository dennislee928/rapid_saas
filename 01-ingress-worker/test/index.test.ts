import { describe, expect, it } from "vitest";
import { handleRequest, type Env, type WebhookQueueMessage } from "../src/index";

const encoder = new TextEncoder();

describe("security webhook ingress worker", () => {
  it("returns health without configured bindings", async () => {
    const response = await handleRequest(new Request("https://worker.test/healthz"), {});

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toMatchObject({ ok: true, service: "security-webhook-ingress" });
  });

  it("rejects oversized bodies before signature verification", async () => {
    const response = await handleRequest(new Request("https://worker.test/webhooks/demo", {
      method: "POST",
      body: "too large"
    }), {
      BODY_CAP_BYTES: "3",
      ENDPOINTS_JSON: JSON.stringify({
        demo: { provider: "generic", secret: "secret" }
      })
    });

    expect(response.status).toBe(413);
  });

  it("verifies generic signatures and enqueues the payload", async () => {
    const sent: WebhookQueueMessage[] = [];
    const body = "hello";
    const signature = await hmacHex("secret", body);
    const env: Env = {
      ENDPOINTS_JSON: JSON.stringify({
        demo: { provider: "generic", secret: "secret", queueTopic: "alerts" }
      }),
      SECURITY_WEBHOOK_QUEUE: {
        send: async (message: WebhookQueueMessage) => {
          sent.push(message);
        }
      } as unknown as Queue<WebhookQueueMessage>
    };

    const response = await handleRequest(new Request("https://worker.test/webhooks/demo", {
      method: "POST",
      headers: {
        "x-webhook-signature": `sha256=${signature}`,
        "cf-connecting-ip": "203.0.113.10"
      },
      body
    }), env);

    expect(response.status).toBe(202);
    expect(sent).toHaveLength(1);
    expect(sent[0]).toMatchObject({
      type: "security.webhook.received",
      schema_version: 1,
      endpointId: "demo",
      tenant_id: "tenant_demo",
      tenantId: "tenant_demo",
      provider: "generic",
      bodyBase64: "aGVsbG8=",
      sourceIp: "203.0.113.10",
      topic: "alerts",
      payload: {
        endpointId: "demo",
        tenantId: "tenant_demo",
        provider: "generic",
        bodyBase64: "aGVsbG8="
      }
    });
    expect(sent[0].event_id).toMatch(/[0-9a-f-]{36}/);
  });

  it("accepts the spec /w/:endpointId route alias", async () => {
    const sent: WebhookQueueMessage[] = [];
    const body = JSON.stringify({ ok: true });
    const signature = await hmacHex("secret", body);
    const response = await handleRequest(new Request("https://worker.test/w/demo", {
      method: "POST",
      headers: { "x-webhook-signature": `sha256=${signature}` },
      body
    }), {
      ENDPOINTS_JSON: JSON.stringify({
        demo: { provider: "generic", secret: "secret", tenantId: "tenant_1" }
      }),
      SECURITY_WEBHOOK_QUEUE: {
        send: async (message: WebhookQueueMessage) => {
          sent.push(message);
        }
      } as unknown as Queue<WebhookQueueMessage>
    });

    expect(response.status).toBe(202);
    expect(sent[0]).toMatchObject({
      tenant_id: "tenant_1",
      endpointId: "demo",
      payload: { tenantId: "tenant_1" }
    });
  });
});

async function hmacHex(secret: string, payload: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    encoder.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  const bytes = new Uint8Array(await crypto.subtle.sign("HMAC", key, encoder.encode(payload)));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}
