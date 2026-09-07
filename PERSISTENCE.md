# Exergism Commons Persistent Identifier Policy

Status: **bootstrap / proposed**

This document defines the persistence contract for identifiers under `https://id.exergism.org/`.

## 1. Purpose

`id.exergism.org` is an identifier surface, not a repository locator and not a general-purpose project website. Public identifiers minted here SHOULD remain valid independently of changes to GitHub repositories, hosting providers, deployment systems, documentation sites, or organizational web design.

## 2. Authority boundaries

Sharing the `id.exergism.org` host does not merge the authority of different Exergism Commons projects.

Reserved vocabulary surfaces include:

- `https://id.exergism.org/commons#` — minimal shared EC primitives, owned by the Governance repository.
- `https://id.exergism.org/governance#` — organization-level institutional governance, owned by the Governance repository.
- `https://id.exergism.org/exergism#` — Exergism vocabulary.
- `https://id.exergism.org/ecl#` — ECL vocabulary.
- `https://id.exergism.org/funding#` — funding-specific vocabulary.

Reserved ontology surfaces include:

- `https://id.exergism.org/ontology/commons`
- `https://id.exergism.org/ontology/governance`
- `https://id.exergism.org/ontology/exergism`
- `https://id.exergism.org/ontology/ecl`
- `https://id.exergism.org/ontology/funding`

Governance also publishes versioned downstream authority profiles such as:

- `https://id.exergism.org/governance/profile/{governance-version}`

Funding canonical records use:

- `https://id.exergism.org/funding/id/{stableId}`

A namespace reservation does not make RDF semantics or institutional authority operative. Each owning project adopts semantics through its own reviewed, versioned canonical process. Resolver registration follows owner adoption; the resolver MUST NOT author, rewrite or infer ontology semantics merely to make publication convenient.

## 3. Shared entity identifiers

Real-world entities referenced by multiple EC projects SHOULD converge on organization-level identifiers rather than receiving incompatible project-local identities.

The reserved family for that work is:

`https://id.exergism.org/entity/...`

Exact schemas MUST be separately specified before stable entity identifiers are minted. Sharing an entity identifier does not merge the authority of ECL, Exergism, Funding or Governance over assertions about that entity.

## 4. Stability and historical enforcement

Once a public identifier is published:

1. it MUST NOT disappear merely because a repository is reorganized;
2. it MUST NOT be reassigned to a different semantic resource;
3. its canonical IRI MUST NOT change;
4. previously published aliases and representation locations MUST remain resolvable or be preserved by an explicit compatible migration;
5. a deprecated term SHOULD continue to resolve and SHOULD expose its deprecation/supersession relationship;
6. provider or repository migrations MUST NOT require changing the identifier.

The repository CI enforces these guarantees against Git history. For every trusted base→HEAD transition it rejects disappearance or reassignment of previously published resolver routes, aliases and representation paths. Routes already published with an `immutable` cache contract additionally freeze both their registered metadata and representation bytes.

This historical check is part of the persistence trust boundary; a test that merely compares current bytes with a mutable expected hash is insufficient.

## 5. Current ontology IRIs and OWL version IRIs

Mutable “current” ontology discovery and immutable ontology-version identity are separate concerns.

For an adopted ontology series:

- the current ontology IRI is `https://id.exergism.org/ontology/{name}`;
- each ontology version MUST declare an `owl:versionIRI`;
- that version IRI MUST be dereferenceable;
- the bytes served at a published version IRI MUST never later change.

Current adopted examples are:

- Commons ontology: `https://id.exergism.org/ontology/commons`
- Commons PRE2 version: `https://id.exergism.org/ontology/commons/0.1-PRE2`
- Governance ontology: `https://id.exergism.org/ontology/governance`
- Governance PRE2 version: `https://id.exergism.org/ontology/governance/0.1-PRE2`

The Governance PRE2 artifact imports the Commons PRE2 `owl:versionIRI`, so its published OWL import closure is reproducible without the resolver modifying owner-authored bytes.

Intended equivalent patterns for migrating domains are:

