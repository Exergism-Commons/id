#!/usr/bin/env python3
"""Fail closed while Funding 0.2 exists only as an unregistered resolver candidate."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CANDIDATE = ROOT / "staging" / "funding" / "0.2.0-pre1" / "candidate.json"
REGISTRY = ROOT / "resolver" / "registry.json"
NAMESPACES = ROOT / "catalog" / "namespaces.json"
ACTIVE_PUBLICATION = ROOT / "resolver" / "publications" / "funding.json"

FORBIDDEN_CANDIDATE_PATHS = {
    "/ontology/funding/0.2.0-pre1",
    "/context/funding/0.1.0-draft",
    "/context/funding/0.2.0-pre1",
    "/funding/id/ECF-DEC-MRG-NORMALIZATION-002",
}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-commit", required=True)
    args = parser.parse_args()

    candidate = json.loads(CANDIDATE.read_text(encoding="utf-8"))
    if candidate.get("source_commit") != args.source_commit:
        raise SystemExit("candidate source_commit does not match CI-pinned Funding source")
    if candidate.get("source_state") != "candidate-pr-head":
        raise SystemExit("staging candidate must remain candidate-pr-head before adoption")
    if candidate.get("activation_requires_authoritative_main_commit") is not True:
        raise SystemExit("candidate must require repinning to authoritative Funding main")
    if candidate.get("registered_publicly") is not False:
        raise SystemExit("candidate must declare itself unregistered")

    registry = json.loads(REGISTRY.read_text(encoding="utf-8"))
    registered_paths = {route["path"] for route in registry.get("routes", [])}
    leaked = sorted(FORBIDDEN_CANDIDATE_PATHS & registered_paths)
    if leaked:
        raise SystemExit("candidate routes leaked into public resolver registry: " + ", ".join(leaked))
    for route in registry.get("routes", []):
        for representation in route.get("representations", []):
            if str(representation.get("file", "")).startswith("staging/"):
                raise SystemExit(f"public route {route['path']} serves a staging artifact")

    namespaces = json.loads(NAMESPACES.read_text(encoding="utf-8"))
    funding = next((entry for entry in namespaces.get("namespaces", []) if entry.get("id") == "funding"), None)
    if funding is None:
        raise SystemExit("Funding namespace entry missing")
    if funding.get("status") != "migrating":
        raise SystemExit("Funding namespace must remain migrating while source is a PR-head candidate")

    active = json.loads(ACTIVE_PUBLICATION.read_text(encoding="utf-8"))
    if active.get("source_commit") == args.source_commit:
        raise SystemExit("candidate Funding source is already active in resolver/publications/funding.json")

    print("Funding 0.2 candidate is provenance-pinned, unregistered and fail-closed")


if __name__ == "__main__":
    main()
