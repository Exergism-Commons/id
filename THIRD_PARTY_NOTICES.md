# Third-Party and Redistribution Notices

Audit date: **4 September 2026**.

## Runtime binary

The `idresolver` Go code currently imports only the Go standard library plus this repository's own `internal/resolver` package. There are no external Go module requirements and no `go.sum` file. The compiled resolver therefore embeds Go runtime/standard-library code but no third-party Go module dependency.

### Go

- Build/runtime component: Go toolchain, runtime and standard library.
- Current stable release reviewed and verified in CI: **Go 1.27.1**.
- Deployment policy: the DigitalOcean bootstrap resolves the current stable release from `go.dev` at install time and verifies the official SHA-256 before installation.
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

The repository currently uses:

- `actions/checkout` **v7.0.1**;
- `actions/setup-go` **v7.0.0**.

They are build/CI tooling and are not shipped in the `idresolver` runtime distribution.

## Operating-system packages

The setup script installs packages from the selected Debian/Ubuntu repositories. Those packages are not copied into this repository and remain governed by their distribution/upstream licences. A redistributed VM image or appliance must separately account for the licences of the packages actually included in that image.

## Exergism Commons semantic representations

This resolver republishes approved HTML/RDF/JSON-LD representations whose semantic authority remains in the relevant Exergism Commons project repository. In particular, Funding representations originate from `Exergism-Commons/funding`.

At the time of this audit, the Funding repository does **not** contain an explicit root `LICENSE` file. That is not a third-party dependency issue, but it is a redistribution-rights gap for downstream parties: public availability on GitHub does not itself grant a general copyright redistribution licence. Before EC offers third parties a broadly redistributable bundle of Funding ontology/record content, the authoritative source repository should adopt an explicit content/software licensing policy and ensure contributor rights are covered.

## Front-end assets

The identifier site uses system font stacks and Exergism Commons branding/theme assets served from `www.exergism.org`. No third-party webfont, CDN JavaScript framework or packaged front-end library was identified in the reviewed identifier repository.

## Scope

This file records the redistribution surface identified in the repository and deployment path. It is not a substitute for qualified legal review, and it should be updated whenever a new dependency, vendored asset, container image, binary package or copied external dataset is added.