- Exergism: `https://id.exergism.org/ontology/exergism/{version}`
- ECL: `https://id.exergism.org/ontology/ecl/{version}`
- Funding: `https://id.exergism.org/ontology/funding/{version}`

A Git commit SHA may be recorded as publication provenance, but a repository commit path is not a substitute for the ontology's owner-authored `owl:versionIRI`.

## 6. Publication provenance

For copied semantic artifacts, `id` MUST be able to prove which authoritative bytes it publishes.

At minimum a publication manifest records:

- owner repository;
- authoritative source commit;
- source path;
- source Git blob SHA;
- current and/or versioned publication path;
- published Git blob SHA;
- declared transformation, which MUST be `none` for owner-authored ontology/version artifacts unless the owner has explicitly adopted a separate derived-artifact contract.

CI fetches the authoritative owner commit and verifies source blob == manifest == published bytes. `id` MUST NOT repair, rewrite or closure-freeze another repository's ontology locally; such semantic changes belong in the owning repository first.

## 7. No retroactive namespace rewriting

Historical releases are not silently rewritten merely because Exergism Commons later adopts a better namespace. If a release was published with a different identifier scheme, migration is recorded in a subsequent release and compatibility mappings are published where appropriate.

In particular:

- historical Exergism artifacts using `http://www.exergia.org/ns/` are not rewritten in place;
- historical ECL artifacts using `urn:ecl:` are not rewritten in place;
- an already published Funding JSON-LD record must not acquire new RDF meaning merely because a mutable external context changes; historical records require an immutable/versioned context or an otherwise content-bound expansion contract.

## 8. Dereferencing and content negotiation

The service SHOULD provide a useful representation when an HTTP(S) identifier is dereferenced.

The native resolver performs server-side content negotiation from `Accept`. Negotiated resources MUST return `Vary: Accept` when multiple representations exist. A representation MUST advertise the media type of the bytes actually returned. Unsupported requested media types SHOULD return `406 Not Acceptable` rather than silently returning HTML as RDF.

Representations MAY include:

- `text/html`
- `text/turtle`
- `application/rdf+xml`
- `application/ld+json`
- `application/json`

For an adopted hash vocabulary such as `https://id.exergism.org/commons#Person`, the fragment is client-side and the HTTP request resolves the namespace document (`/commons`). Adopted namespace documents MUST provide both a human-readable representation and at least one RDF representation.

## 9. Namespace and term catalogs

`catalog/namespaces.json` records ownership, canonical namespace and ontology IRIs, lifecycle state, imports, source provenance and version information. `catalog/terms.json` is a discoverability index and does not define semantics.

For an adopted namespace, the catalog contract requires:

- a resolvable HTML + RDF namespace document;
- a resolvable current ontology IRI;
- a resolvable immutable `owl:versionIRI`;
- an owner repository/source path/source commit;
- indexed terms with owner/type/version/dependency metadata.

Catalog endpoints are themselves published through the resolver. Catalog data MUST NOT be used to mint semantic definitions that do not exist in the authoritative owner ontology.

## 10. Resolver registry

Resolver behavior is declarative and reviewable in `resolver/registry.json`. Registered routes define canonical IRI, aliases, cache policy, representations, media types and artifact paths.

The loader MUST reject collisions among route paths, aliases and public representation paths in either insertion order. A previously published canonical route cannot be silently shadowed by a redirect alias.

Registering a representation does not create semantic authority. In particular, registering RDF for a project requires an approved owner artifact and provenance binding.

## 11. Governance and deployment boundary

Cross-project rules for control and stewardship of the identifier service belong to Exergism Commons organization governance. Project-specific ontology semantics remain controlled by the project that owns the ontology.

A merge into the `id` repository does not by itself prove that production is serving the new resolver state. Any cross-repository adoption that depends on new identifier routes MUST require a post-deployment smoke test of the actual HTTPS endpoints, media negotiation and immutable representation hashes before the downstream dependency is considered available.

Changing this policy does not by itself modify an already released ontology, license, Schedule, patent instrument, Funding decision or other immutable project artifact.
