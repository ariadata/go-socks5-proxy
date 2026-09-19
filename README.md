# Go SOCKS5 Proxy Server

A lightweight SOCKS5 proxy server in Go with optional authentication, internal-destination blocking, and multi-platform releases.

## Features

- SOCKS5 proxy (CONNECT) via [armon/go-socks5](https://github.com/armon/go-socks5)
- Flexible auth: users file, `--user` flags, or anonymous mode
- Blocks private/internal destinations by default (`--allow-internal` to override)
- Minimal Docker image (`scratch`) published to GHCR
- Multi-platform binaries: Linux (amd64/arm64/arm) and Windows (amd64/arm64)
- Automated GitHub Actions for Docker builds and tagged releases

## Installation

### Option 1: Download Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/ariadata/go-socks5-proxy/releases):

| Platform | Asset |
|----------|--------|
| Linux amd64 | `go-socks5-proxy-linux-amd64` |
| Linux arm64 | `go-socks5-proxy-linux-arm64` |
| Linux arm | `go-socks5-proxy-linux-arm` |
| Windows amd64 | `go-socks5-proxy-windows-amd64.exe` |
| Windows arm64 | `go-socks5-proxy-windows-arm64.exe` |

```bash
# Example: Linux amd64
wget https://github.com/ariadata/go-socks5-proxy/releases/latest/download/go-socks5-proxy-linux-amd64
chmod +x go-socks5-proxy-linux-amd64
./go-socks5-proxy-linux-amd64 --help
```

Releases are created by pushing a version tag (`v*`), e.g. `git tag v1.0.0 && git push origin v1.0.0`, or via **Actions → Release → Run workflow**.

### Option 2: Using Docker

```bash
docker pull ghcr.io/ariadata/go-socks5-proxy:latest

docker run -d -p 1080:1080 \
  -v "$(pwd)/users.conf:/users.conf:ro" \
  --name socks5-proxy \
  ghcr.io/ariadata/go-socks5-proxy:latest \
  /socks5-server --host 0.0.0.0 --port 1080 --users /users.conf
```

### Option 3: Build from Source

```bash
git clone https://github.com/ariadata/go-socks5-proxy.git
cd go-socks5-proxy
go build -o socks5-server .
```

## Usage

```bash
./socks5-server [OPTIONS]
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--host HOST` | Address to bind | `0.0.0.0` |
| `--port PORT` | Port to listen on | `1080` |
| `--users FILE` | Path to users config file | (none) |
| `--user USER` | `username:password` (repeatable) | (none) |
| `--allow-internal` | Allow proxying to private/internal IPs | `false` |
| `--version` | Print version | |
| `--help` | Print help | |

### Authentication

If no users are configured, the server runs **without** authentication. If at least one user is set (file and/or `--user`), authentication is required.

#### Anonymous
```bash
./socks5-server --port 1080
```

#### File-based
`users.conf`:
```plaintext
# username:password
# Lines starting with # are comments

user1:secure_password_123
user2:another_secure_password
```

```bash
./socks5-server --users users.conf --port 1080
```

#### Command-line users
```bash
./socks5-server --user "user1:pass1" --user "user2:pass2" --port 1080
```

#### Mixed (file + flags)
```bash
./socks5-server --users users.conf --user "extrauser:extrapass" --port 1080
```

### Internal destination blocking

By default (`--allow-internal=false`), CONNECT to these destinations is rejected (SOCKS rule failure):

- Loopback (`127.0.0.0/8`, `::1`)
- RFC1918 private (`10/8`, `172.16/12`, `192.168/16`)
- Link-local, unspecified (`0.0.0.0`, `::`)
- CGNAT `100.64.0.0/10`
- IPv6 ULA (`fc00::/7`)

Hostnames are resolved first; if they resolve to an internal IP, they are blocked too.

```bash
# Default: block internal
./socks5-server --users users.conf

# Allow internal/private destinations
./socks5-server --users users.conf --allow-internal
```

## Systemd Service

```bash
sudo cp socks5-server /usr/local/bin/
sudo chmod +x /usr/local/bin/socks5-server
sudo mkdir -p /etc/socks5
sudo cp users.conf /etc/socks5/
sudo chmod 600 /etc/socks5/users.conf

sudo tee /etc/systemd/system/socks5-server.service > /dev/null << 'EOF'
[Unit]
Description=SOCKS5 Proxy Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/socks5-server --users /etc/socks5/users.conf --host 0.0.0.0 --port 1080
Restart=always
RestartSec=5
User=nobody
Group=nogroup
WorkingDirectory=/etc/socks5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now socks5-server
sudo systemctl status socks5-server
```

## Docker Deployment

The image is based on `scratch` (static binary only at `/socks5-server`). There is no shell inside the container.

### Docker Run

```bash
# With authentication
docker run -d \
  --name socks5-proxy \
  -p 1080:1080 \
  -v "$(pwd)/users.conf:/users.conf:ro" \
  --restart unless-stopped \
  ghcr.io/ariadata/go-socks5-proxy:latest \
  /socks5-server --host 0.0.0.0 --port 1080 --users /users.conf

# Anonymous (no users file)
docker run -d \
  --name socks5-proxy \
  -p 1080:1080 \
  --restart unless-stopped \
  ghcr.io/ariadata/go-socks5-proxy:latest

# Allow internal destinations
docker run -d \
  --name socks5-proxy \
  -p 1080:1080 \
  -v "$(pwd)/users.conf:/users.conf:ro" \
  --restart unless-stopped \
  ghcr.io/ariadata/go-socks5-proxy:latest \
  /socks5-server --host 0.0.0.0 --port 1080 --users /users.conf --allow-internal
```

### Docker Compose

See the repo `docker-compose.yml`:

```yaml
services:
  socks5-proxy:
    image: ghcr.io/ariadata/go-socks5-proxy:latest
    container_name: go-socks5-proxy
    restart: unless-stopped
    ports:
      - "${DC_PROXY_PORT:-1080}:1080"
    command: ["/socks5-server", "--host", "0.0.0.0", "--port", "1080", "--users", "/users.conf"]
    volumes:
      - ./users.conf:/users.conf:ro
```

```bash
docker compose up -d
```

Optional: set host port with `DC_PROXY_PORT=1080`.

## Testing the Proxy

```bash
# With authentication
curl -x socks5://username:password@127.0.0.1:1080 https://httpbin.org/ip

# Anonymous
curl -x socks5://127.0.0.1:1080 https://httpbin.org/ip

# Resolve hostname through the proxy
curl -x socks5h://username:password@127.0.0.1:1080 https://httpbin.org/ip
```

Browser: SOCKS5 → `127.0.0.1:1080` with credentials if configured.

## Development & Building

### GitHub Actions

1. **`build.yml`** — on push to `main`, builds and pushes `ghcr.io/ariadata/go-socks5-proxy` (`latest` + `main-<sha>`)
2. **`release-multiplatform.yml`** — on tag `v*` (or manual dispatch), builds Linux/Windows binaries and publishes a GitHub Release

### Manual build

```bash
go build -o socks5-server .

GOOS=linux   GOARCH=amd64 go build -o go-socks5-proxy-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o go-socks5-proxy-windows-amd64.exe .

go test ./...
```

## Security Considerations

1. Use strong passwords; keep `users.conf` mode `600`
2. Prefer binding behind a firewall or reverse path control
3. Leave `--allow-internal` off unless you intentionally need LAN/localhost via the proxy
4. Rotate credentials regularly and watch logs for abuse

## Troubleshooting

**Connection refused** — confirm the process listens on the expected port (`ss -tlnp | grep 1080` or `systemctl status socks5-server`).

**Authentication failed** — check `username:password` format (no extra spaces) and file mount path (`/users.conf` in Docker).

**Internal host blocked** — expected by default; pass `--allow-internal` if you need it.

**Docker logs** (no shell in the image):
```bash
docker logs -f socks5-proxy
```

**Systemd logs:**
```bash
sudo journalctl -u socks5-server -f
```

## License

MIT.

## Support

- Help: `./socks5-server --help`
- Issues: [GitHub Issues](https://github.com/ariadata/go-socks5-proxy/issues)

## Acknowledgments

Built with [go-socks5](https://github.com/armon/go-socks5).
