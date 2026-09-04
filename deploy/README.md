# Deployment

## DigitalOcean Droplet bootstrap

For a fresh **Ubuntu 26.04 LTS** or **Debian 13** Droplet, run:

```sh
curl -fsSL https://raw.githubusercontent.com/Exergism-Commons/id.exergism-commons.github.io/main/deploy/setup-digitalocean.sh | sudo bash
```

As reviewed on 4 September 2026, the current stable stack is Go **1.26.8**, Caddy **2.11.4**, `actions/checkout` **v7.0.1** and `actions/setup-go` **v7.0.0**. The bootstrap does not pin Go or Caddy to those point releases: it deliberately resolves the latest stable Go release from `go.dev` and installs Caddy from the official stable package channel so a fresh production deployment receives the then-current stable release.

The bootstrap script:

- updates installed operating-system packages from the selected stable/LTS distribution repositories;
- resolves and installs the current stable Go toolchain, requiring Go 1.26 or newer;
- verifies the official SHA-256 of the downloaded Go archive before installation;
- installs the latest Caddy stable package from its official repository;
- creates the restricted `idexergism` service account;
- clones/updates this repository under `/srv/id.exergism.org`;
- runs `go test ./...` and `go vet ./...` before building;
- installs the resolver at `/usr/local/bin/idresolver`;
- installs the EULA, redistribution notices and third-party licence texts under `/usr/local/share/doc/idresolver`;
- installs and starts the hardened `id-exergism` systemd service;
- exposes only Caddy on ports 80/443 and keeps the resolver on `127.0.0.1:8080`;
- enables UFW for OpenSSH, HTTP and HTTPS;
- runs local HTML and Turtle negotiation smoke tests.

The script deliberately does **not** modify DNS. Keep the existing GitHub Pages target until the resolver is healthy locally. Then point the `id.exergism.org` A/AAAA records to the Droplet; Caddy will obtain the public TLS certificate once DNS resolves to the server.

Useful commands after setup:

```sh
systemctl status id-exergism
journalctl -u id-exergism -f
systemctl status caddy
go version
caddy version
```

Verify after DNS propagation:

```sh
curl -i -H 'Accept: text/html' https://id.exergism.org/funding
curl -i -H 'Accept: text/turtle' https://id.exergism.org/ontology/funding
```

## Redistribution

The runtime dependency surface is intentionally small. `idresolver` uses only the Go standard library plus repository-local code; Caddy runs as a separate upstream-installed process. See `../THIRD_PARTY_NOTICES.md`, `../EULA.md` and `../LICENSES/` before packaging or redistributing the resolver or a complete VM image.
