# Changelog

All notable changes to Moon Panel are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-05-02

Initial public release. Self-hosted dashboard / start page with single-password
admin, intranet ↔ extranet URL switching, and frosted-glass UI — built as a
lightweight Sun-Panel alternative for personal use on NAS / VPS.

Single binary, embedded frontend, pure-Go SQLite (no CGO). Runs on a Raspberry
Pi or Synology with the same image.

### Added

#### Core
- Cards organized in groups; per-card title, description, icon, internal URL,
  external URL, and tags
- One-click intranet ↔ extranet switch on the home page (NetworkSwitcher) — the
  same card resolves to your LAN IP at home and your public domain on mobile data
- Public mode: home page accessible without login (default); private mode hides
  it behind auth
- Search bar on home page filters across groups, cards, descriptions, and URLs
- Configurable search engines (Google / Bing / DuckDuckGo / 百度 seeded; admin
  can add / remove / reorder / pick default)

#### UI / Customization
- Naive UI dark theme with primary-color override (5 presets + HSL color picker)
- Wallpaper system: 3 builtin SVG gradients (night / aurora / graphite) embedded
  in the binary, plus user uploads (auto-compressed client-side to 1920×1080
  WebP). Per-wallpaper backdrop blur 0–20 px
- Acrylic frosted-glass surfaces (Win11 / macOS Big Sur style) on cards, modals,
  and login — gated behind `body.has-wallpaper` so default-dark theme is
  unchanged when no wallpaper is set
- Stateful 4-state input fields (idle → opened → editing → modified) across all
  admin editors — click-to-clear semantics with revert affordance
- Lucide icon library + dashboard-icons catalog with autocomplete picker;
  per-card icon supports URL / `upload:hash` / `lucide:name` / fetched-from-URL
- Drag-and-drop reordering for groups, cards, and search engines (vuedraggable)
- HomeHero: configurable city list (up to 5) showing local time + weather
  emoji; °C / °F toggle; Open-Meteo backend, no API key required
- Mobile-responsive layout with long-press to open card target picker

#### Auth & Security
- Single-password admin login (no email, no signup, no SSO — by design for
  personal-panel scope)
- Bcrypt password hashing with 8-character minimum enforced at every
  bootstrap / change path (defense-in-depth)
- TOTP 2FA enrollment with QR code + 8 single-use backup codes; separate TOTP
  rate-limit independent of password lockout
- IP-based login lockout (5 password attempts / 15 min → 30 min lock, 7 TOTP
  attempts / 10 min → 15 min lock); CIDR allowlist for trusted networks (home
  LAN, fixed office IP) bypasses lockout but still appears in audit log
- Audit log of admin mutations (login / logout / 2FA / password / cards /
  groups / settings / backups), with recursive secret redaction; 90-day
  retention with opportunistic cleanup
- Session invalidation floor (`auth.session_floor`) — stamping a global
  cutoff revokes all in-flight cookies without restarting the container
- "Remember me" sessions (7 d default / 30 d remembered) via httpOnly cookie
- SSRF defense for icon-fetch endpoint: block private IP ranges, schema
  whitelist, optional host allowlist
- ZIP path-traversal guard + 50 MiB size cap on backup-restore upload

#### Backup & Restore
- JSON export of all groups / cards / search engines / settings (excluding
  password hash, TOTP secret, session floor, audit log)
- ZIP export bundles `uploads/` (icons + wallpapers) alongside metadata.json
  for full-restore portability
- Restore replaces existing content atomically in one transaction; preserves
  user / 2FA state on the new instance; auto-fallback for orphaned wallpaper
  references when the backup target file is missing

#### Deployment
- Single static Go binary with frontend embedded via `go:embed`
- Pure-Go SQLite (`modernc.org/sqlite`) — no CGO, cross-compiles in seconds
- Multi-architecture Docker images: `linux/amd64` + `linux/arm64`
- LinuxServer.io-style PUID/PGID env vars for NAS deployments (Synology,
  Unraid, TrueNAS) — data files end up owned by the host user
- Configurable Alpine mirror (`ALPINE_MIRROR` build arg) for CN builders;
  defaults to official `dl-cdn.alpinelinux.org`
- One-volume design: everything under `/data` (SQLite db + uploads/ + jwt.key)

#### Developer Experience
- Local dev workflow with hot reload (Go [air](https://github.com/air-verse/air)
  + Vite HMR), no Docker rebuild loop required for daily iteration
- One-line PowerShell launchers (`backend/dev.ps1`, `frontend/dev.ps1`) for
  Windows developers; auto-detects `go` / `air` install location
- Comprehensive [DEV.md](docs/DEV.md) covering env setup, port conventions,
  data migration via the backup feature, PowerShell ↔ Bash syntax, and
  common pitfalls (shadow configs, NaiveUI cssr classes, etc.)
- Dev / prod data isolation: dev uses `./data-dev` and port 3001, leaves
  production `./data` and port 3000 untouched

[Unreleased]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/bevanho777-max/moon-panel/releases/tag/v0.1.0
