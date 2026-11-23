#!/usr/bin/env python3
"""Prove that every published Funding representation is the owner exporter output.

CI checks out Exergism-Commons/funding at the pinned commit, runs its own
tools/export_publication.py and passes both directories here. The resolver must
serve those bytes unchanged: the versioned and current ontology, both immutable
JSON-LD contexts, the frozen historical context alias and every record.

A historical ontology version (an ontology_snapshots entry with its own older
source_commit) is instead verified against that commit in the same checkout: it
must be an ancestor of the pin and of origin/main, and the published bytes must
be the owner blob. Any other machine-readable representation behind a Funding
URL fails the check.
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


def git_blob_sha(data: bytes) -> str:
    return hashlib.sha1(f"blob {len(data)}\0".encode("ascii") + data).hexdigest()


def is_ancestor(source: Path, commit: str, ref: str) -> bool:
    result = subprocess.run(
        ["git", "-C", str(source), "merge-base", "--is-ancestor", commit, ref],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    return result.returncode == 0


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
    parser.add_argument("--source", type=Path, required=True, help="Funding checkout at the pinned commit, with full history and origin/main")
    parser.add_argument("--publication", type=Path, required=True, help="output directory of its export_publication.py")
    args = parser.parse_args()

    publication = load(ROOT / "resolver" / "publications" / "funding.json")
    pin = publication["source_commit"]
    if publication.get("authority") != FUNDING_REPOSITORY:
        raise SystemExit("resolver/publications/funding.json must name the Funding owner repository")

    head = git(args.source, "rev-parse", "HEAD").decode("ascii").strip()
    if head != pin:
        raise SystemExit(f"Funding checkout is at {head}, but the publication is pinned to {pin}")
    if not is_ancestor(args.source, pin, "origin/main"):
        raise SystemExit(f"pinned Funding commit {pin} is not on Funding origin/main")

    namespaces = load(ROOT / "catalog" / "namespaces.json")
    funding = next((entry for entry in namespaces["namespaces"] if entry["id"] == "funding"), None)
    if funding is None or funding.get("status") != "adopted":
        raise SystemExit("Funding must be adopted in catalog/namespaces.json")
    if funding.get("owner_repository") != FUNDING_REPOSITORY or funding.get("source_commit") != pin:
        raise SystemExit("catalog/namespaces.json Funding owner/source commit diverges from the publication pin")

    # Snapshots at the pin belong to the current publication; any other Funding
    # snapshot must be a verified historical ontology version (checked below).
    snapshots = [
        entry
        for entry in load(ROOT / "catalog" / "ontology-snapshots.json")["snapshots"]
        if entry.get("source_repository") == FUNDING_REPOSITORY
    ]
    current_snapshots = [entry for entry in snapshots if entry.get("source_commit") == pin]
    historical_snapshots = [entry for entry in snapshots if entry.get("source_commit") != pin]
    if not current_snapshots:
        raise SystemExit("catalog/ontology-snapshots.json has no Funding entries at the publication pin")

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

    # Every snapshot at the pin must describe exporter output of its own owner source.
    for entry in current_snapshots:
        paths = [entry["versioned_publication_path"], entry.get("current_publication_path"), entry.get("historical_alias_publication_path")]
        for path in filter(None, paths):
            if path not in expected:
                raise SystemExit(f"snapshot {entry['id']} declares {path}, which is not exporter output")
        if exact.get(expected[entry["versioned_publication_path"]], {}).get("source_path") != entry["source_path"]:
            raise SystemExit(f"snapshot {entry['id']} does not match the exporter artifact for {entry['source_path']}")
        blob = git(args.source, "rev-parse", f"HEAD:{entry['source_path']}").decode("ascii").strip()
        if entry["source_git_blob_sha"] != blob or entry["publication_git_blob_sha"] != blob:
            raise SystemExit(f"snapshot {entry['id']} blob differs from {entry['source_path']} at {pin}")

    # A historical ontology version keeps the exact owner bytes of an older Funding
    # main commit. It is verified against that commit, not against the exporter.
    historical: dict[str, str] = {}
    for entry in historical_snapshots:
        commit, source_path = entry.get("source_commit", ""), entry["source_path"]
        version_iri = entry.get("version_iri")
        if (
            not version_iri
            or entry.get("ontology_iri") != publication["ontology_iri"]
            or any(entry.get(key) for key in ("current_publication_path", "historical_alias_publication_path", "context_iri", "profile_iri"))
            or entry.get("publication_transform") != "none"
        ):
            raise SystemExit(f"historical Funding snapshot {entry['id']} must be an untransformed ontology version only")
        if version_iri == publication["ontology_version_iri"] or entry.get("ontology_version") == manifest["publication_version"]:
            raise SystemExit(f"historical Funding snapshot {entry['id']} claims the current version")
        if not is_ancestor(args.source, commit, "origin/main") or not is_ancestor(args.source, commit, pin):
            raise SystemExit(f"historical Funding snapshot {entry['id']} source {commit} is not an earlier commit of Funding main")
        try:
            source_bytes = git(args.source, "show", f"{commit}:{source_path}")
        except subprocess.CalledProcessError:
            raise SystemExit(f"historical Funding snapshot {entry['id']}: {source_path} does not exist at {commit}") from None
        blob = git_blob_sha(source_bytes)
        if entry["source_git_blob_sha"] != blob or entry["publication_git_blob_sha"] != blob:
            raise SystemExit(f"historical Funding snapshot {entry['id']} blob differs from {source_path} at {commit}")
        file = single_file(routes, path_of(version_iri), immutable=True)
        if file != entry["versioned_publication_path"] or file in expected or file in historical:
            raise SystemExit(f"historical Funding snapshot {entry['id']} route does not serve only its own snapshot file")
        if (ROOT / file).read_bytes() != source_bytes:
            raise SystemExit(f"{file} is not byte-for-byte {source_path} at {commit}")
        historical[file] = f"{commit}:{source_path}"

    # No machine-readable representation behind a Funding URL may escape the comparison.
    for route in routes.values():
        surfaces = [route["path"], *route.get("aliases", [])]
        surfaces += [rep["public_path"] for rep in route.get("representations", []) if rep.get("public_path")]
        if not any(FUNDING_SURFACE.search(surface) for surface in surfaces):
            continue
        for representation in route.get("representations", []):
            if representation["media_type"].startswith("text/html"):
                continue
            if representation["file"] not in expected and representation["file"] not in historical:
                raise SystemExit(f"{route['path']} serves {representation['file']}, which is neither exporter output nor a verified historical version")

    for id_file, output_path in sorted(expected.items()):
        if (ROOT / id_file).read_bytes() != (exported / output_path).read_bytes():
            raise SystemExit(f"{id_file} is not byte-for-byte the exporter output {output_path}")

    print(f"Funding publication verified: {len(expected)} representations equal the owner exporter output at {pin}")
    for id_file, origin in sorted(historical.items()):
        print(f"Historical Funding version verified: {id_file} equals {origin}")


if __name__ == "__main__":
    main()
