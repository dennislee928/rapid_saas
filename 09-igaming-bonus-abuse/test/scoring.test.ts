import { describe, expect, it } from "vitest";
import { scoreEvent } from "../src/worker/scoring";
import type { ScoreRequest, VisitTokenPayload } from "../src/shared/types";

describe("rule-based scoring", () => {
  const visit: VisitTokenPayload = {
    tenant_id: "ten_demo",
    visit_id: "vis_1",
    device_id: "dev_1",
    fingerprint_hash: "abc",
    issued_at: Date.now(),
    expires_at: Date.now() + 1000,
    signals: { timezone: "Asia/Shanghai", webdriver: true, behaviour: { mouseEvents: 0 } }
  };

  it("denies obvious automation and reuse with reason codes", () => {
    const request: ScoreRequest = {
      visit_token: "token",
      account_id: "ring-1",
      event_type: "signup",
      context: { email_domain: "mailinator.com" },
      client: { country: "GB", is_datacenter: true, ja3: "bad-demo-ja3" }
    };

    const result = scoreEvent({
      request,
      visit,
      tenantRules: { allowBelow: 30, denyAtOrAbove: 70, disposableEmailDomains: ["mailinator.com"] },
      reuse: { fingerprintAccounts30d: 4, clusterAccountCount: 5, signupCount24h: 8, ipSignupCount1h: 5 }
    });

    expect(result.recommendation).toBe("deny");
    expect(result.score).toBe(100);
    expect(result.reason_codes.map((reason) => reason.code)).toContain("WEBDRIVER_PRESENT");
    expect(result.reason_codes).toHaveLength(5);
  });

  it("allows low-signal legitimate looking traffic", () => {
    const result = scoreEvent({
      request: { visit_token: "token", account_id: "acct_1", event_type: "login", client: { country: "GB" } },
      visit: { ...visit, signals: { timezone: "Europe/London", webdriver: false, behaviour: { mouseEvents: 12 } } },
      tenantRules: { allowBelow: 30, denyAtOrAbove: 70 },
      reuse: { fingerprintAccounts30d: 0, clusterAccountCount: 1, signupCount24h: 1, ipSignupCount1h: 1 }
    });

    expect(result).toMatchObject({ score: 0, recommendation: "allow", reason_codes: [] });
  });
});
