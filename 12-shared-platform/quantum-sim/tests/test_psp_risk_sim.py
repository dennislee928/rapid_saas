import importlib.util
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).resolve().parents[1] / "experiments" / "psp_risk_sim.py"
SPEC = importlib.util.spec_from_file_location("psp_risk_sim", MODULE_PATH)
psp_risk_sim = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
SPEC.loader.exec_module(psp_risk_sim)


class PspRiskSimulationTest(unittest.TestCase):
    def test_dataset_is_reproducible_and_normalized(self):
        routes = psp_risk_sim.read_routes(psp_risk_sim.DEFAULT_DATASET)

        self.assertEqual(4, len(routes))
        self.assertAlmostEqual(1.0, sum(route.traffic_share for route in routes), places=6)
        self.assertAlmostEqual(2.619532, psp_risk_sim.classical_expected_loss(routes), places=6)

    def test_quantum_state_encodes_event_probability(self):
        probability = 0.125
        amplitude_zero, amplitude_one = psp_risk_sim.ry_state_for_event_probability(probability)

        self.assertAlmostEqual(1.0, amplitude_zero**2 + amplitude_one**2, places=12)
        self.assertAlmostEqual(probability, amplitude_one**2, places=12)

    def test_default_experiment_output_is_deterministic(self):
        result = psp_risk_sim.run_experiment(
            psp_risk_sim.DEFAULT_DATASET,
            psp_risk_sim.DEFAULT_SHOTS,
            psp_risk_sim.DEFAULT_SEED,
        )

        self.assertEqual("research_only_not_on_production_hot_path", result["production_boundary"])
        self.assertEqual(4096, result["shots_per_route"])
        self.assertAlmostEqual(2.619532, result["classical_baseline"]["expected_weighted_loss"], places=6)
        self.assertAlmostEqual(
            2.569721,
            result["quantum_simulator"]["sampled_mean_weighted_loss"],
            places=6,
        )
        self.assertLess(result["quantum_simulator"]["absolute_delta_from_classical"], 0.1)


if __name__ == "__main__":
    unittest.main()

