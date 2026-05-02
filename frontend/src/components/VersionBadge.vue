<script setup lang="ts">
// Bottom-left version badge with click-to-open popover (v0.1.2).
//
// Mounted once on Home and once on admin/Layout. Login.vue intentionally
// skips it — pre-auth UI is minimal by design. The popover loads recent
// releases from the public GitHub API (cached 30 min) so users can spot
// when their deployment falls behind upstream without leaving the panel.

import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  getRecentReleases,
  getVersion,
  type GitHubRelease,
  type VersionInfo,
} from '@/api/version'

defineProps<{ hidden?: boolean }>()

const version = ref<VersionInfo | null>(null)
const releases = ref<GitHubRelease[]>([])
const open = ref(false)
const releasesLoading = ref(false)
const releasesError = ref(false)

const triggerRef = ref<HTMLElement | null>(null)
const popoverRef = ref<HTMLElement | null>(null)

const displayVersion = computed(() => {
  if (!version.value) return '...'
  return 'v' + version.value.version
})

const buildDateDisplay = computed(() => {
  if (!version.value) return null
  if (version.value.build_date === 'unknown') return null
  const d = new Date(version.value.build_date)
  if (Number.isNaN(d.getTime())) return version.value.build_date
  return d.toISOString().slice(0, 10)
})

async function loadVersion() {
  try {
    version.value = await getVersion()
  } catch {
    // /api/version 404 (older binary) → show dev fallback. Keeps the badge
    // visible so the UI never has a "phantom missing element" gap.
    version.value = { version: 'dev', build_date: 'unknown', commit: 'unknown' }
  }
}

async function ensureReleases() {
  if (releases.value.length > 0 || releasesLoading.value) return
  releasesLoading.value = true
  releasesError.value = false
  try {
    releases.value = await getRecentReleases(3)
  } catch {
    releasesError.value = true
  } finally {
    releasesLoading.value = false
  }
}

function toggle() {
  open.value = !open.value
  if (open.value) ensureReleases()
}

function close() {
  open.value = false
}

function onClickOutside(e: MouseEvent) {
  if (!open.value) return
  const target = e.target as Node
  if (popoverRef.value?.contains(target)) return
  if (triggerRef.value?.contains(target)) return
  close()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && open.value) close()
}

function fmtDate(s: string): string {
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  return d.toISOString().slice(0, 10)
}

