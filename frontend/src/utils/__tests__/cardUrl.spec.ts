import { describe, it, expect } from 'vitest'
import { effectiveURL, type NetworkState } from '../cardUrl'
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

const noOverrides: NetworkState = { global: 'auto', overrides: {} }

describe('effectiveURL', () => {
  it('auto mode + url_default=internal → picks internal', () => {
    const r = effectiveURL(makeCard({ url_default: 'internal' }), noOverrides)
    expect(r).toEqual({ url: 'http://192.168.1.1/', side: 'internal', fallback: false })
  })

  it('auto mode + url_default=external → picks external', () => {
    const r = effectiveURL(makeCard({ url_default: 'external' }), noOverrides)
    expect(r).toEqual({ url: 'https://example.com/', side: 'external', fallback: false })
  })

  it('global=internal forces internal even if card url_default=external', () => {
    const r = effectiveURL(
      makeCard({ url_default: 'external' }),
      { global: 'internal', overrides: {} },
    )
    expect(r?.side).toBe('internal')
    expect(r?.url).toBe('http://192.168.1.1/')
    expect(r?.fallback).toBe(false)
  })

  it('global=external forces external', () => {
    const r = effectiveURL(
      makeCard({ url_default: 'internal' }),
      { global: 'external', overrides: {} },
    )
    expect(r?.side).toBe('external')
    expect(r?.url).toBe('https://example.com/')
  })

  it('per-card override beats global', () => {
    const r = effectiveURL(
      makeCard({ id: 42, url_default: 'internal' }),
      { global: 'external', overrides: { 42: 'internal' } },
    )
    expect(r?.side).toBe('internal')
    expect(r?.url).toBe('http://192.168.1.1/')
  })

  it('per-card override beats card url_default in auto mode', () => {
    const r = effectiveURL(
      makeCard({ id: 7, url_default: 'internal' }),
      { global: 'auto', overrides: { 7: 'external' } },
    )
    expect(r?.side).toBe('external')
  })

  it('wanted=internal but url_internal empty → fall back to external, mark fallback', () => {
    const r = effectiveURL(
      makeCard({ url_internal: '', url_default: 'internal' }),
      noOverrides,
    )
    expect(r).toEqual({ url: 'https://example.com/', side: 'external', fallback: true })
  })

  it('wanted=external but url_external empty → fall back to internal', () => {
    const r = effectiveURL(
      makeCard({ url_external: '', url_default: 'external' }),
      noOverrides,
    )
    expect(r?.side).toBe('internal')
    expect(r?.fallback).toBe(true)
  })

  it('whitespace-only URL counts as empty (falls back)', () => {
    const r = effectiveURL(
      makeCard({ url_internal: '   ', url_default: 'internal' }),
      noOverrides,
    )
    expect(r?.fallback).toBe(true)
    expect(r?.side).toBe('external')
  })

  it('both URLs empty → returns null (UI should disable link)', () => {
    const r = effectiveURL(
      makeCard({ url_internal: '', url_external: '', url_default: 'internal' }),
      noOverrides,
    )
    expect(r).toBeNull()
  })

  it('both URLs whitespace → returns null', () => {
    const r = effectiveURL(
      makeCard({ url_internal: '  ', url_external: '\t', url_default: 'external' }),
      noOverrides,
    )
    expect(r).toBeNull()
  })

  it('override pointing at empty side falls back to non-empty side', () => {
    const r = effectiveURL(
      makeCard({ id: 9, url_internal: '', url_default: 'external' }),
      { global: 'auto', overrides: { 9: 'internal' } },
    )
    expect(r?.side).toBe('external')
    expect(r?.fallback).toBe(true)
  })
})
