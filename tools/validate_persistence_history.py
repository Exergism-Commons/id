#!/usr/bin/env python3
"""Enforce persistence and immutable-publication history for id.exergism.org."""

from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[1]
REGISTRY = "resolver/registry.json"
NAMESPACES = "catalog/namespaces.json"
SELF = "tools/validate_persistence_history.py"


def git(*args: str) -> str:
    return subprocess.check_output(["git", *args], cwd=ROOT, text=True).strip()


def show(ref: str, path: str) -> str | None:
    result = subprocess.run(
        ["git", "show", f"{ref}:{path}"],
        cwd=ROOT,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
    )
    return result.stdout if result.returncode == 0 else None


def load_json(ref: str, path: str) -> dict | None:
    raw = show(ref, path)
    return json.loads(raw) if raw is not None else None


def route_map(ref: str) -> dict[str, dict]:
    doc = load_json(ref, REGISTRY)
    if doc is None:
        return {}
    return {route["path"]: route for route in doc.get("routes", [])}


def public_paths(routes: dict[str, dict]) -> dict[str, tuple[str, str]]:
    result: dict[str, tuple[str, str]] = {}
    for route in routes.values():
        for rep in route.get("representations", []):
            path = rep.get("public_path")
            if path:
                result[path] = (route["path"], rep["media_type"])
    return result


def namespace_map(ref: str) -> dict[str, dict]:
    doc = load_json(ref, NAMESPACES)
    if doc is None:
        return {}
    return {item["id"]: item for item in doc.get("namespaces", [])}


def immutable(route: dict) -> bool:
    return "immutable" in route.get("cache_control", "").lower()


def representation_bytes(ref: str, route: dict) -> dict[str, str]:
    result: dict[str, str] = {}
    for rep in route.get("representations", []):
        raw = show(ref, rep["file"])
        if raw is None:
            raise AssertionError(f"{ref}: missing representation {rep['file']} for {route['path']}")
        result[rep["file"]] = raw
    return result


def compare(base: str, head: str) -> None:
    before = route_map(base)
    after = route_map(head)

    for path, old in before.items():
        if path not in after:
            raise AssertionError(f"published resolver route disappeared: {path}")
        new = after[path]
        if new.get("canonical") != old.get("canonical"):
            raise AssertionError(f"published route {path} changed canonical IRI")

        old_aliases = set(old.get("aliases", []))
        new_aliases = set(new.get("aliases", []))
        if not old_aliases.issubset(new_aliases):
            raise AssertionError(f"published aliases removed from {path}: {sorted(old_aliases - new_aliases)}")

        if immutable(old):
            if new != old:
                raise AssertionError(f"immutable route metadata changed: {path}")
            if representation_bytes(base, old) != representation_bytes(head, new):
                raise AssertionError(f"immutable route bytes changed: {path}")

    old_public = public_paths(before)
    new_public = public_paths(after)
    for path, identity in old_public.items():
        if path not in new_public:
            raise AssertionError(f"published representation path disappeared: {path}")
        if new_public[path] != identity:
            raise AssertionError(f"published representation path was reassigned: {path}")

    old_namespaces = namespace_map(base)
    new_namespaces = namespace_map(head)
    for ns_id, old in old_namespaces.items():
        if old.get("status") != "adopted":
            continue
        if ns_id not in new_namespaces:
            raise AssertionError(f"adopted namespace disappeared: {ns_id}")
        new = new_namespaces[ns_id]
        for field in ("namespace", "ontology", "owner_repository"):
            if new.get(field) != old.get(field):
                raise AssertionError(f"adopted namespace {ns_id} changed persistent {field}")


def contract_exists(ref: str) -> bool:
    return show(ref, SELF) is not None


def main() -> None:
    base = os.environ.get("EC_ID_HISTORY_BASE", "").strip()
    if not base or set(base) == {"0"}:
        print("no trusted ID history base supplied; persistence history check skipped")
        return
    git("cat-file", "-e", f"{base}^{{commit}}")

    if not contract_exists(base):
        compare(base, "HEAD")
        print("ID persistence history contract bootstrap transition verified")
        return

    commits = git("rev-list", "--reverse", "--ancestry-path", f"{base}..HEAD").splitlines()
    previous = base
    for commit in commits:
        compare(previous, commit)
        previous = commit
    print(f"ID persistence history verified across {len(commits)} committed transition(s)")


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"ID persistence validation failed: {exc}", file=sys.stderr)
        raise