// Pull a one-line preview from a release body. The body is markdown — we
// don't ship a markdown renderer (out of scope for v0.1.2), so we grab the
// first non-empty, non-decorative line and strip leading "## " / list dash.
function previewBody(body: string): string {
  if (!body) return ''
  const lines = body.split('\n')
  for (const raw of lines) {
    const trimmed = raw.trim()
    if (!trimmed) continue
    if (trimmed === '---') continue
    if (trimmed.startsWith('```')) continue
    return trimmed.replace(/^#+\s*/, '').replace(/^[-*]\s+/, '')
  }
  return ''
}

onMounted(() => {
  loadVersion()
  document.addEventListener('mousedown', onClickOutside)
  document.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onClickOutside)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div v-if="!hidden" class="vb-wrap">
    <button
      ref="triggerRef"
      type="button"
      class="vb-trigger"
      :class="{ 'vb-trigger--open': open }"
      :title="'Click for release notes'"
      @click.stop="toggle"
    >
      {{ displayVersion }}
    </button>
    <transition name="vb-fade">
      <div v-if="open" ref="popoverRef" class="vb-popover" role="dialog">
        <div class="vb-current">
          <div class="vb-current__row">
            <span class="vb-current__label">Current</span>
            <span class="vb-current__version">{{ displayVersion }}</span>
          </div>
          <div v-if="buildDateDisplay" class="vb-current__date">
            Built {{ buildDateDisplay }}
            <span v-if="version && version.commit !== 'unknown'" class="vb-current__commit">
              · {{ version.commit }}
            </span>
          </div>
        </div>

        <div class="vb-divider" />

        <div class="vb-section">
          <div class="vb-section__label">Recent releases</div>
          <div v-if="releasesLoading" class="vb-muted">Loading…</div>
          <div v-else-if="releasesError" class="vb-muted">
            Couldn't reach github.com.
          </div>
          <ul v-else-if="releases.length > 0" class="vb-release-list">
            <li v-for="r in releases" :key="r.tag_name" class="vb-release">
              <a
                :href="r.html_url"
                target="_blank"
                rel="noopener noreferrer"
                class="vb-release__link"
              >
                <span class="vb-release__tag">{{ r.tag_name }}</span>
                <span class="vb-release__date">{{ fmtDate(r.published_at) }}</span>
              </a>
              <div v-if="previewBody(r.body)" class="vb-release__preview">
                {{ previewBody(r.body) }}
              </div>
            </li>
          </ul>
          <div v-else class="vb-muted">No releases yet.</div>
        </div>

        <div class="vb-divider" />

        <a
          class="vb-link"
          href="https://github.com/bevanho777-max/moon-panel/releases"
          target="_blank"
          rel="noopener noreferrer"
        >
          View all on GitHub →
        </a>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.vb-wrap {
  position: fixed;
  left: 16px;
  bottom: 16px;
  z-index: 100;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto,
    'PingFang SC', 'Microsoft YaHei', sans-serif;
}

.vb-trigger {
  display: inline-block;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.02em;
  color: rgba(255, 255, 255, 0.4);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid transparent;
  border-radius: 6px;
  cursor: pointer;
  transition:
    color 150ms ease,
    background-color 150ms ease,
    border-color 150ms ease;
  user-select: none;
}
.vb-trigger:hover,
.vb-trigger--open {
  color: rgba(255, 255, 255, 0.78);
  background: rgba(255, 255, 255, 0.08);
  /* Brand blue rather than the dynamic theme primary — the latter isn't
     exposed as a CSS var (NaiveUI ConfigProvider injects via cssr), and
     plumbing one through is its own follow-up. Brand fallback matches
     CardItem hover glow for visual consistency. */
  border-color: rgba(135, 165, 240, 0.45);
}

.vb-popover {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 0;
  width: 320px;
  max-width: calc(100vw - 32px);
  max-height: 60vh;
  overflow-y: auto;
  padding: 12px 14px;
  background: rgba(20, 20, 24, 0.85);
  -webkit-backdrop-filter: blur(20px) saturate(160%);
  backdrop-filter: blur(20px) saturate(160%);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
  font-size: 12px;
  line-height: 1.5;
  color: rgba(255, 255, 255, 0.85);
}

.vb-current__row {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.vb-current__label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: rgba(255, 255, 255, 0.4);
}
.vb-current__version {
  font-weight: 600;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.95);
}
.vb-current__date {
  margin-top: 2px;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.55);
}
.vb-current__commit {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: rgba(255, 255, 255, 0.4);
}

.vb-divider {
  height: 1px;
  margin: 10px -4px;
  background: rgba(255, 255, 255, 0.08);
}

.vb-section__label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: rgba(255, 255, 255, 0.4);
  margin-bottom: 6px;
}

.vb-release-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.vb-release {
  padding: 6px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
  transition: background-color 150ms ease;
}
.vb-release:hover {
  background: rgba(255, 255, 255, 0.07);
}
.vb-release__link {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  color: inherit;
  text-decoration: none;
}
.vb-release__tag {
  font-weight: 600;
  color: rgba(135, 165, 240, 0.95);
}
.vb-release__date {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.45);
}
.vb-release__preview {
  margin-top: 2px;
  font-size: 11.5px;
  color: rgba(255, 255, 255, 0.7);
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
}

.vb-muted {
  font-size: 11.5px;
  color: rgba(255, 255, 255, 0.45);
}

.vb-link {
  display: inline-block;
  font-size: 11.5px;
  color: rgba(135, 165, 240, 0.95);
  text-decoration: none;
}
.vb-link:hover {
  color: rgba(165, 195, 255, 1);
  text-decoration: underline;
}

.vb-fade-enter-active,
.vb-fade-leave-active {
  transition:
    opacity 200ms ease,
    transform 200ms ease;
}
.vb-fade-enter-from,
.vb-fade-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

@media (max-width: 768px) {
  .vb-wrap {
    left: 12px;
    bottom: 12px;
  }
  .vb-trigger {
    padding: 3px 8px;
    font-size: 11px;
  }
  .vb-popover {
    width: 280px;
  }
}
</style>
