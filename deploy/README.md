# Deployment

## DigitalOcean Droplet bootstrap

For a fresh supported Ubuntu/Debian Droplet, run:

```sh
curl -fsSL https://raw.githubusercontent.com/Exergism-Commons/id/main/deploy/setup-digitalocean.sh | sudo bash
```

The bootstrap installs the distribution Go package as a bootstrap toolchain and uses Go's automatic toolchain management (`GOTOOLCHAIN=auto`) to select the repository toolchain declared in `go.mod`. Caddy is installed from its official stable package channel.

The bootstrap script:

- updates installed operating-system packages;
- installs a bootstrap Go toolchain from APT and selects the repository Go toolchain automatically;
- detects hosts with less than 1 GiB RAM and no swap, then creates a persistent 1 GiB `/swapfile` for build headroom;
- runs Go package compilation serially on tiny hosts to avoid transient compiler OOM failures;
- installs the latest Caddy stable package from its official repository;
- creates the restricted `idexergism` service account;
- clones/updates this repository under `/srv/id.exergism.org`;
- runs `go test -p=1 ./...` and `go vet -p=1 ./...` before building;
- installs the resolver at `/usr/local/bin/idresolver`;
- installs the EULA, redistribution notices and third-party licence texts under `/usr/local/share/doc/idresolver`;
- installs and starts the hardened `id-exergism` systemd service;
- exposes only Caddy on ports 80/443 and keeps the resolver on `127.0.0.1:8080`;
- enables UFW for OpenSSH, HTTP and HTTPS;
- runs local HTML and Turtle negotiation smoke tests.

A 512 MiB Droplet is sufficient for the resolver runtime. Building Go from source on a host that small can briefly exceed physical RAM, which is why the bootstrap provisions swap and limits build parallelism when necessary.

The script deliberately does **not** modify DNS. Keep the existing GitHub Pages target until the resolver is healthy locally. Then point the `id.exergism.org` A/AAAA records to the Droplet; Caddy will obtain the public TLS certificate once DNS resolves to the server.

Useful commands after setup:

```sh
systemctl status id-exergism
journalctl -u id-exergism -f
systemctl status caddy
go version
caddy version
free -h
```

Verify after DNS propagation:

```sh
curl -i -H 'Accept: text/html' https://id.exergism.org/funding
curl -i -H 'Accept: text/turtle' https://id.exergism.org/ontology/funding
```

## Redistribution

The runtime dependency surface is intentionally small. `idresolver` uses only the Go standard library plus repository-local code; Caddy runs as a separate upstream-installed process. See `../THIRD_PARTY_NOTICES.md`, `../EULA.md` and `../LICENSES/` before packaging or redistributing the resolver or a complete VM image.
