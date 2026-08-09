import { describe, it, expect } from 'vitest'
import {
  buildIndex,
  cardHaystack,
  filterIndex,
  highlightSegments,
  matchesTokens,
  tokenize,
  type SearchableGroup,
} from '../cardSearch'
import type { Card } from '@/api/card'

function makeCard(over: Partial<Card> = {}): Card {
  return {
    id: 1,
    group_id: 1,
    title: 'Jellyfin',
    description: '',
    icon: '',
    icon_type: 'url',
    url_internal: '',
    url_external: '',
    url_default: 'internal',
    open_in_new_tab: true,
    sort: 10,
    created_at: '',
    updated_at: '',
    ...over,
  }
}

function makeGroup(name: string, cards: Card[], id = 1): SearchableGroup {
  return { id, name, icon: '', sort: 10, created_at: '', updated_at: '', cards }
}

describe('tokenize', () => {
  it('splits on whitespace and lowercases', () => {
    expect(tokenize('Jelly Media')).toEqual(['jelly', 'media'])
  })

  it('collapses repeated / mixed whitespace', () => {
    expect(tokenize('  jelly \t\n media  ')).toEqual(['jelly', 'media'])
  })

  it('returns [] for empty and whitespace-only input', () => {
    expect(tokenize('')).toEqual([])
    expect(tokenize('   ')).toEqual([])
  })
})

describe('matchesTokens (AND)', () => {
  it('matches when every token is present', () => {
    expect(matchesTokens('jellyfin media server', ['jelly', 'media'])).toBe(true)
  })

  it('rejects when one token is missing', () => {
    expect(matchesTokens('jellyfin media server', ['jelly', 'plex'])).toBe(false)
  })

  it('treats no tokens as a match', () => {
    expect(matchesTokens('anything', [])).toBe(true)
  })
})

describe('filterIndex field coverage', () => {
  const cards = [
    makeCard({ id: 1, title: 'Jellyfin', description: '家庭影音服务' }),
    makeCard({ id: 2, title: 'Plex', url_internal: 'http://192.168.1.50:32400' }),
    makeCard({ id: 3, title: 'Grafana', url_external: 'https://metrics.example.com' }),
  ]
  const index = buildIndex([makeGroup('Media', cards)])

  it('matches on description only', () => {
    const r = filterIndex(index, '服务')
    expect(r.flatMap((g) => g.cards ?? []).map((c) => c.id)).toEqual([1])
  })

  it('matches on url_internal only', () => {
    const r = filterIndex(index, '192.168.1.50')
    expect(r.flatMap((g) => g.cards ?? []).map((c) => c.id)).toEqual([2])
  })

  it('matches on url_external only', () => {
    const r = filterIndex(index, 'metrics.example')
    expect(r.flatMap((g) => g.cards ?? []).map((c) => c.id)).toEqual([3])
  })

  it('AND across fields: title + url', () => {
    expect(filterIndex(index, 'plex 32400').flatMap((g) => g.cards ?? []).map((c) => c.id)).toEqual([2])
    expect(filterIndex(index, 'plex 99999')).toEqual([])
  })
})

describe('filterIndex group semantics', () => {
  const media = makeGroup('Media', [makeCard({ id: 1, title: 'Jellyfin' }), makeCard({ id: 2, title: 'Plex' })], 1)
  const tools = makeGroup('Tools', [makeCard({ id: 3, title: 'Grafana', group_id: 2 })], 2)
  const index = buildIndex([media, tools])

  it('a group-name hit passes the whole group through', () => {
    const r = filterIndex(index, 'media')
    expect(r).toHaveLength(1)
    expect(r[0].name).toBe('Media')
    expect(r[0].cards?.map((c) => c.id)).toEqual([1, 2])
  })

  it('drops groups whose cards all filter out', () => {
    const r = filterIndex(index, 'grafana')
    expect(r.map((g) => g.name)).toEqual(['Tools'])
  })

  it('returns every group untouched for an empty query', () => {
    const r = filterIndex(index, '')
    expect(r).toHaveLength(2)
    expect(r[0].cards?.map((c) => c.id)).toEqual([1, 2])
    expect(r[1].cards?.map((c) => c.id)).toEqual([3])
    expect(r[0]).toBe(media) // original object identity preserved
  })

  it('returns everything for a whitespace-only query', () => {
    expect(filterIndex(index, '   ')).toHaveLength(2)
  })
})

describe('pinyin', () => {
  it('reaches a CJK group name by full pinyin and by initials', () => {
    const index = buildIndex([makeGroup('影音', [makeCard({ id: 1, title: 'Jellyfin' })])])
    expect(filterIndex(index, 'yingyin').flatMap((g) => g.cards ?? [])).toHaveLength(1)
    expect(filterIndex(index, 'yy').flatMap((g) => g.cards ?? [])).toHaveLength(1)
  })

  it('reaches a CJK card title the same way', () => {
    const index = buildIndex([makeGroup('Media', [makeCard({ id: 1, title: '影音中心' })])])
    expect(filterIndex(index, 'yingyin').flatMap((g) => g.cards ?? [])).toHaveLength(1)
    expect(filterIndex(index, 'yyzx').flatMap((g) => g.cards ?? [])).toHaveLength(1)
  })

  it('leaves ASCII-only text without phonetic padding', () => {
    expect(cardHaystack(makeCard({ title: 'Plex' }))).toBe('plex')
  })
})

describe('highlightSegments', () => {
  it('returns a single unmarked segment when there are no tokens', () => {
    expect(highlightSegments('Jellyfin', [])).toEqual([{ text: 'Jellyfin', mark: false }])
  })

  it('marks matches case-insensitively and keeps original casing', () => {
    expect(highlightSegments('Jellyfin', ['jelly'])).toEqual([
      { text: 'Jelly', mark: true },
      { text: 'fin', mark: false },
    ])
  })

  it('marks every token', () => {
    expect(highlightSegments('media server', ['media', 'server'])).toEqual([
      { text: 'media', mark: true },
      { text: ' ', mark: false },
      { text: 'server', mark: true },
    ])
  })

  it('treats regex metacharacters as literals', () => {
    expect(highlightSegments('a.b', ['.'])).toEqual([
      { text: 'a', mark: false },
      { text: '.', mark: true },
      { text: 'b', mark: false },
    ])
    expect(highlightSegments('axb', ['.'])).toEqual([{ text: 'axb', mark: false }])
  })

  it('does not mark pinyin-only hits (no literal substring to mark)', () => {
    expect(highlightSegments('影音', ['yingyin'])).toEqual([{ text: '影音', mark: false }])
  })

  it('returns [] for empty text', () => {
    expect(highlightSegments('', ['jelly'])).toEqual([])
  })
})
