import os
import shutil
import subprocess
import zipfile
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


@dataclass
class InferRequest:
    job_id: str
    input_url: str
    output_put_url_template: str | None = None
    model: str = "htdemucs"
    params: dict[str, Any] = field(default_factory=dict)
    callback_url: str | None = None
    hmac_sig: str | None = None


def build_demucs_command(job_dir: Path, request: InferRequest) -> list[str]:
    params = request.params or {}
    command = [
        os.getenv("DEMUX_BINARY", "demucs"),
        "--name",
        request.model,
        "--out",
        str(job_dir / "out"),
        "--segment",
        str(params.get("segment", 7)),
        "--jobs",
        "1",
        "--device",
        "cpu",
    ]
    two_stems = params.get("two_stems")
    if two_stems:
        command.append(f"--two-stems={two_stems}")
    shifts = int(params.get("shifts", 1))
    if shifts > 1:
        command.extend(["--shifts", str(shifts)])
    command.append(str(job_dir / "input.wav"))
    return command


def run_pipeline(request: InferRequest, safe_stub: bool | None = None) -> dict[str, Any]:
    if safe_stub is None:
        safe_stub = os.getenv("SAFE_STUB_MODE", "true").lower() != "false"

    root = Path(os.getenv("JOB_ROOT", "/tmp/audio-stem-jobs"))
    job_dir = root / request.job_id
    job_dir.mkdir(parents=True, exist_ok=True)

    input_wav = job_dir / "input.wav"
    if safe_stub:
        input_wav.write_bytes(b"stub audio input\n")
    else:
        # Real mode is intentionally explicit and still expects the caller to
        # provide local storage/download plumbing around this scaffold.
        raise RuntimeError("real download/ffmpeg/demucs mode is not implemented in the MVP scaffold")

    command = build_demucs_command(job_dir, request)
    stems_dir = job_dir / "out" / request.model / "input"
    stems_dir.mkdir(parents=True, exist_ok=True)

    if safe_stub:
        (stems_dir / "vocals.wav").write_bytes(b"stub vocals\n")
        (stems_dir / "no_vocals.wav").write_bytes(b"stub instrumental\n")
    else:
        subprocess.run(command, check=True)

    zip_path = job_dir / "stems.zip"
    with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_STORED) as archive:
        for stem in sorted(stems_dir.glob("*.wav")):
            archive.write(stem, stem.name)

    return {
        "job_id": request.job_id,
        "safe_stub": safe_stub,
        "demucs_command": command,
        "output_zip_key": f"out/{request.job_id}/stems.zip",
        "local_zip": str(zip_path),
        "bytes": zip_path.stat().st_size,
    }


def cleanup_job(job_id: str) -> None:
    root = Path(os.getenv("JOB_ROOT", "/tmp/audio-stem-jobs"))
    shutil.rmtree(root / job_id, ignore_errors=True)

