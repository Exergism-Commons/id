#!/usr/bin/env python3
"""Prove that every published Funding representation is the owner exporter output.

CI checks out Exergism-Commons/funding at the pinned commit, runs its own
tools/export_publication.py and passes both directories here. The resolver must
serve those bytes unchanged: the versioned and current ontology, both immutable
JSON-LD contexts, the frozen historical context alias and every record. Any other
machine-readable representation behind a Funding URL fails the check.
"""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]
BASE = "https://id.exergism.org"
FUNDING_REPOSITORY = "Exergism-Commons/funding"
LEGACY_CONTEXT_ALIAS = "/representations/funding-context.jsonld"
OWNER_ONTOLOGY_SOURCE = "ontology/funding.owl.ttl"

# Every public URL surface that may serve Funding content: route paths, route
# aliases and representation public paths.
FUNDING_SURFACE = re.compile(r"^/(funding|ontology/funding|context/funding)(/|$)|^/representations/funding")


def load(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def git(source: Path, *args: str) -> bytes:
    return subprocess.check_output(["git", "-C", str(source), *args])


def path_of(iri: str) -> str:
    if not iri.startswith(BASE + "/"):
        raise SystemExit(f"{iri} is not an id.exergism.org IRI")
    return iri[len(BASE):]


def single_file(routes: dict[str, dict], path: str, *, immutable: bool) -> str:
    route = routes.get(path)
    if route is None:
        raise SystemExit(f"Funding route {path} is not registered")
    if immutable and "immutable" not in route.get("cache_control", "").lower():
        raise SystemExit(f"Funding route {path} must be immutable")
    representations = route.get("representations", [])
    if len(representations) != 1:
        raise SystemExit(f"Funding route {path} must expose exactly one representation")
    return representations[0]["file"]


def media_file(routes: dict[str, dict], path: str, media_prefix: str) -> str:
    route = routes.get(path)
    if route is None:
        raise SystemExit(f"Funding route {path} is not registered")
    files = [
        representation["file"]
        for representation in route.get("representations", [])
        if representation["media_type"].startswith(media_prefix)
    ]
    if len(files) != 1:
        raise SystemExit(f"Funding route {path} must expose exactly one {media_prefix} representation")
    return files[0]


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True, help="Funding checkout at the pinned commit")
    parser.add_argument("--publication", type=Path, required=True, help="output directory of its export_publication.py")
    args = parser.parse_args()

    publication = load(ROOT / "resolver" / "publications" / "funding.json")
    pin = publication["source_commit"]
    if publication.get("authority") != FUNDING_REPOSITORY:
        raise SystemExit("resolver/publications/funding.json must name the Funding owner repository")

    head = git(args.source, "rev-parse", "HEAD").decode("ascii").strip()
    if head != pin:
        raise SystemExit(f"Funding checkout is at {head}, but the publication is pinned to {pin}")

    namespaces = load(ROOT / "catalog" / "namespaces.json")
    funding = next((entry for entry in namespaces["namespaces"] if entry["id"] == "funding"), None)
    if funding is None or funding.get("status") != "adopted":
        raise SystemExit("Funding must be adopted in catalog/namespaces.json")
    if funding.get("owner_repository") != FUNDING_REPOSITORY or funding.get("source_commit") != pin:
        raise SystemExit("catalog/namespaces.json Funding owner/source commit diverges from the publication pin")

    snapshots = [
        entry
        for entry in load(ROOT / "catalog" / "ontology-snapshots.json")["snapshots"]
        if entry.get("source_repository") == FUNDING_REPOSITORY
    ]
    if not snapshots or any(entry.get("source_commit") != pin for entry in snapshots):
        raise SystemExit("catalog/ontology-snapshots.json Funding entries diverge from the publication pin")

    # The exporter output must be internally consistent and owner-authored.
    exported = args.publication
    manifest = load(exported / "manifest.json")
    for key in ("publication_version", "ontology_version_iri", "contexts"):
        if manifest.get(key) != publication.get(key):
            raise SystemExit(f"exporter manifest {key} differs from resolver/publications/funding.json")
    if funding.get("ontology_version") != manifest["publication_version"] or funding.get("version_iri") != manifest["ontology_version_iri"]:
        raise SystemExit("catalog/namespaces.json Funding version differs from the exporter manifest")

    exact = {entry["output_path"]: entry for entry in manifest["exact_artifacts"]}
    records = {entry["stable_id"]: entry for entry in manifest["records"]}
    declared = set(exact) | {entry["output_path"] for entry in records.values()}
    produced = {path.relative_to(exported).as_posix() for path in exported.rglob("*") if path.is_file()}
    if produced != declared | {"manifest.json"}:
        raise SystemExit(f"exporter output and manifest disagree: {sorted(produced ^ (declared | {'manifest.json'}))}")
    for output_path in declared:
        entry = exact.get(output_path) or next(item for item in records.values() if item["output_path"] == output_path)
        if hashlib.sha256((exported / output_path).read_bytes()).hexdigest() != entry["sha256"]:
            raise SystemExit(f"exporter output {output_path} does not match its manifest SHA-256")
    for output_path, entry in exact.items():
        if (exported / output_path).read_bytes() != git(args.source, "show", f"HEAD:{entry['source_path']}"):
            raise SystemExit(f"exporter output {output_path} is not the owner source blob {entry['source_path']}")

    # Map every Funding representation served by id to the exporter file it must equal.
    routes = {route["path"]: route for route in load(ROOT / "resolver" / "registry.json")["routes"]}
    ontology_output = next(
        (path for path, entry in exact.items() if entry["source_path"] == OWNER_ONTOLOGY_SOURCE), None
    )
    if ontology_output is None:
        raise SystemExit(f"exporter did not publish {OWNER_ONTOLOGY_SOURCE}")

    expected: dict[str, str] = {
        single_file(routes, path_of(publication["ontology_version_iri"]), immutable=True): ontology_output,
        media_file(routes, path_of(publication["ontology_iri"]), "text/turtle"): ontology_output,
    }
    legacy_output = None
    for role, iri in publication["contexts"].items():
        output = f"contexts/funding-context-{iri.rsplit('/', 1)[1]}.jsonld"
        if output not in exact:
            raise SystemExit(f"{role} context {iri} has no exact exporter artifact")
        expected[single_file(routes, path_of(iri), immutable=True)] = output
        if role == "legacy":
            legacy_output = output
    if legacy_output is None:
        raise SystemExit("publication must declare the legacy context")
    expected[single_file(routes, LEGACY_CONTEXT_ALIAS, immutable=True)] = legacy_output

    published_ids = set()
    for iri in publication["records"]:
        stable_id = iri.rsplit("/", 1)[1]
        if stable_id not in records:
            raise SystemExit(f"published record {iri} is not produced by the exporter")
        published_ids.add(stable_id)
        expected[media_file(routes, path_of(iri), "application/ld+json")] = records[stable_id]["output_path"]
    if published_ids != set(records):
        raise SystemExit(f"exporter records not published: {sorted(set(records) - published_ids)}")
    registered_ids = {path.rsplit("/", 1)[1] for path in routes if path.startswith("/funding/id/")}
    if registered_ids != published_ids:
        raise SystemExit(f"registered Funding record routes differ from the publication: {sorted(registered_ids ^ published_ids)}")

    # No machine-readable representation behind a Funding URL may escape the comparison.
    for route in routes.values():
        surfaces = [route["path"], *route.get("aliases", [])]
        surfaces += [rep["public_path"] for rep in route.get("representations", []) if rep.get("public_path")]
        if not any(FUNDING_SURFACE.search(surface) for surface in surfaces):
            continue
        for representation in route.get("representations", []):
            if representation["media_type"].startswith("text/html"):
                continue
            if representation["file"] not in expected:
                raise SystemExit(f"{route['path']} serves {representation['file']}, which is not exporter output")

    for id_file, output_path in sorted(expected.items()):
        if (ROOT / id_file).read_bytes() != (exported / output_path).read_bytes():
            raise SystemExit(f"{id_file} is not byte-for-byte the exporter output {output_path}")

    print(f"Funding publication verified: {len(expected)} representations equal the owner exporter output at {pin}")


if __name__ == "__main__":
    main()
