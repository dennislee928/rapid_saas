import os
import tempfile
import unittest
import zipfile
from pathlib import Path

from pipeline import InferRequest, build_demucs_command, cleanup_job, run_pipeline


class PipelineTest(unittest.TestCase):
    def test_build_demucs_command_uses_mvp_safe_defaults(self):
        req = InferRequest(job_id="job_test", input_url="https://example.test/input.wav", params={"two_stems": "vocals"})
        command = build_demucs_command(Path("/tmp/job_test"), req)
        self.assertIn("--two-stems=vocals", command)
        self.assertIn("--segment", command)
        self.assertIn("7", command)
        self.assertIn("--device", command)
        self.assertIn("cpu", command)

    def test_run_pipeline_stub_creates_zip_without_demucs(self):
        with tempfile.TemporaryDirectory() as tmp:
            os.environ["JOB_ROOT"] = tmp
            req = InferRequest(job_id="job_stub", input_url="https://example.test/input.wav", params={"two_stems": "vocals"})
            result = run_pipeline(req, safe_stub=True)
            zip_path = Path(result["local_zip"])
            self.assertTrue(zip_path.exists())
            with zipfile.ZipFile(zip_path) as archive:
                self.assertEqual(sorted(archive.namelist()), ["no_vocals.wav", "vocals.wav"])
            cleanup_job("job_stub")


if __name__ == "__main__":
    unittest.main()

