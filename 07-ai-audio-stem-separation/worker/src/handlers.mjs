const DEFAULT_MAX_BYTES = 50 * 1024 * 1024;
const DEFAULT_MAX_SECONDS = 600;

export function json(data, status = 200, headers = {}) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { "content-type": "application/json; charset=utf-8", ...headers },
  });
}

export async function readJson(request) {
  try {
    return await request.json();
  } catch {
    return null;
  }
}

export function validateUpload(input, env = {}) {
  const maxBytes = Number(env.MAX_UPLOAD_BYTES || DEFAULT_MAX_BYTES);
  const maxSeconds = Number(env.MAX_AUDIO_SECONDS || DEFAULT_MAX_SECONDS);
  const allowed = new Set(["mp3", "wav", "flac", "m4a", "ogg"]);

  if (!input || typeof input !== "object") return "body_required";
  if (!input.filename || typeof input.filename !== "string") return "filename_required";
  if (!Number.isFinite(input.bytes) || input.bytes <= 0) return "bytes_invalid";
  if (input.bytes > maxBytes) return "file_too_large";
  if (!Number.isFinite(input.duration_seconds) || input.duration_seconds <= 0) return "duration_invalid";
  if (input.duration_seconds > maxSeconds) return "duration_too_long";

  const ext = input.filename.split(".").pop()?.toLowerCase();
  if (!allowed.has(ext)) return "format_not_allowed";
  return null;
}

export function buildStubSignedPut(input, now = new Date()) {
  const id = crypto.randomUUID();
  const month = String(now.getUTCMonth() + 1).padStart(2, "0");
  const key = `in/${now.getUTCFullYear()}/${month}/${id}/${input.filename}`;
  return {
    upload_id: id,
    input_r2_key: key,
    method: "PUT",
    url: `https://r2.local.stub/${encodeURIComponent(key)}`,
    expires_in_seconds: 900,
    headers: { "content-type": input.content_type || "application/octet-stream" },
  };
}

export async function handleUploadUrl(request, env) {
  const body = await readJson(request);
  const error = validateUpload(body, env);
  if (error) return json({ error }, 400);
  return json(buildStubSignedPut(body));
}

export async function forwardJson(request, env, path) {
  const base = env.ORCHESTRATOR_URL || "http://localhost:8080";
  const body = await request.text();
  const response = await fetch(`${base}${path}`, {
    method: request.method,
    headers: {
      "content-type": request.headers.get("content-type") || "application/json",
      "x-worker-signature": "stub-hmac",
    },
    body: request.method === "GET" ? undefined : body,
  });
  return new Response(response.body, response);
}

export async function handleDownload(_request, _env, jobId, asset) {
  if (!jobId || !asset) return json({ error: "download_path_invalid" }, 400);
  return json({
    job_id: jobId,
    asset,
    url: `https://r2.local.stub/out/${encodeURIComponent(jobId)}/${encodeURIComponent(asset)}?signed=stub`,
    expires_in_seconds: 3600,
  });
}

export async function handleStripeWebhook(request, env) {
  const event = await readJson(request);
  if (!event?.id || !event?.type) return json({ error: "stripe_event_invalid" }, 400);
  if (!env.ORCHESTRATOR_URL) {
    return json({ received: true, forwarded: false, reason: "orchestrator_not_configured" });
  }
  return forwardJson(new Request(request.url, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(event),
  }), env, "/internal/stripe/event");
}

export async function warmSpace(env) {
  const url = env.HF_SPACE_HEALTH_URL;
  if (!url) return { warmed: false, reason: "health_url_not_configured" };
  const response = await fetch(url, { method: "GET" });
  return { warmed: response.ok, status: response.status };
}

export async function route(request, env = {}) {
  const url = new URL(request.url);
  const path = url.pathname;

  if (request.method === "POST" && path === "/api/upload-url") return handleUploadUrl(request, env);
  if (request.method === "POST" && path === "/api/jobs") return forwardJson(request, env, "/api/jobs");
  if (request.method === "GET" && path.startsWith("/api/jobs/")) return forwardJson(request, env, path);
  if (request.method === "POST" && path === "/api/stripe/webhook") return handleStripeWebhook(request, env);

  const download = path.match(/^\/api\/download\/([^/]+)\/([^/]+)$/);
  if (request.method === "GET" && download) return handleDownload(request, env, download[1], download[2]);

  if (request.method === "POST" && path === "/api/auth/email") {
    const body = await readJson(request);
    if (!body?.email) return json({ error: "email_required" }, 400);
    return json({ sent: true, mode: "stub" });
  }

  if (request.method === "GET" && path === "/healthz") return json({ ok: true, service: "audio-stem-worker" });
  return json({ error: "not_found" }, 404);
}

