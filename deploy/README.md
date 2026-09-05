# Deployment

## Build once, run small

Production Droplets do **not** compile the resolver. GitHub Actions builds the Linux runtime assets, records the source commit and publishes SHA-256 checksums. The Droplet only downloads and verifies the prebuilt binary, checks out the matching repository revision and installs the runtime services.

The release workflow publishes:

- `idresolver-linux-amd64`;
- `idresolver-linux-arm64`;
- `SHA256SUMS`;
- `SOURCE_COMMIT`;
- `BUILD_INFO`.

Pushes to `main` update the rolling prerelease tag `runtime-main`. Tags matching `v*` produce versioned release assets. For an immutable production deployment, set `RELEASE_TAG` to a version tag. `runtime-main` is convenient while bringing up or testing the service.

## DigitalOcean Droplet bootstrap

For a fresh supported Ubuntu or Debian Droplet, run:

```sh
curl -fsSL https://raw.githubusercontent.com/Exergism-Commons/id/main/deploy/setup-digitalocean.sh | sudo bash
```

The bootstrap script:

- updates installed operating-system packages;
- installs only runtime/administrative dependencies (`curl`, `git`, `ufw`, Caddy and related base packages);
- downloads the architecture-matched `idresolver` binary from the selected GitHub Release;
- verifies the binary against the release `SHA256SUMS`;
- reads `SOURCE_COMMIT` and checks out the repository tag referenced by `RELEASE_TAG`;
- refuses deployment if the checked-out commit does not match the binary's recorded source commit;
- creates the restricted `idexergism` service account;
- installs the resolver at `/usr/local/bin/idresolver`;
- records the deployed release tag, source commit and binary checksum under `/usr/local/share/doc/idresolver`;
- installs and starts the hardened `id-exergism` systemd service;
- exposes only Caddy on ports 80/443 and keeps the resolver on `127.0.0.1:8080`;
- enables UFW for OpenSSH, HTTP and HTTPS;
- runs local HTML and Turtle negotiation smoke tests.

No Go toolchain, local compilation or build-time swap is required on the Droplet. A 512 MB VPS is therefore sufficient for the intended runtime.

### Selecting a release

The default deployment channel is:

```sh
RELEASE_TAG=runtime-main
```

To deploy an immutable versioned release instead:

```sh
export RELEASE_TAG=v0.1.0
curl -fsSL https://raw.githubusercontent.com/Exergism-Commons/id/main/deploy/setup-digitalocean.sh | sudo -E bash
```

The script deliberately does **not** modify DNS. Keep the existing GitHub Pages target until the resolver is healthy locally. Then point the `id.exergism.org` A/AAAA records to the Droplet; Caddy will obtain the public TLS certificate once DNS resolves to the server.

Useful commands after setup:

```sh
systemctl status id-exergism
journalctl -u id-exergism -f
systemctl status caddy
cat /usr/local/share/doc/idresolver/release-tag.txt
cat /usr/local/share/doc/idresolver/source-revision.txt
cat /usr/local/share/doc/idresolver/binary-sha256.txt
```

Verify after DNS propagation:

```sh
curl -i -H 'Accept: text/html' https://id.exergism.org/funding
curl -i -H 'Accept: text/turtle' https://id.exergism.org/ontology/funding
```

## Redistribution

The runtime dependency surface is intentionally small. `idresolver` uses only the Go standard library plus repository-local code; Caddy runs as a separate upstream-installed process. See `../THIRD_PARTY_NOTICES.md`, `../EULA.md` and `../LICENSES/` before packaging or redistributing the resolver or a complete VM image.
