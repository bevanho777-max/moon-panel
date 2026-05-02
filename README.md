# 🌙 Moon Panel

> Self-hosted personal dashboard, lightweight Sun-Panel alternative.

[English](README.md) · [中文版](README.zh-CN.md)

[![CI](https://github.com/bevanho777-max/moon-panel/actions/workflows/ci.yml/badge.svg)](https://github.com/bevanho777-max/moon-panel/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/badge/release-coming%20soon-lightgrey.svg)](https://github.com/bevanho777-max/moon-panel/releases)

Moon Panel is a self-hosted start page for your home lab and personal services.
Single static binary, embedded SPA frontend, pure-Go SQLite (no CGO). Runs on
a Raspberry Pi, a Synology NAS, or a VPS from the same multi-arch Docker image.

It's a single-user panel by design — one password, no email, no SSO, no
registration. The home page is public by default; only `/admin` is gated.
Each card carries both an intranet and extranet URL, so the same dashboard
works on your home network and over the internet without changing config.

## Features

### Core
- Cards organized in groups; per-card title, description, icon, internal
  URL, external URL, and tags
- One-click intranet ↔ extranet switch on the home page — the same card
  resolves to your LAN IP at home and your public domain on mobile data
- Public mode (default) keeps the home page accessible without login;
  private mode hides it behind auth
- Search bar on the home page filters across groups, cards, descriptions,
  and URLs
- Configurable search engines (Google / Bing / DuckDuckGo / 百度 seeded);
  add, remove, reorder, pick default
- Drag-and-drop reordering for groups, cards, and search engines
- HomeHero: configurable city list (up to 5) showing local time + weather
  emoji; Open-Meteo backend, no API key required

### UI & Customization
- Naive UI dark theme with primary-color override (5 presets + HSL picker)
- Wallpaper system: 3 builtin SVG gradients (night / aurora / graphite)
  embedded in the binary, plus user uploads (auto-compressed client-side
  to 1920×1080 WebP). Per-wallpaper backdrop blur 0–20 px
- Acrylic frosted-glass surfaces (Win11 / macOS Big Sur style) on cards,
  modals, and login — only active when a wallpaper is set, so the default
  dark theme stays clean
- Stateful 4-state input fields (idle → opened → editing → modified) across
  all admin editors, with click-to-clear and revert affordances
- Lucide icon library + dashboard-icons catalog with autocomplete picker;
  icons can be a URL, an upload, a Lucide name, or fetched from a URL
- Mobile-responsive layout with long-press to open card target picker

### Auth & Security
- Single-password admin login (no email, no signup, no SSO — by design)
- Bcrypt password hashing with an 8-character minimum enforced at every
  bootstrap and change path (defense-in-depth)
- TOTP 2FA enrollment with QR code + 8 single-use backup codes; separate
  TOTP rate-limit independent of password lockout
- IP-based login lockout (5 password attempts / 15 min → 30 min lock;
  7 TOTP attempts / 10 min → 15 min lock); CIDR allowlist for trusted
  networks bypasses lockout but still appears in the audit log
- Audit log of admin mutations (login / logout / 2FA / password / cards /
  groups / settings / backups), with recursive secret redaction; 90-day
  retention with opportunistic cleanup
- Session invalidation floor — stamping a global cutoff revokes all
  in-flight cookies without restarting the container
- "Remember me" sessions (7 d default / 30 d remembered) via httpOnly cookie
- SSRF defense for icon-fetch endpoint: private-IP block, schema whitelist,
  optional host allowlist
- ZIP path-traversal guard + 50 MiB size cap on backup-restore upload

### Backup & Restore
- JSON export of all groups / cards / search engines / settings (excluding
  password hash, TOTP secret, session floor, audit log)
- ZIP export bundles `uploads/` (icons + wallpapers) alongside metadata for
  full-restore portability
- Restore replaces existing content atomically in one transaction; preserves
  user / 2FA state on the new instance; auto-fallback for orphaned wallpaper
  references when the backup target file is missing

### Deployment
- Single static Go binary with frontend embedded via `go:embed`
- Pure-Go SQLite (`modernc.org/sqlite`) — no CGO, cross-compiles in seconds
- Multi-architecture Docker images: `linux/amd64` + `linux/arm64`
- LinuxServer.io-style `PUID` / `PGID` env vars for NAS deployments
  (Synology, Unraid, TrueNAS) — data files end up owned by the host user
- One-volume design: everything under `/data` (SQLite db + `uploads/` +
  `jwt.key`)

## Screenshots

| Home (Desktop) | Home (Mobile) |
|---|---|
| ![Home page on desktop, with wallpaper, hero, search bar, and card grid](docs/screenshots/home-desktop.png) | ![Home page on mobile, responsive layout](docs/screenshots/home-mobile.png) |

| Admin · Cards | Admin · Site Settings |
|---|---|
| ![Cards admin page with acrylic data table](docs/screenshots/admin-cards.png) | ![Site settings page with wallpaper, theme color, and blur controls](docs/screenshots/admin-site-settings.png) |

## Quick Start (Docker)

Until v0.1.0 ships pre-built images, build from source:

```bash
git clone https://github.com/bevanho777-max/moon-panel.git
cd moon-panel
cp docker-compose.yml.example docker-compose.yml
# IMPORTANT: edit MOON_ADMIN_PASSWORD inside docker-compose.yml first
PUID=$(id -u) PGID=$(id -g) docker compose up -d --build
```

Open `http://localhost:3000` — login as `admin` with the password you set.
Comment out or remove `MOON_ADMIN_PASSWORD` from compose after first login;
it's only consulted while the users table is empty.

> After v0.1.0 release this becomes 2 lines: `curl` the compose file +
> `docker compose up -d` against the published
> `ghcr.io/bevanho777-max/moon-panel` image. The release badge above will
> turn green when that ships.

### Synology / NAS notes

Set `PUID=1026 PGID=100` (default `users` group on DSM) so files in `./data`
end up owned by your DSM user instead of the container's anonymous UID.

## Deployment

The Quick Start above is the build-from-source shortcut. This section walks
through the published-image flow with NAS-specific PUID/PGID guidance and
the rest of the operational lifecycle.

### Prerequisites

- Docker 24+ with the Compose v2 plugin (`docker compose ...`)
- Linux, Synology DSM 7.2+, Unraid, or TrueNAS Scale
- ~50 MiB free disk for the image, 5 MiB+ for the data volume

### Step 1 — Determine PUID / PGID

Files written under your data directory will end up owned by this UID/GID.
Match it to the host user that should own `./data`:

| Platform | Find with | Typical |
|---|---|---|
| Generic Linux | `id $USER` | `1000:1000` |
| Synology DSM | SSH in, run `id $USER` | `1026:100` (`users` group) |
| Unraid | (well-known) | `99:100` (`nobody:users`) |
| TrueNAS Scale | dataset owner per pool config | varies |

### Step 2 — Prepare the data directory

```bash
mkdir -p /your/path/moon-panel/data
sudo chown -R PUID:PGID /your/path/moon-panel/data
```

The container's entrypoint runs `chown -R` on every start, so the host-side
`chown` here is optional in v0.1.1+. On hosts where the Docker daemon
creates bind-mount targets as `root` (most Linuxes do), pre-creating the
directory yourself avoids a brief moment of root-owned files.

### Step 3 — docker-compose.yml

Copy [docker-compose.yml.example](docker-compose.yml.example) and edit
`MOON_ADMIN_PASSWORD`. Pin the image to a specific tag for reproducibility:

```yaml
services:
  moon-panel:
    image: ghcr.io/bevanho777-max/moon-panel:0.1.1
    container_name: moon-panel
    restart: unless-stopped
    ports:
      - "3000:3000"
    environment:
      PUID: 1026
      PGID: 100
      MOON_ENV: production
      MOON_PUBLIC_MODE: "false"
      MOON_ADMIN_PASSWORD: <set on first start, then comment out>
    volumes:
      - ./data:/data
```

### Step 4 — Start

```bash
docker compose up -d
docker compose logs -f moon-panel
```

Expected log sequence on first start:

```
[entrypoint] PUID=1026 PGID=100 DATA_DIR=/data (user=moon group=users)
[entrypoint] chown'd /data to 1026:100; starting moon-panel
moon-panel listening on :3000 (env=production public_mode=false ...)
```

### Step 5 — Access

Open `http://<host-ip>:3000` in your browser. Log in as `admin` with the
password from `MOON_ADMIN_PASSWORD`. After first login, comment out
`MOON_ADMIN_PASSWORD` from the compose file — it's only consulted while
the users table is empty, so leaving it set is harmless but unnecessary.

## Updating

### Track patch releases (recommended)

Pin to the minor track in `docker-compose.yml`:

```yaml
image: ghcr.io/bevanho777-max/moon-panel:0.1
```

Then:

```bash
docker compose pull
docker compose up -d
```

`:0.1` always points to the latest `0.1.x` patch, so bug fixes and
security patches arrive automatically. Your data under `./data` survives
image swaps untouched.

### Manual / version-pinned update

If you'd rather pin to a specific tag and update on your own schedule:

1. Edit `docker-compose.yml`: change `image:` to the exact target version.
2. `docker compose pull`
3. `docker compose up -d`

### Rolling back

Pin the image to a previous tag (e.g., `:0.1.0`) in compose and
`docker compose up -d`. Data files are forward-compatible across patch
versions but **not** guaranteed across major-version downgrades.

## Common Issues

### `permission denied opening /data/jwt.key`

Cause: the host directory ownership doesn't match `PUID:PGID`, or the
volume is mounted read-only.

Fix:

```bash
sudo chown -R PUID:PGID /your/path/data
docker compose restart
```

v0.1.1+ entrypoint logs the chown step explicitly. Check
`docker compose logs moon-panel` for `[entrypoint] chown'd /data to ...`
— if you don't see that line, the entrypoint exited before reaching chown
(check earlier log lines for the cause).

### `Bind mount failed: source path does not exist`

Cause: docker compose tried to mount `./data` before the host directory
existed. Some Docker versions auto-create the path as root; others fail.

Fix:

```bash
mkdir -p ./data
docker compose up -d
```

### Container restarts repeatedly

Cause: the entrypoint or the server crashes shortly after start. Common
reasons in order of likelihood:

1. Permission denied opening `/data/jwt.key` (see above)
2. `MOON_ADMIN_PASSWORD` shorter than 8 characters — backend rejects the
   bootstrap and exits non-zero on first start
3. Port conflict (`bind: address already in use :3000`)

Diagnose:

```bash
docker compose logs moon-panel | tail -50
```

The lines between `[entrypoint] PUID=...` and the next visible failure
usually point at the cause. If the log is empty, the entrypoint died
before printing — most often a missing or unreadable volume.

## Configuration

All environment variables (set them in `docker-compose.yml`):

| Variable | Default | Description |
|---|---|---|
| `MOON_PORT` | `3000` | HTTP listen port |
| `MOON_DATA_DIR` | `/data` (in container) | SQLite db + `uploads/` + `jwt.key` |
| `MOON_PUBLIC_MODE` | `true` | `false` requires login for the home page too |
| `MOON_ADMIN_PASSWORD` | _(empty)_ | Bootstrap only — used while the users table is empty, ignored after first admin exists. ≥ 8 chars enforced |
| `MOON_TOKEN_TTL_DAYS` | `7` | Default session lifetime (no "remember me") |
| `MOON_TOKEN_REMEMBER_TTL_DAYS` | `30` | Session lifetime when "remember me" is checked at login |
| `MOON_COOKIE_SECURE` | `false` | Set `true` when serving over HTTPS |
| `MOON_CORS_ORIGINS` | _(empty)_ | Comma-separated origins; leave empty for same-origin (default Docker deploy) |
| `MOON_TRUSTED_PROXIES` | `127.0.0.1,172.16.0.0/12` | CIDRs whose `X-Forwarded-*` headers are trusted |
| `MOON_JWT_SECRET` | _(auto)_ | Override the auto-generated secret persisted in `data/jwt.key` |
| `PUID` / `PGID` | `1000` / `1000` | Host user mapping for `./data` ownership |

For SSRF tuning (icon fetch) and other knobs, see comments in
[docker-compose.yml.example](docker-compose.yml.example).

## Roadmap

**v0.1 (current)** — core panel, auth + 2FA, themes & wallpapers, JSON / ZIP
backup, multi-arch Docker.

**v0.2 (planned)**
- Service health checks (ping / HTTP) → green/red dot on cards
- Bookmark batch import (Chrome / Firefox HTML format)
- Full-text search across card titles / descriptions
- PWA with offline home page

**Not planned**
- Multi-user / RBAC — single-user is intentional, simplifies a huge surface area
- Email login / SSO — same reason

See [PROJECT.md](PROJECT.md) for the original Phase 0–5 design notes.

## AI Collaboration

Moon Panel was built with
[Claude Code](https://claude.com/claude-code) (Anthropic's agentic coding tool)
from initial scaffolding to v0.1 release. The [memory/](memory/) directory
contains lessons learned during development — useful for both future
contributors using Claude Code and as a real-world AI-collaboration case
study.

## Contributing

PRs welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for the commit convention,
PR checklist, and issue templates. Local development workflow (Go +
[air](https://github.com/air-verse/air) hot reload + Vite HMR) is documented
in [docs/DEV.md](docs/DEV.md).

## License

[MIT](LICENSE)
