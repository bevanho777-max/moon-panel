import type { Card } from '@/api/card'
import type { EffectiveMode, NetworkOverride } from '@/stores/network'

export interface CardURLState {
  /** Per-card override (overrides[card.id]). Beats effectiveMode entirely. */
  override?: NetworkOverride
  /** Resolved direction from the network store. Sources global+detected+session. */
  effectiveMode: EffectiveMode
}

export interface EffectiveURL {
  url: string
  side: 'internal' | 'external'
  /** True if we wanted side X but it was empty so we fell back to the other. */
  fallback: boolean
}

/**
 * Decide which URL a card should link to given the network state.
 *
 * Resolution order:
 *   1. Per-card override (state.override) wins over everything else.
 *   2. effectiveMode picks the side:
 *        'internal' / 'lan' → primary=internal, fallback=external
 *        'external'         → primary=external, fallback=internal
 *        'wan'              → primary=external, NO fallback (return null)
 *   3. For modes other than 'wan', if the chosen side is empty, fall back to
 *      the other side and mark `fallback: true`.
 *   4. If no URL resolvable, return null — UI disables the link.
 *
 * The asymmetric 'wan' behavior is intentional: when auto-detection says the
 * user is on an external network, falling back to an internal URL would
 * produce an unreachable address. Better to disable the card than to send
 * the user to a guaranteed failure.
 *
 * url_default is intentionally unused in v0.2.23 — direction now comes from
 * auto-detection (effectiveMode) instead of the per-card hint. The field
 * persists for backward compat / future use.
 */
export function effectiveURL(card: Card, state: CardURLState): EffectiveURL | null {
  if (state.override) {
    return resolveWithFallback(card, state.override)
  }
  switch (state.effectiveMode) {
    case 'internal':
    case 'lan':
      return resolveWithFallback(card, 'internal')
    case 'external':
      return resolveWithFallback(card, 'external')
    case 'wan':
      return resolveWANStrict(card)
  }
}

function resolveWithFallback(
  card: Card,
  wanted: 'internal' | 'external',
): EffectiveURL | null {
  const primary = wanted === 'internal' ? card.url_internal : card.url_external
  const secondary = wanted === 'internal' ? card.url_external : card.url_internal

  if (primary && primary.trim() !== '') {
    return { url: primary, side: wanted, fallback: false }
  }
  if (secondary && secondary.trim() !== '') {
    const otherSide: 'internal' | 'external' =
      wanted === 'internal' ? 'external' : 'internal'
    return { url: secondary, side: otherSide, fallback: true }
  }
  return null
}

function resolveWANStrict(card: Card): EffectiveURL | null {
  const url = card.url_external
  if (url && url.trim() !== '') {
    return { url, side: 'external', fallback: false }
  }
  return null
}
