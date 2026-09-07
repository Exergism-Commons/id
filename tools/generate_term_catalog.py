#!/usr/bin/env python3
"""Generate/verify the semantic term discovery catalog from published owner bytes.

The catalog is a discoverability projection only. This script deliberately does
not invent semantics: it extracts declarations from the exact Commons and
Governance Turtle representations already provenance-bound to the owner repo.
"""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / "catalog" / "terms.json"
SOURCE_COMMIT = "517c0df4c905c79ec2b2824231b40ccaa6a1b510"

NAMESPACES = {
    "commons": {
        "prefix": "ec",
        "iri_prefix": "https://id.exergism.org/commons#",
        "path": ROOT / "representations" / "commons.ttl",
        "source_path": "ontology/commons.ttl",
        "dependencies": [],
    },
    "governance": {
        "prefix": "ecg",
        "iri_prefix": "https://id.exergism.org/governance#",
        "path": ROOT / "representations" / "governance.ttl",
        "source_path": "ontology/governance.ttl",
        "dependencies": ["commons"],
    },
}

DECLARATION = re.compile(r"^(?P<term>(?:ec|ecg):[A-Za-z0-9_-]+)\s+a\s+(?P<type>[^;.,\s]+)")
LABEL = re.compile(r'rdfs:label\s+"(?P<label>[^"]+)"(?:@[A-Za-z-]+)?')
VERSION = re.compile(r'owl:versionInfo\s+"(?P<version>[^"]+)"')


def parse_namespace(namespace_id: str, config: dict) -> dict:
    text = config["path"].read_text(encoding="utf-8")
    version_match = VERSION.search(text)
    if not version_match:
        raise ValueError(f"{namespace_id}: owl:versionInfo missing")
    version = version_match.group("version")

    terms: list[dict] = []
    seen: set[str] = set()
    lines = text.splitlines()
    for index, line in enumerate(lines):
        stripped = line.strip()
        match = DECLARATION.match(stripped)
        if not match:
            continue
        qname = match.group("term")
        prefix, local_name = qname.split(":", 1)
        if prefix != config["prefix"] or local_name in seen:
            continue
        seen.add(local_name)

        statement = stripped
        cursor = index + 1
        while not statement.rstrip().endswith(".") and cursor < len(lines):
            statement += " " + lines[cursor].strip()
            cursor += 1
        label_match = LABEL.search(statement)
        label = label_match.group("label") if label_match else local_name
        rdf_type = match.group("type")
        terms.append(
            {
                "iri": config["iri_prefix"] + local_name,
                "local_name": local_name,
                "label": label,
                "rdf_type": rdf_type,
                "description": f"{label} in the Exergism Commons {namespace_id} vocabulary.",
                "status": "adopted-pre-1.0",
                "mappings": [],
            }
        )

    terms.sort(key=lambda item: item["iri"])
    return {
        "owner_repository": "Exergism-Commons/governance",
        "source_path": config["source_path"],
        "source_commit": SOURCE_COMMIT,
        "ontology_version": version,
        "dependencies": config["dependencies"],
        "terms": terms,
    }


def generate() -> dict:
    namespaces = {
        namespace_id: parse_namespace(namespace_id, config)
        for namespace_id, config in NAMESPACES.items()
    }
    namespaces.update(
        {
            "funding": {
                "status": "deferred-to-authoritative-publication",
                "reason": "Funding PR #8 is still normalizing ownership against commons/governance; index after the owning source lands.",
            },
            "exergism": {
                "status": "pending-canonical-migration",
                "reason": "Do not index legacy exergia.org IRIs as canonical id.exergism.org terms.",
            },
            "ecl": {
                "status": "pending-canonical-migration",
                "reason": "Do not index urn:ecl terms as canonical id.exergism.org terms before ECL migration lands.",
            },
        }
    )
    return {
        "schema_version": 2,
        "generated": True,
        "authority_boundary": "Discoverability only; owner_repository/source_path in the Namespace Registry remains semantic authority.",
        "namespaces": namespaces,
    }


def serialized() -> str:
    return json.dumps(generate(), indent=2, sort_keys=True, ensure_ascii=False) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    expected = serialized()
    if args.check:
        actual = OUTPUT.read_text(encoding="utf-8")
        if actual != expected:
            raise SystemExit("catalog/terms.json is stale; run tools/generate_term_catalog.py")
        print("term catalog matches authoritative published ontology bytes")
        return
    OUTPUT.write_text(expected, encoding="utf-8")
    print(f"wrote {OUTPUT.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
