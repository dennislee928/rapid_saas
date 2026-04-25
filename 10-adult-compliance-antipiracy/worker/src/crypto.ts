export interface JwtPayload {
  iss: string;
  aud: string;
  sub: string;
  iat: number;
  exp: number;
  kid: string;
  vrf: string;
  vid_hash: string;
  ag: boolean;
  geo?: string;
  v: number;
}

const encoder = new TextEncoder();

export function base64UrlEncode(input: ArrayBuffer | string): string {
  const bytes = typeof input === "string" ? encoder.encode(input) : new Uint8Array(input);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replaceAll("=", "");
}

export function base64UrlDecode(input: string): Uint8Array {
  const normalized = input.replaceAll("-", "+").replaceAll("_", "/");
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
  const binary = atob(padded);
  return Uint8Array.from(binary, (char) => char.charCodeAt(0));
}

async function importHmacKey(secret: string): Promise<CryptoKey> {
  return crypto.subtle.importKey("raw", encoder.encode(secret), { name: "HMAC", hash: "SHA-256" }, false, [
    "sign",
    "verify",
  ]);
}

export async function hmacHex(secret: string, value: string): Promise<string> {
  const key = await importHmacKey(secret);
  const signature = await crypto.subtle.sign("HMAC", key, encoder.encode(value));
  return [...new Uint8Array(signature)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

export async function sha256Hex(value: ArrayBuffer | string): Promise<string> {
  const bytes = typeof value === "string" ? encoder.encode(value) : value;
  const digest = await crypto.subtle.digest("SHA-256", bytes);
  return [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

export async function signJwt(payload: JwtPayload, secret: string): Promise<string> {
  const header = { alg: "HS256", typ: "JWT", kid: payload.kid };
  const signingInput = `${base64UrlEncode(JSON.stringify(header))}.${base64UrlEncode(JSON.stringify(payload))}`;
  const key = await importHmacKey(secret);
  const signature = await crypto.subtle.sign("HMAC", key, encoder.encode(signingInput));
  return `${signingInput}.${base64UrlEncode(signature)}`;
}

export async function verifyJwt(token: string, secret: string, expectedAudience: string, now = Math.floor(Date.now() / 1000)): Promise<JwtPayload> {
  const parts = token.split(".");
  if (parts.length !== 3) throw new Error("jwt_malformed");

  const [encodedHeader, encodedPayload, encodedSignature] = parts;
  const header = JSON.parse(new TextDecoder().decode(base64UrlDecode(encodedHeader))) as { alg?: string };
  if (header.alg !== "HS256") throw new Error("jwt_alg_unsupported");

  const key = await importHmacKey(secret);
  const valid = await crypto.subtle.verify(
    "HMAC",
    key,
    base64UrlDecode(encodedSignature),
    encoder.encode(`${encodedHeader}.${encodedPayload}`),
  );
  if (!valid) throw new Error("jwt_signature_invalid");

  const payload = JSON.parse(new TextDecoder().decode(base64UrlDecode(encodedPayload))) as JwtPayload;
  if (payload.iss !== "gatekeep.aegis.adult") throw new Error("jwt_issuer_invalid");
  if (payload.aud !== expectedAudience) throw new Error("jwt_audience_invalid");
  if (!payload.ag) throw new Error("jwt_age_gate_false");
  if (payload.exp <= now) throw new Error("jwt_expired");
  return payload;
}

