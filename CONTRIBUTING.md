# Contributing to Moon Panel

Thanks for considering a contribution! This doc covers the workflow,
conventions, and review checklist used to keep `main` shippable.

[English](CONTRIBUTING.md) · [中文版](CONTRIBUTING.zh-CN.md)

If you're not sure whether an idea fits — open an issue first and ask.
Moon Panel is intentionally a single-user, no-account, opinionated panel;
some popular feature requests (multi-user / SSO / email registration) are
out of scope **by design**. The Roadmap section of the README is the
authoritative list of what's planned.

## Development setup

The full local-dev workflow (Go + air + Vite hot reload, port conventions,
data-isolation tips, troubleshooting) lives in [docs/DEV.md](docs/DEV.md).
The TL;DR:

```bash
# Backend — terminal A (port 3001)
cd backend
go mod tidy
.\dev.ps1                  # Windows PowerShell launcher (uses air)
# or, generic:
MOON_PORT=3001 MOON_ADMIN_PASSWORD=devdev99 MOON_DATA_DIR=./data-dev air

# Frontend — terminal B (port 5173)
cd frontend
npm install
npm run dev                # proxies /api/* and /uploads/* to :3001
```

Open http://localhost:5173 and log in as `admin` with whatever password
you set in `MOON_ADMIN_PASSWORD`.

Required tools: **Go 1.23+**, **Node 20+**. Recommended: `air` for backend
hot-reload (`go install github.com/air-verse/air@latest`).

## Branching & PRs

- Fork the repo and create a topic branch on your fork. Names like
  `feat/widget-foo`, `fix/login-redirect`, `docs/contributing-tweak`
  scan well — anything descriptive works.
- Keep each branch focused on one logical change. Several small PRs
  review faster than one mega-PR.
- Push your branch, then open a PR against `main`. The PR template
  loads automatically — fill it out; the checklist there is the gating
  criteria for review.
- CI must be green before review (Frontend job: typecheck + vitest +
  Playwright e2e; Backend job: vet + build).

## Commit message style — Conventional Commits

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short imperative description>
```

Allowed types:

| Type | Use for |
|---|---|
| `feat` | new user-visible feature |
| `fix` | bug fix |
| `docs` | docs-only change (README, CONTRIBUTING, comments) |
| `refactor` | code restructuring without behavior change |
| `test` | adding or updating tests |
| `chore` | tooling, deps, dotfile changes |
| `ci` | CI/CD config changes |
| `perf` | performance improvement |

Scope is optional but encouraged for larger code areas:
`feat(cards): bulk delete`, `fix(auth): handle empty cookie`.

Use a single `git commit -m "..."` for the short title. Add a body with
a second `-m ""` when the **why** needs more than 70 chars. The first
line is what shows up in `git log --oneline`, so make it count.

Example:

```
fix(security): per-IP login lockout, not global

Previous global counter let a single attacker DoS every user out.
Counts failed attempts per source IP over a 15-min window with
exponential backoff after the third lock. Trusted CIDRs in the
allowlist bypass the counter but still appear in the audit log.

Tested manually with curl from two source IPs to confirm independent
counters. CI vet + build covers the wiring.
```

## PR review checklist

Before requesting review:

- [ ] CI is green (Frontend ✓ + Backend ✓).
- [ ] PR description explains **what changed** and **why** (not just what).
- [ ] If the change is visible in the UI, attach before/after screenshots.
  For mobile-affecting changes, include both desktop and mobile shots.
- [ ] Tests cover the new behavior where reasonable. This is a personal
  panel, not banking software — but auth / security / data-mutation
  paths warrant a test.
- [ ] No unrelated formatting churn. Keep the diff minimal.
- [ ] No secrets in commits (passwords, tokens, real IPs, API keys). If
  one slipped through, force-pushing a clean version is fine pre-merge,
  but flag it so reviewers know to invalidate the leaked value.

Reviewer expectations: first-pass review within 48 h. If a reviewer asks
for changes, push follow-up commits to the same branch — don't force-push
during review unless rebasing for merge.

## Things we won't accept

To save everyone time, here's what's off the table:

- **Multi-user accounts / RBAC** — Moon Panel is single-user by design.
- **Email-based registration / password reset / OAuth / SSO** — same reason.
- **Telemetry, analytics, "phone-home" features** — local-only is a feature,
  not a bug.
- **Large feature additions without prior discussion** — open an issue
  to socialize the idea first. Surprise PRs that change architecture
  may be closed without merge.
- **Vendoring dependencies** — use `go mod` / `package.json` like normal.

## Issue templates

When opening an issue, pick the closest template:

- **Bug report** — something doesn't work as documented. Include repro
  steps, expected vs actual, and your environment.
- **Feature request** — a new capability you'd find useful. Describe the
  use case and any alternatives you considered.

Both templates live under [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/).
The PR template at [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md)
auto-loads when you open a pull request.

## License

By submitting a contribution you agree your work will be released under
the [MIT License](LICENSE) that covers the project.
