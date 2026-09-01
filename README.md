# id.exergism.org

Persistent identifier infrastructure for **Exergism Commons**.

Public identifier authority: `https://id.exergism.org/`

This repository currently provides the static GitHub Pages resolver/documentation layer for the identifier host. Public identifiers are deliberately independent of this repository name and of GitHub as a hosting provider.

## Reserved identifier surfaces

- Exergism vocabulary: `https://id.exergism.org/exergism#`
- Exergism ontology: `https://id.exergism.org/ontology/exergism`
- ECL vocabulary: `https://id.exergism.org/ecl#` (reserved only; not adopted by ECL unless ECL explicitly migrates)

See [`PERSISTENCE.md`](PERSISTENCE.md) before minting new identifiers.

## Authority boundary

This repository resolves identifiers. It does not become the canonical source of the philosophical corpus, ontology semantics, ECL legal artifacts, governance decisions, or future registry data merely because those resources use identifiers under this host.

Project-specific authority remains with the repository and release process that owns the relevant layer.
