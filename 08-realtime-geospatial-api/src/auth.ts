import type { Bbox, SubscribeClaims } from "./protocol";

export interface ApiPrincipal {
  tenant_id: string;
  plan: string;
  ccu_cap: number;
  write_rps_cap: number;
  scopes: string[];
  key_prefix: string;
}

export interface AuthEnv {
  DEMO_API_KEYS?: string;
  SUBSCRIBE_JWT_SECRET?: string;
}

const encoder = new TextEncoder();

export async function authenticateApiKey(request: Request, env: AuthEnv): Promise<ApiPrincipal | null> {
  const header = request.headers.get("authorization");
  const token = header?.match(/^Bearer\s+(.+)$/i)?.[1];
  if (!token) {
    return null;
  }

  const keys = parseDemoKeys(env.DEMO_API_KEYS);
  const principal = keys[token];
  if (!principal?.tenant_id) {
    return null;
  }

  return {
    tenant_id: principal.tenant_id,
    plan: principal.plan ?? "free",
    ccu_cap: positiveInt(principal.ccu_cap, 200),
    write_rps_cap: positiveInt(principal.write_rps_cap, 100),
    scopes: Array.isArray(principal.scopes) ? principal.scopes : ["publish"],
    key_prefix: token.slice(0, 12)
  };
}

export function assertScope(principal: ApiPrincipal, scope: "publish" | "admin", groupId?: string): boolean {
  return principal.scopes.includes(scope)
    || principal.scopes.includes("admin")
    || (scope === "publish" && Boolean(groupId) && principal.scopes.some((item) => item === `publish:group:${groupId}`));
}

export async function mintSubscribeToken(
  claims: Omit<SubscribeClaims, "exp"> & { ttl_s: number },
  secret: string
): Promise<{ token: string; expires_at: number }> {
  const expiresAt = Math.floor(Date.now() / 1000) + Math.max(1, Math.min(claims.ttl_s, 3600));
  const token = await signJwt({ ...claims, exp: expiresAt }, secret);
  return { token, expires_at: expiresAt };
}

export async function verifySubscribeToken(token: string, secret: string): Promise<SubscribeClaims> {
  const [encodedHeader, encodedPayload, encodedSignature] = token.split(".");
  if (!encodedHeader || !encodedPayload || !encodedSignature) {
    throw new Error("Malformed token.");
  }

  const expected = await hmacSha256Base64Url(secret, `${encodedHeader}.${encodedPayload}`);
  if (!timingSafeEqual(encodedSignature, expected)) {
    throw new Error("Invalid token signature.");
  }

  const claims = JSON.parse(base64UrlDecode(encodedPayload)) as SubscribeClaims;
  if (!claims.tenant_id || !claims.group_id || !claims.exp) {
    throw new Error("Token is missing required claims.");
  }
  if (claims.exp <= Math.floor(Date.now() / 1000)) {
    throw new Error("Token expired.");
  }
  return claims;
}

async function signJwt(payload: SubscribeClaims, secret: string): Promise<string> {
  const header = { alg: "HS256", typ: "JWT" };
  const encodedHeader = base64UrlEncode(JSON.stringify(header));
  const encodedPayload = base64UrlEncode(JSON.stringify(payload));
  const signature = await hmacSha256Base64Url(secret, `${encodedHeader}.${encodedPayload}`);
  return `${encodedHeader}.${encodedPayload}.${signature}`;
}

async function hmacSha256Base64Url(secret: string, payload: string): Promise<string> {
  const key = await crypto.subtle.importKey("raw", encoder.encode(secret), { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
  const signature = new Uint8Array(await crypto.subtle.sign("HMAC", key, encoder.encode(payload)));
  return base64UrlEncodeBytes(signature);
}

function parseDemoKeys(raw: string | undefined): Record<string, Partial<ApiPrincipal>> {
  if (!raw) {
    return {};
  }
  return JSON.parse(raw) as Record<string, Partial<ApiPrincipal>>;
}

function positiveInt(value: unknown, fallback: number): number {
  return typeof value === "number" && Number.isInteger(value) && value > 0 ? value : fallback;
}

function base64UrlEncode(value: string): string {
  return base64UrlEncodeBytes(encoder.encode(value));
}

function base64UrlEncodeBytes(bytes: Uint8Array): string {
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replaceAll("=", "");
}

function base64UrlDecode(value: string): string {
  const normalized = value.replaceAll("-", "+").replaceAll("_", "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
  return atob(normalized);
}

function timingSafeEqual(a: string, b: string): boolean {
  const left = encoder.encode(a);
  const right = encoder.encode(b);
  let diff = left.length ^ right.length;
  const length = Math.max(left.length, right.length);
  for (let i = 0; i < length; i += 1) {
    diff |= (left[i] ?? 0) ^ (right[i] ?? 0);
  }
  return diff === 0;
}

export function clampAllowedBbox(requested: Bbox | undefined, allowed: Bbox | undefined): Bbox | undefined {
  if (!requested) {
    return allowed;
  }
  if (!allowed) {
    return requested;
  }
  return [
    Math.max(requested[0], allowed[0]),
    Math.max(requested[1], allowed[1]),
    Math.min(requested[2], allowed[2]),
    Math.min(requested[3], allowed[3])
  ];
}
