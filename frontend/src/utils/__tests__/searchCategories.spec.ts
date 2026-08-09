import { describe, it, expect } from 'vitest'
import {
  CATEGORY_LABELS,
  CATEGORY_ORDER,
  categoryLabel,
  groupEnginesByCategory,
} from '../searchCategories'
import type { SearchEngine } from '@/api/searchEngine'

function makeEngine(over: Partial<SearchEngine> = {}): SearchEngine {
  return {
    id: 1,
    name: 'Google',
    url_template: 'https://www.google.com/search?q={query}',
    icon: '',
    category: 'web',
    is_default: false,
    sort: 10,
    created_at: '',
    updated_at: '',
    ...over,
  }
}

describe('CATEGORY_ORDER / labels', () => {
  it('matches the backend valid set, in display order', () => {
    expect([...CATEGORY_ORDER]).toEqual(['web', 'image', 'music', 'video'])
  })

  it('has a label for every ordered category', () => {
    for (const key of CATEGORY_ORDER) {
      expect(CATEGORY_LABELS[key]).toBeTruthy()
    }
  })

  it('falls back to 其它 for unknown keys', () => {
    expect(categoryLabel('web')).toBe('网页')
    expect(categoryLabel('news')).toBe('其它')
    expect(categoryLabel('')).toBe('其它')
  })
})

describe('groupEnginesByCategory', () => {
  it('orders groups by CATEGORY_ORDER regardless of input order', () => {
    const engines = [
      makeEngine({ id: 1, category: 'video', name: 'YouTube' }),
      makeEngine({ id: 2, category: 'web', name: 'Google' }),
      makeEngine({ id: 3, category: 'music', name: 'Spotify' }),
      makeEngine({ id: 4, category: 'image', name: 'Pinterest' }),
    ]
    expect(groupEnginesByCategory(engines).map((g) => g.key)).toEqual([
      'web',
      'image',
      'music',
      'video',
    ])
  })

  it('labels each group', () => {
    const groups = groupEnginesByCategory([makeEngine({ category: 'image' })])
    expect(groups).toHaveLength(1)
    expect(groups[0]).toMatchObject({ key: 'image', label: '图片' })
  })

  it('sorts within a group by sort ASC then id ASC', () => {
    const engines = [
      makeEngine({ id: 3, sort: 20 }),
      makeEngine({ id: 1, sort: 30 }),
      makeEngine({ id: 2, sort: 20 }),
    ]
    expect(groupEnginesByCategory(engines)[0].engines.map((e) => e.id)).toEqual([2, 3, 1])
  })

  it('skips empty groups', () => {
    const groups = groupEnginesByCategory([
      makeEngine({ id: 1, category: 'web' }),
      makeEngine({ id: 2, category: 'video' }),
    ])
    expect(groups.map((g) => g.key)).toEqual(['web', 'video'])
  })

  it('collects unknown categories into a trailing 其它 group', () => {
    const groups = groupEnginesByCategory([
      makeEngine({ id: 1, category: 'news' }),
      makeEngine({ id: 2, category: 'web' }),
      makeEngine({ id: 3, category: '' }),
    ])
    expect(groups.map((g) => g.key)).toEqual(['web', 'other'])
    expect(groups[1].label).toBe('其它')
    expect(groups[1].engines.map((e) => e.id)).toEqual([1, 3])
  })

  it('returns [] for no engines', () => {
    expect(groupEnginesByCategory([])).toEqual([])
  })

  it('does not mutate the input array order', () => {
    const engines = [makeEngine({ id: 2, sort: 30 }), makeEngine({ id: 1, sort: 10 })]
    groupEnginesByCategory(engines)
    expect(engines.map((e) => e.id)).toEqual([2, 1])
  })
})
