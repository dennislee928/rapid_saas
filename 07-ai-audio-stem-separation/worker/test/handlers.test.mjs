import assert from "node:assert/strict";
import test from "node:test";
import { route, validateUpload } from "../src/handlers.mjs";

test("validateUpload accepts MVP formats within limits", () => {
  const error = validateUpload({ filename: "song.wav", bytes: 1024, duration_seconds: 120 }, {});
  assert.equal(error, null);
});

test("validateUpload rejects large files and long audio", () => {
  assert.equal(validateUpload({ filename: "song.wav", bytes: 60 * 1024 * 1024, duration_seconds: 120 }, {}), "file_too_large");
  assert.equal(validateUpload({ filename: "song.wav", bytes: 1024, duration_seconds: 601 }, {}), "duration_too_long");
});

test("POST /api/upload-url returns signed upload stub", async () => {
  const response = await route(new Request("https://example.com/api/upload-url", {
    method: "POST",
    body: JSON.stringify({ filename: "mix.mp3", bytes: 2048, duration_seconds: 30 }),
  }));
  assert.equal(response.status, 200);
  const body = await response.json();
  assert.equal(body.method, "PUT");
  assert.match(body.input_r2_key, /^in\//);
});

test("GET /api/download returns signed download stub", async () => {
  const response = await route(new Request("https://example.com/api/download/job_123/stems.zip"));
  assert.equal(response.status, 200);
  const body = await response.json();
  assert.equal(body.job_id, "job_123");
  assert.equal(body.asset, "stems.zip");
});

