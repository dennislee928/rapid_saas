#!/usr/bin/env python3
"""Research-only PSP risk exposure simulation.

The simulator is intentionally dependency-free and isolated from production
services. It compares an analytical classical baseline with samples from a
minimal single-qubit state simulator for each PSP route.
"""

from __future__ import annotations

import argparse
import csv
import json
import math
import random
from dataclasses import dataclass
from pathlib import Path
from statistics import mean


DEFAULT_DATASET = Path(__file__).resolve().parents[1] / "data" / "psp_risk_routes.csv"
DEFAULT_SEED = 20260426
DEFAULT_SHOTS = 4096
CHARGEBACK_LOSS_MULTIPLIER = 1.15


@dataclass(frozen=True)
class RouteRisk:
    route_id: str
    psp: str
    region: str
    traffic_share: float
    success_rate: float
    chargeback_rate: float
    avg_amount: float
    incident_loss: float

    @property
    def adverse_event_probability(self) -> float:
        return clamp_probability(1.0 - self.success_rate + self.chargeback_rate)

    @property
    def adverse_event_loss(self) -> float:
        return self.avg_amount * CHARGEBACK_LOSS_MULTIPLIER + self.incident_loss

    @property
    def weighted_expected_loss(self) -> float:
        return self.traffic_share * self.adverse_event_probability * self.adverse_event_loss


def clamp_probability(value: float) -> float:
    return max(0.0, min(1.0, value))


def read_routes(path: Path) -> list[RouteRisk]:
    with path.open(newline="", encoding="utf-8") as dataset:
        rows = csv.DictReader(dataset)
        routes = [
            RouteRisk(
                route_id=row["route_id"],
                psp=row["psp"],
                region=row["region"],
                traffic_share=float(row["traffic_share"]),
                success_rate=float(row["success_rate"]),
                chargeback_rate=float(row["chargeback_rate"]),
                avg_amount=float(row["avg_amount"]),
                incident_loss=float(row["incident_loss"]),
            )
            for row in rows
        ]

    total_share = sum(route.traffic_share for route in routes)
    if not math.isclose(total_share, 1.0, rel_tol=0.0, abs_tol=0.000001):
        raise ValueError(f"traffic_share must sum to 1.0, got {total_share:.6f}")
    return routes


def classical_expected_loss(routes: list[RouteRisk]) -> float:
    return sum(route.weighted_expected_loss for route in routes)


def ry_state_for_event_probability(probability: float) -> tuple[float, float]:
    """Return amplitudes after Ry(theta)|0>, where Pr(|1>) is probability."""
    probability = clamp_probability(probability)
    theta = 2.0 * math.asin(math.sqrt(probability))
    return (math.cos(theta / 2.0), math.sin(theta / 2.0))


def sample_quantum_route_loss(route: RouteRisk, shots: int, rng: random.Random) -> dict[str, float | int | str]:
    amplitude_zero, amplitude_one = ry_state_for_event_probability(route.adverse_event_probability)
    event_probability = amplitude_one * amplitude_one
    events = sum(1 for _ in range(shots) if rng.random() < event_probability)
    sampled_loss = route.traffic_share * (events / shots) * route.adverse_event_loss
    return {
        "route_id": route.route_id,
        "psp": route.psp,
        "region": route.region,
        "shots": shots,
        "event_count": events,
        "state_amplitude_zero": round(amplitude_zero, 8),
        "state_amplitude_one": round(amplitude_one, 8),
        "encoded_event_probability": round(event_probability, 8),
        "sampled_weighted_loss": round(sampled_loss, 6),
        "classical_weighted_loss": round(route.weighted_expected_loss, 6),
    }


def quantum_simulated_losses(routes: list[RouteRisk], shots: int, seed: int) -> list[dict[str, float | int | str]]:
    rng = random.Random(seed)
    return [sample_quantum_route_loss(route, shots, rng) for route in routes]


def percentile(values: list[float], percentile_rank: float) -> float:
    if not values:
        raise ValueError("cannot calculate percentile for an empty list")
    ordered = sorted(values)
    index = math.ceil((percentile_rank / 100.0) * len(ordered)) - 1
    return ordered[max(0, min(index, len(ordered) - 1))]


def run_experiment(dataset: Path, shots: int, seed: int) -> dict[str, object]:
    routes = read_routes(dataset)
    classical_loss = classical_expected_loss(routes)
    quantum_routes = quantum_simulated_losses(routes, shots, seed)
    quantum_weighted_losses = [float(route["sampled_weighted_loss"]) for route in quantum_routes]
    quantum_mean_loss = sum(quantum_weighted_losses)

    return {
        "experiment": "monte_carlo_psp_risk_exposure",
        "dataset": str(dataset),
        "seed": seed,
        "shots_per_route": shots,
        "production_boundary": "research_only_not_on_production_hot_path",
        "classical_baseline": {
            "expected_weighted_loss": round(classical_loss, 6),
            "route_count": len(routes),
        },
        "quantum_simulator": {
            "sampled_mean_weighted_loss": round(quantum_mean_loss, 6),
            "absolute_delta_from_classical": round(abs(quantum_mean_loss - classical_loss), 6),
            "p95_route_weighted_loss": round(percentile(quantum_weighted_losses, 95), 6),
            "mean_route_weighted_loss": round(mean(quantum_weighted_losses), 6),
            "routes": quantum_routes,
        },
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the research-only PSP risk quantum simulation prototype.")
    parser.add_argument("--dataset", type=Path, default=DEFAULT_DATASET, help="CSV dataset path.")
    parser.add_argument("--shots", type=int, default=DEFAULT_SHOTS, help="Measurement shots per route.")
    parser.add_argument("--seed", type=int, default=DEFAULT_SEED, help="Deterministic RNG seed.")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    if args.shots <= 0:
        raise SystemExit("--shots must be positive")
    result = run_experiment(args.dataset, args.shots, args.seed)
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()

