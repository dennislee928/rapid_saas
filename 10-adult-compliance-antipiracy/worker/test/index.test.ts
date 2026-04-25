import { describe, expect, it, vi } from "vitest";
import { signJwt, verifyJwt } from "../src/crypto";
import { handleRequest, type Env } from "../src/index";

const env: Env = {
  AEGIS_ENV: "test",
  PLATFORM_HMAC_SECRET: "platform-test-secret-with-enough-bytes",
  DEFAULT_TENANT_SECRET: "tenant-test-secret-with-enough-bytes",
};

describe("GateKeep Worker", () => {
  it("signs and verifies HS256 JWTs without PII claims", async () => {
    const now = Math.floor(Date.now() / 1000);
    const token = await signJwt(
      {
        iss: "gatekeep.aegis.adult",
        aud: "tenant_demo",
        sub: "session_test",
        iat: now,
        exp: now + 60,
        kid: "k_2026_04",
        vrf: "persona",
        vid_hash: "hash",
        ag: true,
        geo: "GB",
        v: 1,
      },
      env.DEFAULT_TENANT_SECRET,
    );

    const payload = await verifyJwt(token, env.DEFAULT_TENANT_SECRET, "tenant_demo");
    expect(payload.ag).toBe(true);
    expect(JSON.stringify(payload)).not.toMatch(/dob|email|document|name/i);
  });

  it("redirects missing JWT sessions to hosted verify", async () => {
    const response = await handleRequest(
      new Request("https://customer.example/video", {
        headers: { accept: "text/html", "user-agent": "vitest" },
      }),
      env,
    );

    expect(response.status).toBe(302);
    expect(response.headers.get("location")).toContain("/verify");
    expect(response.headers.get("set-cookie")).toContain("gk_state=");
  });

  it("blocks bot-like requests before proxying", async () => {
    const response = await handleRequest(new Request("https://customer.example/video"), env);
    expect(response.status).toBe(403);
  });

  it("injects CSP on valid proxied HTML", async () => {
    const now = Math.floor(Date.now() / 1000);
    const token = await signJwt(
      {
        iss: "gatekeep.aegis.adult",
        aud: "tenant_demo",
        sub: "session_test",
        iat: now,
        exp: now + 60,
        kid: "k_2026_04",
        vrf: "persona",
        vid_hash: "hash",
        ag: true,
        v: 1,
      },
      env.DEFAULT_TENANT_SECRET,
    );
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("<h1>origin</h1>", { headers: { "content-type": "text/html" } })),
    );

    const response = await handleRequest(
      new Request("https://customer.example/video", {
        headers: {
          accept: "text/html",
          "user-agent": "vitest",
          cookie: `gk_session=${token}`,
          "x-aegis-origin": "https://origin.example.test",
        },
      }),
      env,
    );

    expect(response.status).toBe(200);
    expect(response.headers.get("content-security-policy")).toContain("frame-ancestors");
    vi.unstubAllGlobals();
  });
});

describe("Reclaim Worker", () => {
  it("returns an upload hash workflow stub without persisting content", async () => {
    const response = await handleRequest(
      new Request("https://aegis.example/api/reclaim/upload", { method: "POST", body: "sample" }),
      env,
    );
    const body = await response.json() as { workflow: string; source_retention: string };
    expect(body.workflow).toBe("queued_for_hasher");
    expect(body.source_retention).toBe("delete_original_within_24h");
  });

  it("renders takedown drafts with human review default", async () => {
    const response = await handleRequest(
      new Request("https://aegis.example/api/reclaim/takedown-draft", {
        method: "POST",
        body: JSON.stringify({ candidateUrl: "https://leak.example/post/1", matchScore: 0.97 }),
      }),
      env,
    );
    const body = await response.json() as { auto_send: boolean; draft: string; status: string };
    expect(body.status).toBe("draft_requires_human_review");
    expect(body.auto_send).toBe(false);
    expect(body.draft).toContain("not legal advice");
  });
});

