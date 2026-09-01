# Exergism Commons Persistent Identifier Policy

Status: **bootstrap / proposed**

This document defines the persistence contract for identifiers under `https://id.exergism.org/`.

## 1. Purpose

`id.exergism.org` is an identifier surface, not a repository locator and not a general-purpose project website. Public identifiers minted here SHOULD remain valid independently of changes to GitHub repositories, hosting providers, deployment systems, documentation sites, or organizational web design.

## 2. Authority boundaries

Sharing the `id.exergism.org` host does not merge the authority of different Exergism Commons projects.

- `https://id.exergism.org/exergism#` is reserved for the Exergism vocabulary.
- `https://id.exergism.org/ecl#` is reserved for ECL and MUST NOT become operative merely because the reservation exists.
- Future registry identifiers MUST use separately documented paths and MUST NOT silently inherit philosophical, analytical, governance, or legal conclusions.

Each project adopts a namespace through its own versioned canonical process.

## 3. Stability

Once a public identifier is declared stable:

1. it MUST NOT be reassigned to a different semantic resource;
2. its meaning MUST NOT be silently replaced by an incompatible meaning;
3. a moved representation MUST be reached through a maintained redirect or resolver rule;
4. a deprecated term SHOULD continue to resolve and SHOULD expose its deprecation/supersession relationship;
5. provider or repository migrations MUST NOT require changing the identifier.

## 4. Versioned artifacts

Mutable 'current' resolution and immutable release identification are separate concerns.

For Exergism the intended pattern is:

- vocabulary namespace: `https://id.exergism.org/exergism#`
- current ontology identifier: `https://id.exergism.org/ontology/exergism`
- versioned ontology identifier: `https://id.exergism.org/ontology/exergism/{version}`

A versioned identifier MUST resolve to the semantic artifact for that version and MUST NOT later be repointed to different bytes while claiming to represent the same immutable release.

Versioned resolver routes SHOULD use long-lived immutable cache directives once their published bytes and checksums have been fixed. Non-versioned current routes MAY use shorter caching and may advance only through an explicit release/publishing change.

## 5. No retroactive namespace rewriting

Historical releases are not silently rewritten merely because Exergism Commons later adopts a better namespace. If a release was published with a different identifier scheme, migration is recorded in a subsequent release and compatibility mappings are published where appropriate.

## 6. Dereferencing and content negotiation

The service SHOULD provide a useful representation when an HTTP(S) identifier is dereferenced.

The native resolver in this repository performs server-side content negotiation from the HTTP `Accept` header. Negotiated resources MUST return `Vary: Accept`. A representation MUST advertise the media type of the bytes actually returned. Unsupported requested media types SHOULD return `406 Not Acceptable` rather than silently pretending that HTML is RDF.

Representations MAY include, as applicable:

- `text/html`
- `text/turtle`
- `application/rdf+xml`
- `application/ld+json`

A representation is not considered published merely because the resolver supports its MIME type. It becomes available only when the corresponding approved artifact is registered and deployed.

## 7. Hash vocabularies

Vocabulary terms MAY use hash IRIs, for example:

`https://id.exergism.org/exergism#Autonomy`

The fragment is client-side; the HTTP request is made for the exact namespace document path:

`https://id.exergism.org/exergism`

The production resolver SHOULD serve that exact path directly. A trailing-slash alias MAY redirect permanently to the canonical no-slash path, but infrastructure convenience MUST NOT redefine the persistent namespace.

## 8. Resolver registry

Resolver behavior SHOULD be declarative and reviewable. Registered routes define the canonical IRI, approved aliases, cache policy, representations, media types and artifact paths.

Changing a resolver rule MUST NOT create new project semantics by itself. In particular, registering an RDF serialization for a project requires that the canonical project has already adopted and published the relevant ontology identifiers.

## 9. Governance

Cross-project rules for control and stewardship of the identifier service belong to Exergism Commons organization governance. Project-specific ontology semantics remain controlled by the project that owns the ontology.

Changing this policy does not by itself modify an already released ontology, license, Schedule, patent instrument, or other immutable project artifact.
