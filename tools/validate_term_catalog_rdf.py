#!/usr/bin/env python3
"""Independently prove the generated term catalog covers adopted RDF declarations.

The catalog generator is deliberately simple; this audit uses RDFLib instead of
its textual extraction logic so formatting changes cannot silently hide terms.
"""

from __future__ import annotations

import json
from pathlib import Path

from rdflib import Graph, RDF, URIRef
from rdflib.namespace import OWL

ROOT = Path(__file__).resolve().parents[1]
CATALOG = json.loads((ROOT / "catalog" / "terms.json").read_text(encoding="utf-8"))
NAMESPACES = json.loads((ROOT / "catalog" / "namespaces.json").read_text(encoding="utf-8"))

SOURCE_FILES = {
    "commons": ROOT / "representations" / "commons.ttl",
    "governance": ROOT / "representations" / "governance.ttl",
}

DECLARATION_TYPES = {
    OWL.Class,
    OWL.ObjectProperty,
    OWL.DatatypeProperty,
    OWL.AnnotationProperty,
}


def declared_terms(namespace_iri: str, source: Path) -> set[str]:
    graph = Graph().parse(source.as_posix(), format="turtle")
    prefix = namespace_iri
    terms: set[str] = set()
    for subject, rdf_type in graph.subject_objects(RDF.type):
        if not isinstance(subject, URIRef):
            continue
        iri = str(subject)
        if not iri.startswith(prefix):
            continue
        # Include standard OWL declarations and domain-specific typed vocabulary
        # individuals such as Governance roles/decision classes.
        if rdf_type in DECLARATION_TYPES or isinstance(rdf_type, URIRef):
            terms.add(iri)
    return terms


def main() -> None:
    adopted = {
        entry["id"]: entry
        for entry in NAMESPACES.get("namespaces", [])
        if entry.get("status") == "adopted"
    }
    catalog_namespaces = CATALOG.get("namespaces", {})

    for namespace_id, source in SOURCE_FILES.items():
        entry = adopted.get(namespace_id)
        if entry is None:
            raise SystemExit(f"{namespace_id}: source is published but namespace is not adopted")
        expected = declared_terms(entry["namespace"], source)
        catalog_entry = catalog_namespaces.get(namespace_id)
        if not isinstance(catalog_entry, dict):
            raise SystemExit(f"{namespace_id}: missing generated catalog namespace")
        actual = {
            term.get("iri")
            for term in catalog_entry.get("terms", [])
            if isinstance(term, dict) and isinstance(term.get("iri"), str)
        }
        missing = sorted(expected - actual)
        extra = sorted(actual - expected)
        if missing or extra:
            details = []
            if missing:
                details.append("missing=" + ", ".join(missing))
            if extra:
                details.append("extra=" + ", ".join(extra))
            raise SystemExit(f"{namespace_id}: generated term catalog != RDF declarations: {'; '.join(details)}")
        print(f"{namespace_id}: {len(actual)} catalog terms exactly match RDF declarations")


if __name__ == "__main__":
    main()
