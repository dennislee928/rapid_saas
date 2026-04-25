const apiBase = window.localStorage.getItem("audioStemApiBase") || "http://localhost:8787";
const fileInput = document.querySelector("#file");
const durationInput = document.querySelector("#duration");
const modelInput = document.querySelector("#model");
const log = document.querySelector("#log");

function write(value) {
  log.textContent = typeof value === "string" ? value : JSON.stringify(value, null, 2);
}

document.querySelector("#start").addEventListener("click", async () => {
  const file = fileInput.files?.[0];
  if (!file) {
    write("Choose a file first.");
    return;
  }

  const uploadResponse = await fetch(`${apiBase}/api/upload-url`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      filename: file.name,
      bytes: file.size,
      duration_seconds: Number(durationInput.value),
      content_type: file.type || "application/octet-stream",
    }),
  });
  const upload = await uploadResponse.json();
  if (!uploadResponse.ok) {
    write(upload);
    return;
  }

  const jobResponse = await fetch(`${apiBase}/api/jobs`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      user_id: "user_demo",
      idempotency_key: `${file.name}:${file.size}:${durationInput.value}`,
      model: modelInput.value,
      input_r2_key: upload.input_r2_key,
      input_bytes: file.size,
      input_seconds: Number(durationInput.value),
      params: { two_stems: "vocals", segment: 7, shifts: 1 },
    }),
  });
  const job = await jobResponse.json();
  write({ upload, job });
});

