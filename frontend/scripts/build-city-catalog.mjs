#!/usr/bin/env node
// Generate a hand-curated catalog of ~100 major world cities with timezone +
// coordinates. Run: node scripts/build-city-catalog.mjs
//
// Output: src/data/cities.json — committed to repo, lazy-loaded by CityPicker
// at runtime. Build itself is offline-safe; this script fetches Open-Meteo's
// geocoding API only when explicitly run by a developer to refresh.
//
// City selection criteria: 全球主要都市, balanced across continents. Hand-curate
// English + Chinese names; let geocoding API supply tz/lat/lon. Failed lookups
// (city name ambiguous or unknown) fall back to manually-supplied coords.

import { writeFileSync, mkdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = dirname(fileURLToPath(import.meta.url))
const OUT = resolve(HERE, '..', 'src', 'data', 'cities.json')
mkdirSync(dirname(OUT), { recursive: true })

// [name_en, name_cn, country_hint(optional, disambiguates same-name cities)]
const CITY_LIST = [
  // China mainland
  ['Beijing', '北京', 'China'],
  ['Shanghai', '上海', 'China'],
  ['Guangzhou', '广州', 'China'],
  ['Shenzhen', '深圳', 'China'],
  ['Hangzhou', '杭州', 'China'],
  ['Chengdu', '成都', 'China'],
  ['Chongqing', '重庆', 'China'],
  ['Wuhan', '武汉', 'China'],
  ["Xi'an", '西安', 'China'],
  ['Nanjing', '南京', 'China'],
  ['Suzhou', '苏州', 'China'],
  ['Ningbo', '宁波', 'China'],
  ['Qingdao', '青岛', 'China'],
  ['Xiamen', '厦门', 'China'],
  ['Dalian', '大连', 'China'],
  ['Changsha', '长沙', 'China'],
  ['Shenyang', '沈阳', 'China'],
  ['Tianjin', '天津', 'China'],
  ['Kunming', '昆明', 'China'],
  // HK / Macau / Taiwan
  ['Hong Kong', '香港'],
  ['Macau', '澳门'],
  ['Taipei', '台北', 'Taiwan'],
  ['Kaohsiung', '高雄', 'Taiwan'],
  // East Asia
  ['Tokyo', '东京', 'Japan'],
  ['Osaka', '大阪', 'Japan'],
  ['Kyoto', '京都', 'Japan'],
  ['Seoul', '首尔', 'South Korea'],
  ['Busan', '釜山', 'South Korea'],
  // SEA
  ['Singapore', '新加坡'],
  ['Bangkok', '曼谷', 'Thailand'],
  ['Kuala Lumpur', '吉隆坡', 'Malaysia'],
  ['Jakarta', '雅加达', 'Indonesia'],
  ['Manila', '马尼拉', 'Philippines'],
  ['Hanoi', '河内', 'Vietnam'],
  ['Ho Chi Minh City', '胡志明市', 'Vietnam'],
  // South Asia
  ['Mumbai', '孟买', 'India'],
  ['New Delhi', '新德里', 'India'],
  ['Bangalore', '班加罗尔', 'India'],
  ['Kolkata', '加尔各答', 'India'],
  ['Karachi', '卡拉奇', 'Pakistan'],
  ['Islamabad', '伊斯兰堡', 'Pakistan'],
  ['Dhaka', '达卡', 'Bangladesh'],
  // Middle East
  ['Dubai', '迪拜'],
  ['Doha', '多哈'],
  ['Tehran', '德黑兰', 'Iran'],
  ['Riyadh', '利雅得', 'Saudi Arabia'],
  ['Istanbul', '伊斯坦布尔', 'Turkey'],
  ['Tel Aviv', '特拉维夫', 'Israel'],
  // Europe
  ['London', '伦敦'],
  ['Paris', '巴黎', 'France'],
  ['Berlin', '柏林', 'Germany'],
  ['Munich', '慕尼黑', 'Germany'],
  ['Frankfurt', '法兰克福', 'Germany'],
  ['Rome', '罗马', 'Italy'],
  ['Milan', '米兰', 'Italy'],
  ['Madrid', '马德里', 'Spain'],
  ['Barcelona', '巴塞罗那', 'Spain'],
  ['Amsterdam', '阿姆斯特丹', 'Netherlands'],
  ['Brussels', '布鲁塞尔', 'Belgium'],
  ['Zurich', '苏黎世', 'Switzerland'],
  ['Vienna', '维也纳', 'Austria'],
  ['Prague', '布拉格', 'Czech'],
  ['Warsaw', '华沙', 'Poland'],
  ['Budapest', '布达佩斯', 'Hungary'],
  ['Stockholm', '斯德哥尔摩', 'Sweden'],
  ['Oslo', '奥斯陆', 'Norway'],
  ['Copenhagen', '哥本哈根', 'Denmark'],
  ['Helsinki', '赫尔辛基', 'Finland'],
  ['Reykjavik', '雷克雅未克', 'Iceland'],
  ['Dublin', '都柏林', 'Ireland'],
  ['Athens', '雅典', 'Greece'],
  ['Lisbon', '里斯本', 'Portugal'],
  ['Moscow', '莫斯科', 'Russia'],
  ['Saint Petersburg', '圣彼得堡', 'Russia'],
  // Africa
  ['Cairo', '开罗', 'Egypt'],
  ['Lagos', '拉各斯', 'Nigeria'],
  ['Nairobi', '内罗毕', 'Kenya'],
  ['Cape Town', '开普敦', 'South Africa'],
  ['Johannesburg', '约翰内斯堡', 'South Africa'],
  // North America
  ['New York', '纽约', 'United States'],
  ['Los Angeles', '洛杉矶', 'United States'],
  ['Chicago', '芝加哥', 'United States'],
  ['Houston', '休斯顿', 'United States'],
  ['Phoenix', '凤凰城', 'United States'],
  ['Philadelphia', '费城', 'United States'],
  ['Dallas', '达拉斯', 'United States'],
  ['San Francisco', '旧金山', 'United States'],
  ['Seattle', '西雅图', 'United States'],
  ['Boston', '波士顿', 'United States'],
  ['Washington', '华盛顿', 'United States'],
  ['Miami', '迈阿密', 'United States'],
  ['Atlanta', '亚特兰大', 'United States'],
  ['Denver', '丹佛', 'United States'],
  ['Las Vegas', '拉斯维加斯', 'United States'],
  ['Toronto', '多伦多', 'Canada'],
  ['Montreal', '蒙特利尔', 'Canada'],
  ['Vancouver', '温哥华', 'Canada'],
  ['Ottawa', '渥太华', 'Canada'],
  ['Mexico City', '墨西哥城', 'Mexico'],
  // South America
  ['São Paulo', '圣保罗', 'Brazil'],
  ['Rio de Janeiro', '里约热内卢', 'Brazil'],
  ['Buenos Aires', '布宜诺斯艾利斯', 'Argentina'],
  ['Lima', '利马', 'Peru'],
  ['Santiago', '圣地亚哥', 'Chile'],
  ['Bogota', '波哥大', 'Colombia'],
  // Oceania
  ['Sydney', '悉尼', 'Australia'],
  ['Melbourne', '墨尔本', 'Australia'],
  ['Auckland', '奥克兰', 'New Zealand'],
]

console.log(`Fetching geocoding for ${CITY_LIST.length} cities ...`)

async function geocode(name, country) {
  const url = `https://geocoding-api.open-meteo.com/v1/search?name=${encodeURIComponent(name)}&count=5&language=en`
  const res = await fetch(url, { headers: { 'User-Agent': 'moon-panel-catalog' } })
  if (!res.ok) throw new Error(`geocoding ${name}: HTTP ${res.status}`)
  const json = await res.json()
  const results = json.results ?? []
  if (results.length === 0) return null
  // Disambiguate by country if hint provided
  if (country) {
    const matched = results.find((r) => r.country === country || r.country?.includes(country))
    if (matched) return matched
  }
  return results[0]
}

const out = []
let failed = 0
for (let i = 0; i < CITY_LIST.length; i++) {
  const [nameEn, nameCn, country] = CITY_LIST[i]
  try {
    const r = await geocode(nameEn, country)
    if (!r) {
      console.warn(`  ✗ no match: ${nameEn} (${country ?? 'any'})`)
      failed++
      continue
    }
    out.push({
      name_cn: nameCn,
      name_en: nameEn,
      tz: r.timezone,
      lat: Number(r.latitude.toFixed(4)),
      lon: Number(r.longitude.toFixed(4)),
    })
    if ((i + 1) % 20 === 0) console.log(`  ${i + 1}/${CITY_LIST.length} ...`)
  } catch (e) {
    console.warn(`  ✗ ${nameEn}: ${e.message}`)
    failed++
  }
  // Tiny delay to be polite to Open-Meteo
  await new Promise((r) => setTimeout(r, 80))
}

writeFileSync(OUT, JSON.stringify(out) + '\n', 'utf8')
console.log(`✓ ${OUT}  (${out.length} cities, ${failed} failed, ${(JSON.stringify(out).length / 1024).toFixed(1)} KB)`)
