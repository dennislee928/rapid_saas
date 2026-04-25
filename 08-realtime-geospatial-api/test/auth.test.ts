import { describe, expect, it } from "vitest";
import { authenticateApiKey, clampAllowedBbox, mintSubscribeToken, verifySubscribeToken } from "../src/auth";

describe("auth helpers", () => {
  it("authenticates configured demo api keys", async () => {
    const principal = await authenticateApiKey(new Request("https://worker.test", {
      headers: { authorization: "Bearer pk_live_demo" }
    }), {
      DEMO_API_KEYS: JSON.stringify({
        pk_live_demo: {
          tenant_id: "tenant_demo",
          plan: "free",
          ccu_cap: 200,
          write_rps_cap: 100,
          scopes: ["publish", "admin"]
        }
      })
    });

    expect(principal).toMatchObject({ tenant_id: "tenant_demo", scopes: ["publish", "admin"] });
  });

  it("mints and verifies subscribe tokens", async () => {
    const { token } = await mintSubscribeToken({
      tenant_id: "tenant_demo",
      group_id: "fleet",
      ttl_s: 60
    }, "secret");

    await expect(verifySubscribeToken(token, "secret")).resolves.toMatchObject({
      tenant_id: "tenant_demo",
      group_id: "fleet"
    });
    await expect(verifySubscribeToken(token, "wrong")).rejects.toThrow(/signature/);
  });

  it("intersects requested bbox with allowed token bbox", () => {
    expect(clampAllowedBbox([0, 0, 10, 10], [5, 5, 20, 20])).toEqual([5, 5, 10, 10]);
  });
});
