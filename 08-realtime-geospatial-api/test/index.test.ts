import { describe, expect, it } from "vitest";
import { handleRequest, type Env } from "../src/index";

describe("realtime geospatial worker", () => {
  it("returns health without external bindings", async () => {
    const response = await handleRequest(new Request("https://worker.test/healthz"), {});

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toMatchObject({ ok: true, service: "realtime-geospatial-api" });
  });

  it("requires auth for publishing", async () => {
    const response = await handleRequest(new Request("https://worker.test/v1/positions", {
      method: "POST",
      body: JSON.stringify({ group_id: "fleet", entity_id: "driver", lat: 1, lng: 2 })
    }), {});

    expect(response.status).toBe(401);
  });

  it("publishes a single position to the right Durable Object stub", async () => {
    const calls: Array<{ name: string; body: unknown }> = [];
    const env: Env = {
      DEMO_API_KEYS: demoKeys(),
      GROUP_HUB: fakeNamespace(calls)
    };

    const response = await handleRequest(new Request("https://worker.test/v1/positions", {
      method: "POST",
      headers: {
        authorization: "Bearer pk_live_demo",
        "content-type": "application/json"
      },
      body: JSON.stringify({ group_id: "fleet", entity_id: "driver-1", lat: 51.51, lng: -0.118 })
    }), env);

    expect(response.status).toBe(204);
    expect(calls).toHaveLength(1);
    expect(calls[0].name).toBe("tenant_demo::fleet");
    expect(calls[0].body).toMatchObject({
      items: [{ group_id: "fleet", entity_id: "driver-1", lat: 51.51, lng: -0.118 }]
    });
  });

  it("mints subscribe tokens for admin keys", async () => {
    const response = await handleRequest(new Request("https://worker.test/v1/subscribe-token", {
      method: "POST",
      headers: {
        authorization: "Bearer pk_live_demo",
        "content-type": "application/json"
      },
      body: JSON.stringify({ group_id: "fleet", ttl_s: 60 })
    }), {
      DEMO_API_KEYS: demoKeys(),
      SUBSCRIBE_JWT_SECRET: "secret"
    });

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toMatchObject({ expires_at: expect.any(Number), token: expect.any(String) });
  });

  it("rejects websocket subscribe attempts without upgrade", async () => {
    const tokenResponse = await handleRequest(new Request("https://worker.test/v1/subscribe-token", {
      method: "POST",
      headers: {
        authorization: "Bearer pk_live_demo",
        "content-type": "application/json"
      },
      body: JSON.stringify({ group_id: "fleet", ttl_s: 60 })
    }), {
      DEMO_API_KEYS: demoKeys(),
      SUBSCRIBE_JWT_SECRET: "secret"
    });
    const { token } = await tokenResponse.json() as { token: string };

    const response = await handleRequest(new Request(`https://worker.test/v1/groups/fleet/ws?token=${token}`), {
      GROUP_HUB: fakeNamespace([]),
      SUBSCRIBE_JWT_SECRET: "secret"
    });

    expect(response.status).toBe(426);
  });
});

function demoKeys(): string {
  return JSON.stringify({
    pk_live_demo: {
      tenant_id: "tenant_demo",
      plan: "free",
      ccu_cap: 200,
      write_rps_cap: 100,
      scopes: ["publish", "admin"]
    }
  });
}

function fakeNamespace(calls: Array<{ name: string; body: unknown }>): DurableObjectNamespace {
  return {
    idFromName(name: string) {
      return { toString: () => name } as DurableObjectId;
    },
    get(id: DurableObjectId) {
      const name = id.toString();
      return {
        fetch: async (input: RequestInfo | URL, init?: RequestInit) => {
          const request = input instanceof Request ? input : new Request(input, init);
          calls.push({ name, body: request.method === "POST" ? await request.json() : undefined });
          return new Response(JSON.stringify({ ok: true }), {
            headers: { "content-type": "application/json" }
          });
        }
      } as DurableObjectStub;
    }
  } as DurableObjectNamespace;
}
