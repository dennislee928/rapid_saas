import type { ReasonCode, Recommendation, ScoreRequest, TenantRuleConfig, VisitTokenPayload } from "../shared/types";

export interface ScoreInputs {
  request: ScoreRequest;
  visit: VisitTokenPayload;
  tenantRules: TenantRuleConfig;
  reuse: {
    fingerprintAccounts30d: number;
    clusterAccountCount: number;
    signupCount24h: number;
    ipSignupCount1h: number;
  };
}

export interface RuleScore {
  score: number;
  recommendation: Recommendation;
  reason_codes: ReasonCode[];
}

const DEFAULT_RULES: TenantRuleConfig = {
  allowBelow: 30,
  denyAtOrAbove: 70,
  disposableEmailDomains: ["mailinator.com", "tempmail.test", "10minutemail.com"]
};

export function scoreEvent(input: ScoreInputs): RuleScore {
  const reasons: ReasonCode[] = [];
  const { request, visit, reuse } = input;
  const rules = { ...DEFAULT_RULES, ...input.tenantRules };
  const signals = visit.signals;
  const client = request.client ?? {};

  if (reuse.fingerprintAccounts30d >= 2) {
    reasons.push({
      code: "FP_REUSE_HIGH",
      weight: 35,
      detail: `Fingerprint matches ${reuse.fingerprintAccounts30d} accounts in last 30d`
    });
  }

  if (reuse.clusterAccountCount >= 4) {
    reasons.push({
      code: "CLUSTER_REUSE",
      weight: 25,
      detail: `Device cluster contains ${reuse.clusterAccountCount} accounts`
    });
  }

  if (client.is_datacenter) {
    reasons.push({ code: "IP_DC", weight: 15, detail: "Client IP is marked as datacenter hosting" });
  }

  if (client.is_residential_proxy) {
    reasons.push({ code: "IP_RES_PROXY", weight: 20, detail: "Client IP is marked as residential proxy" });
  }

  if (signals.webdriver) {
    reasons.push({ code: "WEBDRIVER_PRESENT", weight: 40, detail: "navigator.webdriver was true" });
  }

  if ((signals.behaviour?.mouseEvents ?? 1) === 0 || (signals.behaviour?.mouseStraightLineRatio ?? 0) > 0.92) {
    reasons.push({ code: "MOUSE_LINEAR", weight: 15, detail: "Mouse movement was absent or unusually linear" });
  }

  if (client.country && signals.timezone && isTimezoneGeoMismatch(client.country, signals.timezone)) {
    reasons.push({
      code: "TZ_GEO_MISMATCH",
      weight: 12,
      detail: `timezone=${signals.timezone}, ip_country=${client.country}`
    });
  }

  if (client.ja3 && knownBadJa3(client.ja3)) {
    reasons.push({ code: "JA3_KNOWN_BAD", weight: 25, detail: "JA3 is on the demo known-bad list" });
  }

  const emailDomain = request.context?.email_domain?.toLowerCase();
  if (emailDomain && rules.disposableEmailDomains?.includes(emailDomain)) {
    reasons.push({ code: "EMAIL_NEW_DOMAIN", weight: 10, detail: `Disposable or throwaway email domain: ${emailDomain}` });
  }

  if (reuse.signupCount24h > 5) {
    reasons.push({
      code: "VELOCITY_SIGNUP_BURST",
      weight: 25,
      detail: `${reuse.signupCount24h} signups from this fingerprint in 24h`
    });
  }

  if (reuse.ipSignupCount1h > 3) {
    reasons.push({
      code: "IP24_VELOCITY",
      weight: 15,
      detail: `${reuse.ipSignupCount1h} signups from this IP bucket in 1h`
    });
  }

  if (client.country && rules.countryDenylist?.includes(client.country)) {
    reasons.push({ code: "COUNTRY_RULE_DENY", weight: 30, detail: `Tenant rule flags country ${client.country}` });
  }

  const raw = reasons.reduce((sum, reason) => sum + reason.weight, 0);
  const score = Math.min(100, Math.max(0, raw));
  const recommendation = score >= rules.denyAtOrAbove ? "deny" : score < rules.allowBelow ? "allow" : "review";

  return {
    score,
    recommendation,
    reason_codes: reasons.sort((a, b) => b.weight - a.weight).slice(0, 5)
  };
}

function isTimezoneGeoMismatch(country: string, timezone: string): boolean {
  const normalized = country.toUpperCase();
  if (normalized === "GB") {
    return !["Europe/London", "UTC", "Etc/UTC"].includes(timezone);
  }
  if (normalized === "IE") {
    return !["Europe/Dublin", "Europe/London", "UTC", "Etc/UTC"].includes(timezone);
  }
  return false;
}

function knownBadJa3(ja3: string): boolean {
  return new Set(["e7d705a3286e19ea42f587b344ee6865", "bad-demo-ja3"]).has(ja3.toLowerCase());
}
