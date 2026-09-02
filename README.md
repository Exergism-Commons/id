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

Project vocabularies:

- Exergism: `https://id.exergism.org/exergism#`
- ECL: `https://id.exergism.org/ecl#`
- Funding governance: `https://id.exergism.org/funding#`

Ontology IRIs:

- `https://id.exergism.org/ontology/exergism`
- `https://id.exergism.org/ontology/ecl`
- `https://id.exergism.org/ontology/funding`

Funding governance records:

- `https://id.exergism.org/funding/id/{stableId}`

A cross-project entity family is reserved for future specification:

- `https://id.exergism.org/entity/...`

See [`PERSISTENCE.md`](PERSISTENCE.md) before minting new identifiers.

## Migration boundary

`id.exergism.org` is now the intended persistent identifier authority for EC's semantic projects, but namespace adoption remains project-controlled:

- Exergism must migrate from `http://www.exergia.org/ns/` through a new canonical release.
- ECL must migrate from `urn:ecl:` through a new canonical release.
- Funding should adopt the `id.exergism.org` namespace before its machine-readable governance layer receives its first canonical release.

Historical versioned artifacts are not silently rewritten. Compatibility mappings are published where useful and semantically safe.

## Current publication boundary

The resolver publishes only representations that have been approved by the canonical project repository. Reserving a route or namespace here does not make its RDF semantics operative.

The service root additionally exposes a JSON-LD description of the identifier service itself.

## Authority boundary

This repository resolves identifiers. It does not become the canonical source of the philosophical corpus, ontology semantics, ECL legal artifacts, funding-governance decisions or future registry data merely because those resources use identifiers under this host.

Project-specific authority remains with the repository and release process that owns the relevant layer. Organization-level governance controls the persistence contract and stewardship of the identifier authority.
