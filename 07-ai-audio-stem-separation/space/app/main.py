from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from .pipeline import InferRequest, run_pipeline

app = FastAPI(title="AI Audio Stem Separation Space", version="0.1.0")


class InferBody(BaseModel):
    job_id: str
    input_url: str
    output_put_url_template: str | None = None
    model: str = "htdemucs"
    params: dict = Field(default_factory=dict)
    callback_url: str | None = None
    hmac_sig: str | None = None


@app.get("/healthz")
def healthz():
    return {"ok": True, "service": "audio-stem-space", "safe_stub_default": True}


@app.post("/infer")
def infer(body: InferBody):
    try:
        return run_pipeline(InferRequest(**body.model_dump()))
    except Exception as exc:
        raise HTTPException(status_code=500, detail=str(exc)) from exc

