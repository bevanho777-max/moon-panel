import type { Card } from '@/api/card'
import type { NetworkMode, NetworkOverride } from '@/stores/network'

export interface NetworkState {
  global: NetworkMode
  overrides: Record<number, NetworkOverride>
}

export interface EffectiveURL {
  url: string
  side: 'internal' | 'external'
  /** True if we wanted side X but it was empty so we fell back to the other. */
  fallback: boolean
}

/**
 * Decide which URL a card should link to given the current network state.
 *
 * Resolution order:
 *   1. Per-card override (network.overrides[card.id]) wins over everything.
 *   2. Global mode: 'internal' / 'external' force that side; 'auto' uses
 *      the card's own url_default.
 *   3. If the chosen side is empty, fall back to the other side and mark
 *      `fallback: true` so the UI can show a hint.
 *   4. If both sides are empty, return null — the UI should disable the link.
 */
export function effectiveURL(card: Card, state: NetworkState): EffectiveURL | null {
  const wanted: 'internal' | 'external' =
    state.overrides[card.id] ??
    (state.global === 'auto' ? card.url_default : state.global)

  const primary = wanted === 'internal' ? card.url_internal : card.url_external
  const secondary = wanted === 'internal' ? card.url_external : card.url_internal

  if (primary && primary.trim() !== '') {
    return { url: primary, side: wanted, fallback: false }
  }
  if (secondary && secondary.trim() !== '') {
    const otherSide: 'internal' | 'external' = wanted === 'internal' ? 'external' : 'internal'
    return { url: secondary, side: otherSide, fallback: true }
  }
  return null
}
