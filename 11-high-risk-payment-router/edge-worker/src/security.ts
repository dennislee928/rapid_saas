export type IdempotencyRecord = {
  bodyHash: string;
  responseStatus: number;
  responseBody: string;
};

const forbiddenFieldNames = new Set([
  "card_number",
  "cardNumber",
  "pan",
  "primary_account_number",
  "cvv",
  "cvc",
  "expiry",
  "expiration"
]);

const rawPanLike = /\b[0-9][0-9 -]{11,22}[0-9]\b/;

export async function sha256Hex(input: string): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(input));
  return [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

export async function verifyMerchantSignature(secret: string, request: Request, body: string): Promise<boolean> {
  const sig = request.headers.get("x-sig") ?? "";
  const match = /^t=(\d+),v1=([a-f0-9]{64})$/i.exec(sig);
  if (!match) return false;
  const timestamp = Number(match[1]);
  if (!Number.isFinite(timestamp)) return false;
  if (Math.abs(Date.now() / 1000 - timestamp) > 300) return false;
  const path = new URL(request.url).pathname;
  const signed = `${timestamp}.${request.method.toUpperCase()}.${path}.${body}`;
  const expected = await hmacHex(secret, signed);
  return timingSafeEqual(expected, match[2].toLowerCase());
}

export function rejectRawCardData(value: unknown): string | null {
  return walk(value, []);
}

function walk(value: unknown, path: string[]): string | null {
  if (typeof value === "string") {
    if (rawPanLike.test(value)) {
      return `raw card-like value is not allowed at ${path.join(".") || "$"}`;
    }
    return null;
  }
  if (Array.isArray(value)) {
    for (let i = 0; i < value.length; i += 1) {
      const err = walk(value[i], [...path, String(i)]);
      if (err) return err;
    }
    return null;
  }
  if (value && typeof value === "object") {
    for (const [key, nested] of Object.entries(value as Record<string, unknown>)) {
      if (forbiddenFieldNames.has(key)) {
        return `raw card field ${key} is not allowed`;
      }
      const err = walk(nested, [...path, key]);
      if (err) return err;
    }
  }
  return null;
}

async function hmacHex(secret: string, payload: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  const sig = await crypto.subtle.sign("HMAC", key, new TextEncoder().encode(payload));
  return [...new Uint8Array(sig)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

function timingSafeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let out = 0;
  for (let i = 0; i < a.length; i += 1) {
    out |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }
  return out === 0;
}

