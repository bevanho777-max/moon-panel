#!/usr/bin/env node
// Generate icon catalog JSONs from upstream sources.
// Run: node scripts/build-icon-catalog.mjs
//
// Outputs (committed to repo):
//   src/data/dashboard-icons.json   — walkxcode/dashboard-icons PNG names
//   src/data/lucide-icons.json      — lucide-vue-next exported names (kebab-case)
//
// Usage cadence: re-run when upstream catalogs add/remove icons. Not run at
// build time — keeps `npm run build` offline-safe and fast.

import { writeFileSync, mkdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = dirname(fileURLToPath(import.meta.url))
const DATA_DIR = resolve(HERE, '..', 'src', 'data')
mkdirSync(DATA_DIR, { recursive: true })

async function fetchDashboardIcons() {
  // GitHub git/trees API: one call returns the full tree of the default branch.
  // Anonymous rate limit is 60/h, plenty for periodic refresh.
  // Falls back to gh main → master if needed.
  const tryBranches = ['main', 'master']
  let tree = null
  for (const branch of tryBranches) {
    const url = `https://api.github.com/repos/walkxcode/dashboard-icons/git/trees/${branch}?recursive=1`
    console.log(`Fetching ${url} ...`)
    const res = await fetch(url, {
      headers: { Accept: 'application/vnd.github+json', 'User-Agent': 'moon-panel-icon-catalog' },
    })
    if (res.ok) {
      tree = await res.json()
      break
    }
    console.log(`  branch ${branch}: ${res.status}, trying next`)
  }
  if (!tree) throw new Error('GitHub API failed for both main/master')
  if (tree.truncated) {
    console.warn('⚠ GitHub returned truncated tree — catalog may be incomplete')
  }
  const names = tree.tree
    .filter((e) => e.type === 'blob' && e.path.startsWith('png/') && e.path.endsWith('.png'))
    .map((e) => e.path.slice('png/'.length, -'.png'.length))
    .sort()
  console.log(`  → ${names.length} dashboard-icons names`)
  return names
}

async function fetchLucideNames() {
  // lucide-vue-next exports each icon under THREE aliases:
  //   Wrench       (canonical PascalCase)
  //   WrenchIcon   (alias with -Icon suffix)
  //   LucideWrench (alias with Lucide- prefix)
  // We only want one entry per icon — keep canonical (no prefix, no Icon suffix).
  const lucide = await import('lucide-vue-next')
  const pascalNames = Object.keys(lucide).filter(
    (k) =>
      /^[A-Z]/.test(k) &&
      !k.startsWith('Lucide') &&
      !k.endsWith('Icon') &&
      // Skip a few non-icon exports
      !['default', 'createLucideIcon', 'Icon'].includes(k),
  )
  // PascalCase → kebab-case: ArrowUp → arrow-up, ShieldCheck → shield-check
  const kebabNames = pascalNames
    .map((n) =>
      n
        .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
        .replace(/([A-Z]+)([A-Z][a-z])/g, '$1-$2')
        .toLowerCase(),
    )
    .sort()
  const unique = [...new Set(kebabNames)]
  console.log(`  → ${unique.length} lucide names`)
  return unique
}

async function main() {
  const [dashboard, lucide] = await Promise.all([
    fetchDashboardIcons(),
    fetchLucideNames(),
  ])

  const dashPath = resolve(DATA_DIR, 'dashboard-icons.json')
  const lucPath = resolve(DATA_DIR, 'lucide-icons.json')
  writeFileSync(dashPath, JSON.stringify(dashboard) + '\n', 'utf8')
  writeFileSync(lucPath, JSON.stringify(lucide) + '\n', 'utf8')

  const fmtBytes = (n) => `${(n / 1024).toFixed(1)} KB`
  console.log(`✓ ${dashPath}  (${fmtBytes(JSON.stringify(dashboard).length)})`)
  console.log(`✓ ${lucPath}   (${fmtBytes(JSON.stringify(lucide).length)})`)
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
