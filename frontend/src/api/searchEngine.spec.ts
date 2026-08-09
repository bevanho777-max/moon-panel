import { describe, it, expect } from 'vitest'
import { buildSearchURL } from './searchEngine'

// buildSearchURL is the only place a user's query becomes a URL, and the
// {q}/{query} placeholder contract is duplicated in Go (search_engine.go
// placeholderPattern) — worth pinning on both sides.
describe('buildSearchURL', () => {
  it('substitutes {q}', () => {
    expect(buildSearchURL('https://example.com/s?q={q}', 'moon')).toBe(
      'https://example.com/s?q=moon',
    )
  })

  it('substitutes {query}', () => {
    expect(buildSearchURL('https://www.google.com/search?q={query}', 'moon')).toBe(
      'https://www.google.com/search?q=moon',
    )
  })

  it('substitutes both placeholders in the same template', () => {
    expect(buildSearchURL('https://e.com/{q}?x={query}', 'moon')).toBe(
      'https://e.com/moon?x=moon',
    )
  })

  it('replaces every occurrence of the same placeholder', () => {
    expect(buildSearchURL('https://e.com/{q}/{q}', 'moon')).toBe('https://e.com/moon/moon')
  })

  it('encodes spaces', () => {
    expect(buildSearchURL('https://e.com/s?q={q}', 'moon panel')).toBe(
      'https://e.com/s?q=moon%20panel',
    )
  })

  it('encodes CJK', () => {
    expect(buildSearchURL('https://www.baidu.com/s?wd={query}', '月亮')).toBe(
      'https://www.baidu.com/s?wd=%E6%9C%88%E4%BA%AE',
    )
  })

  it('encodes & and = so the query cannot inject extra params', () => {
    expect(buildSearchURL('https://e.com/s?q={q}', 'a&b=c')).toBe(
      'https://e.com/s?q=a%26b%3Dc',
    )
  })

  it('returns the template unchanged when it has no placeholder', () => {
    expect(buildSearchURL('https://e.com/static', 'moon')).toBe('https://e.com/static')
  })

  it('leaves an empty query as an empty substitution', () => {
    expect(buildSearchURL('https://e.com/s?q={q}', '')).toBe('https://e.com/s?q=')
  })
})
