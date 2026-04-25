import type { VisitTokenPayload } from "../shared/types";

const encoder = new TextEncoder();
const decoder = new TextDecoder();

export async function signVisitToken(payload: VisitTokenPayload, secret: string): Promise<string> {
  const body = base64UrlEncode(encoder.encode(JSON.stringify(payload)));
  const signature = await hmac(body, secret);
  return `${body}.${signature}`;
}

export async function verifyVisitToken(token: string, secret: string, now = Date.now()): Promise<VisitTokenPayload> {
  const [body, signature] = token.split(".");
  if (!body || !signature) {
    throw new Error("Malformed visit token");
  }

  const expected = await hmac(body, secret);
  if (!timingSafeEqual(signature, expected)) {
    throw new Error("Invalid visit token signature");
  }

  const payload = JSON.parse(decoder.decode(base64UrlDecode(body))) as VisitTokenPayload;
  if (payload.expires_at < now) {
    throw new Error("Visit token expired");
  }
  return payload;
}

async function hmac(payload: string, secret: string): Promise<string> {
  const key = await crypto.subtle.importKey("raw", encoder.encode(secret), { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
  const bytes = new Uint8Array(await crypto.subtle.sign("HMAC", key, encoder.encode(payload)));
  return base64UrlEncode(bytes);
}

function base64UrlEncode(bytes: Uint8Array): string {
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replaceAll("=", "");
}

function base64UrlDecode(value: string): Uint8Array {
  const padded = value.replaceAll("-", "+").replaceAll("_", "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
  const binary = atob(padded);
  return Uint8Array.from(binary, (char) => char.charCodeAt(0));
}

function timingSafeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) {
    return false;
  }
  let diff = 0;
  for (let i = 0; i < a.length; i += 1) {
    diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }
  return diff === 0;
}
