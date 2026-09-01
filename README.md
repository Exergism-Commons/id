# id.exergism.org

Persistent identifier infrastructure for **Exergism Commons**.

Public identifier authority: `https://id.exergism.org/`

This repository contains both the static bootstrap surface currently usable with GitHub Pages and a small native HTTP resolver for production deployment. Public identifiers are deliberately independent of this repository name, GitHub Pages, the VPS provider, and the TLS frontend.

## Resolver architecture

The production resolver is a dependency-free Go service driven by [`resolver/registry.json`](resolver/registry.json).

It provides:

- exact canonical paths such as `/exergism` without forcing a trailing slash;
- HTTP `Accept` content negotiation;
- correct `Content-Type` values for each registered representation;
- `406 Not Acceptable` when a requested representation is not published;
- `Vary: Accept` on negotiated resources;
- canonical and alternate `Link` headers;
- ETags and conditional `304 Not Modified` responses;
- `GET` and `HEAD` semantics;
- explicit permanent redirects only for registered aliases.

The resolver does **not** generate ontology semantics. It serves representations that have already been produced and approved by the canonical project repository.

### Run locally

```sh
go run ./cmd/idresolver -listen 127.0.0.1:8080 -root . -registry resolver/registry.json
```

Examples:

```sh
curl -i -H 'Accept: text/html' http://127.0.0.1:8080/exergism
curl -i -H 'Accept: application/ld+json' http://127.0.0.1:8080/
curl -i -H 'Accept: text/turtle' http://127.0.0.1:8080/exergism
```

The last request intentionally returns `406` until an Exergism Turtle representation using the adopted namespace has actually been published.

## Minimal VPS deployment

A 512 MB Debian VPS is sufficient for this service. The intended deployment is:

```text
Internet -> TLS frontend -> idresolver (127.0.0.1:8080) -> versioned/static representations
```

[`deploy/Caddyfile`](deploy/Caddyfile) is deliberately tiny: Caddy terminates HTTPS and proxies bytes. All semantic routing and MIME negotiation belongs to `idresolver`.

A hardened [`systemd` unit](deploy/id-exergism.service) is included. No database, Docker runtime, Java service, or application framework is required.

## Reserved identifier surfaces

- Exergism vocabulary: `https://id.exergism.org/exergism#`
- Exergism ontology: `https://id.exergism.org/ontology/exergism`
- ECL vocabulary: `https://id.exergism.org/ecl#` (reserved only; not adopted by ECL unless ECL explicitly migrates)

See [`PERSISTENCE.md`](PERSISTENCE.md) before minting new identifiers.

## Current publication boundary

The Exergism namespace and ontology IRI are reserved here, but the current registry publishes only human-readable HTML for those routes. RDF serializations MUST NOT be registered until the canonical Exergism project adopts the new namespace in a versioned release.

The service root additionally exposes a JSON-LD description of the identifier service itself. This does not change or pre-empt the Exergism ontology migration.

## Authority boundary

This repository resolves identifiers. It does not become the canonical source of the philosophical corpus, ontology semantics, ECL legal artifacts, governance decisions, or future registry data merely because those resources use identifiers under this host.

Project-specific authority remains with the repository and release process that owns the relevant layer.
