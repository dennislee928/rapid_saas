import { describe, expect, it } from "vitest";
import { handleRequest, type Env, type QueueEvent } from "../src/worker/index";

describe("tiltguard worker", () => {
  it("returns health", async () => {
    const response = await handleRequest(new Request("https://api.test/healthz"), {});
    await expect(response.json()).resolves.toMatchObject({ ok: true, service: "tiltguard-worker" });
  });

  it("collects a visit token and scores it", async () => {
    const sent: QueueEvent[] = [];
    const env: Env = {
      TOKEN_SECRET: "test-secret",
      EVENTS_QUEUE: {
        send: async (message: QueueEvent) => {
          sent.push(message);
        }
      } as unknown as Queue<QueueEvent>
    };

    const collectResponse = await handleRequest(new Request("https://api.test/collect", {
      method: "POST",
      body: JSON.stringify({
        tenant_id: "ten_demo",
        signals: {
          timezone: "Asia/Shanghai",
          webdriver: true,
          canvasHash: "canvas",
          webglRenderer: "SwiftShader",
          behaviour: { mouseEvents: 0 }
        }
      })
    }), env);

    expect(collectResponse.status).toBe(202);
    const collected = await collectResponse.json() as { visit_token: string; device_id: string };
    expect(collected.device_id).toMatch(/^dev_/);

    const scoreResponse = await handleRequest(new Request("https://api.test/v1/score", {
      method: "POST",
      headers: {
        authorization: "Bearer dev_tiltguard_demo_key",
        "content-type": "application/json"
      },
      body: JSON.stringify({
        visit_token: collected.visit_token,
        account_id: "ring-42",
        event_type: "signup",
        context: { email_domain: "mailinator.com" },
        client: { country: "GB", is_datacenter: true }
      })
    }), env);

    expect(scoreResponse.status).toBe(200);
    const scored = await scoreResponse.json() as { recommendation: string; reason_codes: Array<{ code: string }> };
    expect(scored.recommendation).toBe("deny");
    expect(scored.reason_codes.map((reason) => reason.code)).toContain("WEBDRIVER_PRESENT");
    expect(sent.map((event) => event.type)).toEqual(["collect", "score"]);
  });

  it("rejects score calls without a bearer API key", async () => {
    const response = await handleRequest(new Request("https://api.test/v1/score", { method: "POST", body: "{}" }), {});
    expect(response.status).toBe(401);
  });
});
