import { pinyin } from 'pinyin-pro'
import type { Card } from '@/api/card'
import type { Group } from '@/api/group'

/** Home's panel shape: a group with its cards inlined. */
export type SearchableGroup = Group & { cards?: Card[] }

// CJK unified ideographs + ext A + compatibility ideographs. Only text that
// trips this gets sent through pinyin conversion.
const CJK =/[一-鿿㐀-䶿豈-﫿]/

/**
 * Split a raw query into lowercase terms. Every term must match (AND), so
 * "jelly media" narrows rather than widens.
 */
export function tokenize(query: string): string[] {
  const trimmed = query.trim().toLowerCase()
  if (!trimmed) return []
  return trimmed.split(/\s+/).filter(Boolean)
}

/** AND semantics: every token must appear in the haystack. No tokens → match. */
export function matchesTokens(haystack: string, tokens: string[]): boolean {
  return tokens.every((t) => haystack.includes(t))
}

/**
 * Pinyin expansion for CJK text: full spelling plus initials, so 「影音」 is
 * reachable as both "yingyin" and "yy". Returns '' for non-CJK input — the
 * check is a cheap regex test, and skipping it keeps buildIndex off the
 * pinyin-pro dictionary for the (common) all-ASCII case.
 */
export function pinyinVariants(text: string): string {
  if (!text || !CJK.test(text)) return ''
  const full = pinyin(text, { toneType: 'none', type: 'array' }).join('')
  const initials = pinyin(text, { pattern: 'first', toneType: 'none', type: 'array' }).join('')
  return ` ${full} ${initials}`
}

/**
 * The searchable blob for one card: its own text fields plus the name of the
 * group it lives in. Folding the group name into every card's haystack is what
 * makes "group name matches → whole group passes" fall out for free.
 */
export function cardHaystack(card: Card, groupName = ''): string {
  const fields = [
    card.title,
    card.description,
    card.url_internal,
    card.url_external,
    groupName,
  ].filter(Boolean)
  const literal = fields.join(' ')
  const phonetic = fields.map(pinyinVariants).join('')
  return (literal + phonetic).toLowerCase()
}

export interface IndexedCard {
  card: Card
  haystack: string
}

export interface IndexedGroup {
  group: SearchableGroup
  cards: IndexedCard[]
}

export type SearchIndex = IndexedGroup[]

/**
 * Precompute every card's haystack once per panel load. Pinyin conversion is
 * the expensive part here, so it must not sit on the keystroke path —
 * filterIndex only does substring tests against what this produced.
 */
export function buildIndex(groups: SearchableGroup[]): SearchIndex {
  return groups.map((group) => ({
    group,
    cards: (group.cards ?? []).map((card) => ({
      card,
      haystack: cardHaystack(card, group.name),
    })),
  }))
}

/**
 * Filter a prebuilt index. Empty query returns the original groups untouched
 * (same object identity, so Vue's keyed transitions don't churn). Groups left
 * with no matching cards are dropped.
 */
export function filterIndex(index: SearchIndex, query: string): SearchableGroup[] {
  const tokens = tokenize(query)
  if (tokens.length === 0) return index.map((g) => g.group)

  const out: SearchableGroup[] = []
  for (const entry of index) {
    const cards = entry.cards.filter((c) => matchesTokens(c.haystack, tokens)).map((c) => c.card)
    if (cards.length > 0) out.push({ ...entry.group, cards })
  }
  return out
}

export interface HighlightSegment {
  text: string
  /** True when this run of text matched one of the query tokens. */
  mark: boolean
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * Split text into literal / matched runs for <mark> rendering. Returns plain
 * strings — callers render them as text nodes, never v-html, so a card titled
 * `<img onerror=...>` stays inert.
 *
 * Pinyin-only hits produce no segments (there is no literal substring to mark
 * in the Chinese source); the card still shows, just without highlighting.
 */
export function highlightSegments(text: string, tokens: string[]): HighlightSegment[] {
  if (!text) return []
  const valid = tokens.filter(Boolean)
  if (valid.length === 0) return [{ text, mark: false }]

  const re = new RegExp(valid.map(escapeRegExp).join('|'), 'gi')
  const segments: HighlightSegment[] = []
  let last = 0
  for (const m of text.matchAll(re)) {
    const start = m.index ?? 0
    if (start > last) segments.push({ text: text.slice(last, start), mark: false })
    segments.push({ text: m[0], mark: true })
    last = start + m[0].length
  }
  if (last < text.length) segments.push({ text: text.slice(last), mark: false })
  return segments
}
