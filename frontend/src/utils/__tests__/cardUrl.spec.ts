import { describe, it, expect } from 'vitest'
import { effectiveURL, type CardURLState } from '../cardUrl'
import type { Card } from '@/api/card'

function makeCard(over: Partial<Card> = {}): Card {
  return {
    id: 1,
    group_id: 1,
    title: 'Test',
    description: '',
    icon: '',
    icon_type: 'url',
    url_internal: 'http://192.168.1.1/',
    url_external: 'https://example.com/',
    url_default: 'internal',
    open_in_new_tab: true,
    sort: 10,
    created_at: '',
    updated_at: '',
    ...over,
  }
}

const auto: CardURLState = { effectiveMode: 'lan' }

describe('effectiveURL', () => {
  it("effectiveMode='lan' → primary internal", () => {
    const r = effectiveURL(makeCard(), auto)
    expect(r).toEqual({ url: 'http://192.168.1.1/', side: 'internal', fallback: false })
  })

  it("effectiveMode='internal' (forced) → primary internal, allows fallback", () => {
    const r = effectiveURL(
      makeCard({ url_internal: '' }),
      { effectiveMode: 'internal' },
    )
    expect(r).toEqual({ url: 'https://example.com/', side: 'external', fallback: true })
  })

  it("effectiveMode='external' (forced) → primary external, allows fallback to internal", () => {
    const r = effectiveURL(
      makeCard({ url_external: '' }),
      { effectiveMode: 'external' },
    )
    expect(r).toEqual({ url: 'http://192.168.1.1/', side: 'internal', fallback: true })
  })

  it("effectiveMode='wan' + url_external set → external, no fallback", () => {
    const r = effectiveURL(makeCard(), { effectiveMode: 'wan' })
    expect(r).toEqual({ url: 'https://example.com/', side: 'external', fallback: false })
  })

  it("★ effectiveMode='wan' + only url_internal set + url_default='external' → null (intentional, NO fallback)", () => {
    // v0.2.24 D3: explicit url_default='external' disables the D3 fallback,
    // restoring the v0.2.23 strict-WAN behavior tested here. With makeCard's
    // default url_default='internal', D3 would now fall back to internal.
    const r = effectiveURL(
      makeCard({ url_external: '', url_default: 'external' }),
      { effectiveMode: 'wan' },
    )
    expect(r).toBeNull()
  })

  it("★ effectiveMode='wan' + url_external whitespace + url_internal set + url_default='external' → null", () => {
    const r = effectiveURL(
      makeCard({ url_external: '   ', url_default: 'external' }),
      { effectiveMode: 'wan' },
    )
    expect(r).toBeNull()
  })

  it("per-card override='internal' beats effectiveMode='wan' (override forces internal even on WAN)", () => {
    const r = effectiveURL(
      makeCard(),
      { override: 'internal', effectiveMode: 'wan' },
    )
    expect(r?.side).toBe('internal')
    expect(r?.url).toBe('http://192.168.1.1/')
    expect(r?.fallback).toBe(false)
  })

  it("per-card override='external' beats effectiveMode='lan'", () => {
    const r = effectiveURL(
      makeCard(),
      { override: 'external', effectiveMode: 'lan' },
    )
    expect(r?.side).toBe('external')
    expect(r?.url).toBe('https://example.com/')
  })

  it("override='external' with empty url_external falls back to internal (override is NOT WAN-strict)", () => {
    const r = effectiveURL(
      makeCard({ url_external: '' }),
      { override: 'external', effectiveMode: 'wan' },
    )
    expect(r?.side).toBe('internal')
    expect(r?.fallback).toBe(true)
  })

  it('whitespace-only URL counts as empty (falls back in non-WAN mode)', () => {
    const r = effectiveURL(
      makeCard({ url_internal: '   ' }),
      { effectiveMode: 'lan' },
    )
    expect(r?.fallback).toBe(true)
    expect(r?.side).toBe('external')
  })

  it('both URLs empty → null (any mode)', () => {
    for (const m of ['lan', 'wan', 'internal', 'external'] as const) {
      const r = effectiveURL(
        makeCard({ url_internal: '', url_external: '' }),
        { effectiveMode: m },
      )
      expect(r).toBeNull()
    }
  })

  it('both URLs whitespace → null', () => {
    const r = effectiveURL(
      makeCard({ url_internal: '  ', url_external: '\t' }),
      { effectiveMode: 'lan' },
    )
    expect(r).toBeNull()
  })

  it("override='internal' pointing at empty side falls back to external", () => {
    const r = effectiveURL(
      makeCard({ url_internal: '' }),
      { override: 'internal', effectiveMode: 'lan' },
    )
    expect(r?.side).toBe('external')
    expect(r?.fallback).toBe(true)
  })

  it("effectiveMode='lan' falls back to external when url_internal empty", () => {
    const r = effectiveURL(
      makeCard({ url_internal: '' }),
      { effectiveMode: 'lan' },
    )
    expect(r).toEqual({ url: 'https://example.com/', side: 'external', fallback: true })
  })

  describe('D3 wan-strict with url_default fallback (v0.2.24)', () => {
    it("wan: url_external empty + url_default='internal' → fall back to internal", () => {
      const r = effectiveURL(
        makeCard({ url_external: '', url_default: 'internal' }),
        { effectiveMode: 'wan' },
      )
      expect(r).toEqual({ url: 'http://192.168.1.1/', side: 'internal', fallback: true })
    })

    it("wan: url_external empty + url_default='external' → null (no fallback)", () => {
      const r = effectiveURL(
        makeCard({ url_external: '', url_default: 'external' }),
        { effectiveMode: 'wan' },
      )
      expect(r).toBeNull()
    })

    it("wan: url_external empty + url_default='' (X2 no preference) → null", () => {
      // X2 (v0.2.24): backend writes url_default='' for new cards until the
      // user opts in. The Card type widened to include '' in Phase B.
      const r = effectiveURL(
        makeCard({ url_external: '', url_default: '' }),
        { effectiveMode: 'wan' },
      )
      expect(r).toBeNull()
    })

    it("wan: url_external set + url_default='internal' → external (D3 only on empty)", () => {
      const r = effectiveURL(
        makeCard({ url_default: 'internal' }),
        { effectiveMode: 'wan' },
      )
      expect(r).toEqual({ url: 'https://example.com/', side: 'external', fallback: false })
    })
  })
})
