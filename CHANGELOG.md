# Changelog

All notable changes to Moon Panel are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.3] - 2026-05-03

Hotfix release for two functional bugs surfaced testing v0.1.2 in
production, plus a long-standing placeholder filled in.

### Fixed

- Version badge no longer renders as `vundefined`. The frontend
  `getVersion()` helper in v0.1.2 typed the axios call as `<VersionInfo>`
  and read `.data` as if it were the inner payload — but the backend
  wraps every response in `{code, msg, data}`, and the http client (plain
  axios) doesn't unwrap. Result: `version.value.version` was reading the
  envelope's nonexistent `version` field, returning `undefined`. The
  LDFLAGS injection itself was working correctly the whole time. Fixed
  to follow the same `data.data!` pattern as `panel.ts`.
- Version-badge popover preview now skips heading / hr / code-fence /
  link-reference lines and takes the first genuine prose paragraph
  (joined up to 2 lines, truncated at 120 chars). v0.1.2's preview
  returned the heading line stripped of `##`, which read as a redundant
  date repeat (e.g. `[0.1.2] - 2026-05-02`).

### Added

- `GET /api/admin/stats` (auth-gated) returning `groups_count`,
  `cards_count`, `engines_count`, and the count of audit-log entries
  written in the last 7 days. Drives the admin Overview page —
  previously hardcoded to `0`.
- Admin Overview now displays the four real counters instead of the
  three zero-filled placeholders. The "本页未来会显示..." NAlert is
  removed; the page is now an actual overview.

## [0.1.2] - 2026-05-02

Adds an in-app version indicator so deployments can see at a glance whether
they're behind upstream, plus the build-time wiring (LDFLAGS) to embed real
version metadata into the released binary.

### Added

- Version badge in the bottom-left corner of the home page and admin
  layout. Click to open a popover showing the running version + build
  date + short commit, the most recent 3 GitHub releases (tag, date,
  one-line preview), and a "View all on GitHub" link.
- `GET /api/version` (public, no auth) returning the binary's
  `{ version, build_date, commit }`. Frontend reads this on every page
  load; backend value is set at `go build` time via `-ldflags -X` so it
  reflects the actual published image, not a hardcoded constant.
- Recent releases are pulled from the public
  `api.github.com/repos/.../releases` endpoint (no auth, 60 req/h is
  plenty for click-to-open) and cached in `localStorage` for 30 minutes
  per session. Network / 429 errors fall back to stale cache or hide the
  releases section gracefully — the current-version display always
  works.

### Changed

- `Dockerfile`: backend build stage now accepts `VERSION`, `BUILD_DATE`,
  `COMMIT` build args and feeds them into `-ldflags -X` overrides for
  `internal/api.{Version,BuildDate,Commit}`. Local `docker build` without
  the args keeps the dev defaults; release builds get real values.
- `.github/workflows/release.yml`: passes the tag-derived version,
  workflow start time, and full commit SHA into the build via
  `--build-arg`. No retag required to refresh metadata — the next tag
  push fills in fresh values.

## [0.1.1] - 2026-05-02

Hotfix release covering two functional bugs surfaced in v0.1.0 production
deployment, an entrypoint robustness pass, and a substantial deployment
documentation expansion.

### Fixed

- Icon autocomplete now correctly commits the selected option's value
  (`lucide:<name>` for Lucide icons, full CDN URL for dashboard-icons)
  instead of the bare name. NAutoComplete's default behavior writes the
  option's `label` into the v-model on selection — that's correct for
  picker UIs where label === value, but our options use display name as
  label and prefixed/qualified strings as value. The fix overrides the
  v-model in `nextTick()` after select so the prefix isn't lost; saves
  no longer trip the icon-format validation warning.
- Default builtin wallpapers now appear in the admin Site Settings
  picker on private-mode (`MOON_PUBLIC_MODE=false`) deployments. The
  initial `ui.ensureLoaded()` call hits a 401 before login (the panel
  endpoint requires auth in private mode) and silently bails, leaving
  `ui.builtins` as `[]`. App.vue now watches `auth.authenticated` and
  re-fetches the panel on the false→true transition — covers regular
  login, TOTP verification, and first-time admin init in a single watch.

### Changed

- `docker/entrypoint.sh`: chown the data directory by numeric
  `$PUID:$PGID` rather than via resolved user/group names, so a
  corrupted `/etc/passwd` or a silent `addgroup`/`adduser` failure
  doesn't leave files root-owned. Added an explicit startup log line
  (`[entrypoint] chown'd /data to ...`) and fail-fast on chown error
  (a clear FATAL message instead of crash-looping with an opaque
  permission-denied later in the stack).

### Added

- README: full Deployment / Updating / Common Issues sections in both
  English and Chinese. Walks through PUID/PGID determination per
  platform (Linux / Synology / Unraid / TrueNAS), step-by-step compose
  setup, expected log sequence, the `:0.1` minor-track upgrade pattern,
  and troubleshooting for the most common failure modes (permission
  denied on jwt.key, bind-mount path missing, restart loops).

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

[Unreleased]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.3...HEAD
[0.1.3]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/bevanho777-max/moon-panel/releases/tag/v0.1.0
