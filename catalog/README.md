# Namespace Registry and Term Catalog

`id.exergism.org` is the persistent identifier authority for Exergism Commons. It is not the semantic authority for every vocabulary it resolves.

This directory separates two machine-readable concerns from the HTTP route registry in `resolver/registry.json`:

- `namespaces.json` records namespace ownership, canonical ontology IRIs, migration state, imports and authoritative source locations.
- `terms.json` is a discoverability index. A catalog entry never creates a term and never overrides the ontology in the owning repository.

## Ownership rule

Before minting a new EC term, projects should first inspect the Namespace Registry and Term Catalog. Reuse an existing term when its documented semantics are sufficient. Otherwise mint the term in the narrowest authoritative domain namespace. Cross-project primitives belong in `commons#` only after explicit governance review; institutional governance belongs in `governance#`; domain semantics remain with their owning project.

## Lifecycle

Namespace states are deliberately explicit:

- `adopted`: the owning project has adopted the canonical namespace;
- `migrating`: the namespace is reserved and a coordinated migration is in progress;
- future states may include `deprecated` or `retired`, but identifiers are never reassigned.

Terms from a migrating namespace are not presented as canonical until the owning project has materialized and reviewed the migration. Historical released namespaces remain historical facts and are not rewritten.

## Resolver boundary

`resolver/registry.json` answers how a concrete HTTP resource is dereferenced. `catalog/namespaces.json` answers which project owns a semantic namespace. `catalog/terms.json` answers which canonical terms are discoverable. Keeping those concerns separate prevents the resolver from accidentally becoming the source of semantic truth.
