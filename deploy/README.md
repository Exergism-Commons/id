# Deployment

## DigitalOcean Droplet bootstrap

For a fresh Debian or Ubuntu Droplet, run:

```sh
curl -fsSL https://raw.githubusercontent.com/Exergism-Commons/id.exergism-commons.github.io/main/deploy/setup-digitalocean.sh | sudo bash
```

The bootstrap script:

- resolves the current stable Go release directly from `https://go.dev/VERSION?m=text` and installs that exact stable toolchain, replacing an older preinstalled Go version if necessary;
- installs Caddy from its official package repository;
- creates the restricted `idexergism` service account;
- clones/updates this repository under `/srv/id.exergism.org`;
- runs `go test ./...` and `go vet ./...` before building;
- installs the resolver at `/usr/local/bin/idresolver`;
- installs and starts the hardened `id-exergism` systemd service;
- exposes only Caddy on ports 80/443 and keeps the resolver on `127.0.0.1:8080`;
- enables UFW for OpenSSH, HTTP and HTTPS;
- runs local HTML and Turtle negotiation smoke tests.

`go.mod` intentionally keeps the repository's minimum language/toolchain compatibility at Go 1.22. Production bootstrap and CI use the current stable Go release rather than pinning deployment to that minimum.

The script deliberately does **not** modify DNS. Keep the existing GitHub Pages target until the resolver is healthy locally. Then point the `id.exergism.org` A/AAAA records to the Droplet; Caddy will obtain the public TLS certificate once DNS resolves to the server.

Useful commands after setup:

```sh
systemctl status id-exergism
journalctl -u id-exergism -f
systemctl status caddy
go version
```

Verify after DNS propagation:

```sh
curl -i -H 'Accept: text/html' https://id.exergism.org/funding
curl -i -H 'Accept: text/turtle' https://id.exergism.org/ontology/funding
```
