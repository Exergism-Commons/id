#!/usr/bin/env python3
"""Fail closed while Funding 0.2 exists only as an unregistered resolver candidate."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
CANDIDATE = ROOT / "staging" / "funding" / "0.2.0-pre1" / "candidate.json"
REGISTRY = ROOT / "resolver" / "registry.json"
NAMESPACES = ROOT / "catalog" / "namespaces.json"
ACTIVE_PUBLICATION = ROOT / "resolver" / "publications" / "funding.json"

# candidate-pr-head stages an unmerged Funding PR; adopted-main stages a commit
# that CI proves is on funding main. Neither state may register anything.
ALLOWED_SOURCE_STATES = {"candidate-pr-head", "adopted-main"}

FORBIDDEN_CANDIDATE_PATHS = {
    "/ontology/funding/0.2.0-pre1",
    "/context/funding/0.1.0-draft",
    "/context/funding/0.2.0-pre1",
    "/funding/id/ECF-DEC-MRG-NORMALIZATION-002",
}

# Every public URL surface that may serve Funding content: route paths, route
# aliases and representation public paths.
FUNDING_SURFACE = re.compile(r"^/(funding|ontology/funding|context/funding)(/|$)|^/representations/funding")

# Exact bytes of the active Funding 0.1 publication served by those routes.
# Until activation replaces this guard, no machine-readable Funding
# representation may change, appear or disappear behind an existing route.
ACTIVE_FUNDING_REPRESENTATIONS = {
    "representations/funding-context.jsonld": "9a67a50bfcb85ee0cf08d5e93d29334f7698dc7b8351723f97f23c77dc073655",
    "representations/funding-records/ECF-DEC-MRG-BOOTSTRAP-001.jsonld": "04c0946e924ac9a23ba476409c43dbc1775ad924cd5d51424cd9dd03be605725",
    "representations/funding-records/ECF-OPP-COST-OPEN-CALL-2026.jsonld": "f5345f3c8ede3ce0a0a1692c5b20281b6439c0d4dc65ea58a46df095f1af0e5f",
    "representations/funding-records/ECF-OPP-DC-EDIC-ENGAGEMENT.jsonld": "e60fdd1811bc645fa89026fb033e75d6699280406b6c40a7c88b0e95f0911284",
    "representations/funding-records/ECF-OPP-MSCA-PF-2027.jsonld": "2b0e185bc2f92371cc59b15ef2335e92cd48effefb86206454e6e5efb261ccd0",
    "representations/funding-records/ECF-OPP-NLNET-2026-09.jsonld": "15b62ffcf6bffd4e75d7047da309ca6e3c3fa7a2acdec24fe67f30ef33bb693f",
    "representations/funding-records/ECF-STATE-PHASE-BOOTSTRAP-2026.jsonld": "c00adfe0f79528c4e3ab87ca8146e16849ffdde1d8c866c2e161615a580ff49f",
    "representations/funding.owl.ttl": "f92cd793f8724867ec35de3f0c7f58262fd1f1538e8ba8c064cbb275ce416294",
}


def public_surfaces(route: dict) -> list[str]:
    surfaces = [route["path"], *route.get("aliases", [])]
    surfaces.extend(
        representation["public_path"]
        for representation in route.get("representations", [])
        if representation.get("public_path")
    )
    return surfaces


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-commit", required=True)
    args = parser.parse_args()

    candidate = json.loads(CANDIDATE.read_text(encoding="utf-8"))
    if candidate.get("source_commit") != args.source_commit:
        raise SystemExit("candidate source_commit does not match CI-pinned Funding source")
    if candidate.get("source_state") not in ALLOWED_SOURCE_STATES:
        raise SystemExit(f"staging candidate source_state must be one of {sorted(ALLOWED_SOURCE_STATES)}")
    if candidate.get("activation_requires_authoritative_main_commit") is not True:
        raise SystemExit("candidate must require repinning to authoritative Funding main")
    if candidate.get("registered_publicly") is not False:
        raise SystemExit("candidate must declare itself unregistered")

    registry = json.loads(REGISTRY.read_text(encoding="utf-8"))
    served: dict[str, str] = {}
    for route in registry.get("routes", []):
        surfaces = public_surfaces(route)
        leaked = sorted(FORBIDDEN_CANDIDATE_PATHS.intersection(surfaces))
        if leaked:
            raise SystemExit(f"candidate URLs leaked into public resolver route {route['path']}: " + ", ".join(leaked))
        for representation in route.get("representations", []):
            if str(representation.get("file", "")).startswith("staging/"):
                raise SystemExit(f"public route {route['path']} serves a staging artifact")
        if any(FUNDING_SURFACE.search(surface) for surface in surfaces):
            for representation in route.get("representations", []):
                file = representation["file"]
                if not file.endswith(".html"):
                    served[file] = hashlib.sha256((ROOT / file).read_bytes()).hexdigest()

    if served != ACTIVE_FUNDING_REPRESENTATIONS:
        changed = sorted(
            file
            for file in served.keys() | ACTIVE_FUNDING_REPRESENTATIONS.keys()
            if served.get(file) != ACTIVE_FUNDING_REPRESENTATIONS.get(file)
        )
        raise SystemExit("publicly served Funding representations differ from the active 0.1 publication: " + ", ".join(changed))

    namespaces = json.loads(NAMESPACES.read_text(encoding="utf-8"))
    funding = next((entry for entry in namespaces.get("namespaces", []) if entry.get("id") == "funding"), None)
    if funding is None:
        raise SystemExit("Funding namespace entry missing")
    if funding.get("status") != "migrating":
        raise SystemExit("Funding namespace must remain migrating while Funding 0.2 is only staged")

    active = json.loads(ACTIVE_PUBLICATION.read_text(encoding="utf-8"))
    if active.get("source_commit") == args.source_commit:
        raise SystemExit("candidate Funding source is already active in resolver/publications/funding.json")

    print("Funding 0.2 candidate is provenance-pinned, unregistered and fail-closed")


if __name__ == "__main__":
    main()
