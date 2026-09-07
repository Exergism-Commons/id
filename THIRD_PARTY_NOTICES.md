# Third-Party and Redistribution Notices

Audit date: **8 September 2026**.

## Runtime binary

The `idresolver` Go code currently imports only the Go standard library plus this repository's own `internal/resolver` package. There are no external Go module requirements and no `go.sum` file. The compiled resolver therefore embeds Go runtime/standard-library code but no third-party Go module dependency.

### Go

- Build/runtime component: Go toolchain, runtime and standard library.
- Current stable release reviewed and verified in CI: **Go 1.27.1**.
- Licence: BSD-style Go licence.
- Required redistribution text: `LICENSES/Go.txt`.

The Go licence requires binary redistributions to reproduce its copyright notice, conditions and disclaimer in accompanying documentation/materials. The deployment script installs that notice alongside the resolver documentation.

## Reverse proxy / TLS frontend

### Caddy

- Current stable release reviewed: **Caddy 2.11.4**.
- Deployment policy: installed separately from Caddy's official stable Debian/Ubuntu package repository.
- Relationship to `idresolver`: separate process; not linked into or bundled inside the resolver binary.
- Licence: Apache License 2.0.
- Licence text retained at `LICENSES/Apache-2.0.txt` for redistribution/documentation convenience.

If a VM image, appliance or other distribution actually includes Caddy binaries, the distributor must comply with Apache-2.0 and preserve any applicable upstream notices. The bootstrap script itself merely installs Caddy from its upstream repository.

## CI-only software

The repository uses GitHub Actions by immutable commit SHA, including `actions/checkout`, `actions/setup-go` and `actions/setup-python`. They are build/CI tooling and are not shipped inside the `idresolver` runtime binary.

The semantic-catalog completeness audit additionally installs RDFLib from the exact upstream Git commit recorded in `requirements-semantic.txt`. RDFLib and its Python dependency closure are CI-only validation tooling: they are not vendored into this repository, embedded in the Go resolver, included in runtime release assets, or installed by the production bootstrap. Their role is deliberately independent of the text-based term-catalog generator so CI can detect catalog omissions caused by Turtle formatting changes.

## Operating-system packages

The setup script installs packages from the selected Debian/Ubuntu repositories. Those packages are not copied into this repository and remain governed by their distribution/upstream licences. A redistributed VM image or appliance must separately account for the licences of the packages actually included in that image.

## Exergism Commons semantic representations

`id.exergism.org` republishes reviewed semantic representations for dereferencing. **Publication by the resolver is not a relicensing event and does not transfer semantic authority to this repository.** The owning repository and its adopted governance remain authoritative.

### Commons and Governance

The Commons and Governance Turtle representations, versioned ontology snapshots and downstream-authority profile are copied byte-for-byte from `Exergism-Commons/governance`, currently pinned to source commit `517c0df4c905c79ec2b2824231b40ccaa6a1b510`. CI independently fetches that commit and verifies the source/publication byte relationship.

At this audit date, `Exergism-Commons/governance` does **not** contain an explicit root `LICENSE` granting general downstream redistribution rights. Accordingly, this resolver's publication of EC-owned copies must not be interpreted as a general licence for third parties to redistribute, modify or relicense those semantic artifacts. An explicit source-repository licensing/adoption decision is required before describing those artifacts as broadly redistributable.

### Funding

Funding representations originate from `Exergism-Commons/funding`. At this audit date, that repository likewise does **not** contain an explicit root `LICENSE` granting general downstream redistribution rights. Public availability on GitHub and dereferencing through `id.exergism.org` do not themselves create such a grant.

Until the relevant owning repositories adopt an explicit content/software licensing policy and contributor-rights basis, `id` should limit itself to EC-controlled publication/dereferencing and should preserve exact source provenance. Third-party redistribution bundles must not claim permissions that the owner repositories have not granted.

## Front-end assets

The identifier site uses system font stacks and Exergism Commons branding/theme assets served from `www.exergism.org`. No third-party webfont, CDN JavaScript framework or packaged front-end library was identified in the reviewed identifier repository.

## Scope

This file records the redistribution surface identified in the repository and deployment path. It is not a substitute for qualified legal review, and it should be updated whenever a new dependency, vendored asset, container image, binary package, copied semantic artifact or external dataset is added.
