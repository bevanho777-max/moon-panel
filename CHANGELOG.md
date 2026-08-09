# Changelog

All notable changes to Moon Panel are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.30] - 2026-08-09

### Added

- 首页搜索升级为多词 AND 全文搜索 — 「jelly media」两词都命中才显示, 空格
  分词, 逐词收窄而不是放宽
- 拼音搜索 (pinyin-pro): 中文字段额外索引【全拼】+【首字母】, 分组「影音」
  可用 `yingyin` 或 `yy` 搜到。只对含 CJK 的字符串跑转换
- 命中高亮: 卡片标题 / 描述里的命中片段用 `<mark>` 标出。分段渲染成文本
  节点, 全程不用 `v-html`
- 搜索内核抽成 `frontend/src/utils/cardSearch.ts` (`tokenize` /
  `matchesTokens` / `cardHaystack` / `buildIndex` / `filterIndex` /
  `highlightSegments`), 23 条单测覆盖

### Changed

- 首页与「管理后台 → 卡片」共用 `cardSearch` 底层原语 (`tokenize` +
  `matchesTokens` + `cardHaystack`) — 两处原本各写一份 `includes()` 链, 现在
  分词 / AND / 拼音行为一致。后台仍在本地建 haystack (扁平卡片表, 分组名要
  按 `group_id` 从 store 取), 搜索字段不变
- `CardItem` 新增可选 prop `highlightTokens` (默认 `[]`) — 其它调用点行为不变

### Internal

- 索引与查询分离: `buildIndex` 只随 panel 数据变化重建 (拼音转换在这一步),
  按键只跑 `filterIndex` 的子串比对, 拼音开销不上按键路径
- `filterIndex` 空查询返回原分组对象本身 (保持引用), 避免 keyed transition
  无谓抖动
- `highlightSegments` 对正则元字符做转义, 查 `.` 不会匹配任意字符
- 依赖: 新增 `pinyin-pro ^3.28.2` (dependencies)。它被打进独立的
  `cardSearch` chunk (~139 kB gzip), 首页与后台卡片页按需加载

## [0.2.29] - 2026-08-09

### Added

- 内置搜索引擎扩充到 7 个 — 新增 Brave / Startpage / Yandex (隐私向补位),
  原有 Google / Bing / DuckDuckGo / 百度 不变
- 后端 `POST /api/admin/search-engines/restore-builtins` — 补回被删除的内置
  引擎。增量语义: 按 name 比对, 已存在的一律跳过 (用户改过的条目不被覆盖),
  新插入的 `is_default=false` + `sort` 落到列表底部, 整体单事务
- 管理端「站点设置 → 搜索引擎」加「恢复内置引擎」次要按钮 — 补了 N 个提示
  「已补充 N 个内置引擎」, N=0 提示「内置引擎已齐全」

### Fixed

- `HeaderSearchBox`: 首页搜索框不认 `lucide:` 图标, 会渲染成 `?` — 现与
  卡片 / 管理端预览一致支持 `http(s)://` + `upload:` + `lucide:` 三种编码

### Internal

- 内置引擎列表抽成单一真源 `api.BuiltinSearchEngines()` — 开服 seed
  (`bootstrapDefaultEngines`) 与 restore 端点共用, 不再各写一份
- 测试: 新增 `backend/internal/api/search_engine_test.go` (builtin 列表纯函数
  断言: 数量 / 唯一默认 / name 唯一 / https + 占位符) 与
  `frontend/src/api/searchEngine.spec.ts` (`buildSearchURL` 占位符替换与
  encode 行为)
- `bootstrapDefaultEngines` 空表判断逻辑不变 — 已 seed 过的部署升级后不会
  被重新塞引擎

## [0.2.28] - 2026-05-22

### Internal

- model: Group/Card 加 `OwnerID` 字段 (纯 index, FK 留 R6) — multi-user
  数据归属地基
- store: 新增 `MigrateOwnerID` — 现有数据 atomic 迁移到默认 user (幂等,
  `WHERE username = "admin"` 不假设 ID)
- backup: restore 后补跑 migration (防旧 backup 数据无主)
- ci: 加 `go test` 步骤 (项目首引 Go test, Linux 首跑通过)
- 用户无感: 前端 0 改动, Login/UI/数据展示行为不变

### Notes

- A.5 multi-user roadmap 第 1 步 (地基, 功能未启用)
- 升级说明见 [docs/MIGRATION.md](docs/MIGRATION.md) (建议升级前 backup)

## [0.2.27] - 2026-05-22

### Changed

- frontend: 星空呼吸微调 — 周期 15s → 8s, 星星不再像定在原位
- frontend: 星星明暗对比拉大 (`base` 0.15-0.40 / `amp` 0.35-0.80) — 部分
  星完整经历熄灭→全亮, 此起彼伏的忽隐忽现感
- 起因: v0.2.26 真机观感星星"像定在原位", 缺流动感

### Internal

- 仅 `StarfieldBackground.vue` 3 处常量微调 (流星不动, Bevan 反馈"非常棒")
- alpha clip `[0, 1]` 已有 — 谷值负数 clip 到 0 = 星星熄灭效果
- 视觉门: 真机当视觉门 (同 v0.2.26)

## [0.2.26] - 2026-05-22

### Changed

- frontend: 动态背景从整屏呼吸+光晕 (v0.2.25) 改为星空粒子 — 起因 v0.2.25
  真机观感"圈圈放大不自然 + 大范围呼吸不适"
- frontend: 星仅 opacity 呼吸 (15s 周期, phase offset 错开), 不再 scale
  放大

### Added

- frontend: `StarfieldBackground.vue` (canvas 独立组件) — 26 颗星稀疏呼吸
  + 偶发流星 (右上→左下, 20-40s)
- frontend: 双主题星色 `--mp-star-*` (Moon 冷白 / Risen 暖金)

### Removed

- frontend: v0.2.25 aurora/glow 整屏动画 (App.vue + main.css ~104 行,
  含 `mpAuroraFlow` / `mpGlowBreathe` keyframe + `--mp-bg-*` token)

### Internal

- 工程首引 `requestAnimationFrame` + 首个 `<canvas>` 渲染组件, 建立
  cleanup pattern (RAF id ref + `onBeforeUnmount` cancel +
  `visibilitychange` 暂停)
- 性能: 限帧 30fps (时间戳 delta) / `document.hidden` 暂停 / DPR cap 2
- reduced-motion JS `matchMedia` 静态星空 (OS 运行时切换支持)
- canvas `isolation: isolate` 保留 (Login card acrylic 不采样, 双保险)
- canvas +3KB minified (vs v0.2.25 整屏 gradient, 体积/性能更轻)
- admin `v-if` 禁星空 (复用 v0.2.25 `isAdminRoute`)
- 视觉门: 本地 dev 缺 backend, 真机当视觉门 (同 v0.2.25)

## [0.2.25] - 2026-05-21

### Added

- frontend: Home + Login 动态背景 (CSS-only, 双主题定制) — 两层动画:
  渐变流动 (aurora translate) + 光晕呼吸 (glow scale+opacity), 周期错开
- frontend: Moon 蓝紫冷调 (aurora 22s / glow 10s) / Risen 暖金沉稳
  (26s / 12s) 双主题 token (`--mp-bg-*`)
- a11y: `prefers-reduced-motion` 静态 fallback (全工程首引, WCAG 2.3.3)

### Changed

- docs: CLAUDE.md stale 数字清理 (#46a 根因, b6dfe8c)

### Internal

- 动态背景层 `isolation: isolate` 隔离 stacking context — Login card
  acrylic (mp-acrylic-strong) 不采样动画层, 避开 INP 退化
  (v0.2.x acrylic 教训)
- admin 路由 `v-if` 条件渲染禁动效 (非 CSS hide, 0 paint, Cards drag
  INP 优先)
- 动态层 `aria-hidden` — 不进 a11y selector tree, e2e selector 全保留
- 视觉门方式: 本地 dev 缺 backend + 缺壁纸, 真机 (NAS) 当视觉门
- Flag #49: CLAUDE.md 被 PS Invoke-WebRequest 响应污染 (+89 行垃圾,
  UTF-16 编码), revert 止血, 未进 history
- 开关推迟 v0.2.26 (单一职责: 本 release 仅动效, 不含设置系统)

## [0.2.24] - 2026-05-20

### Added

- frontend: NRadioGroup 智能自动跟随 URL 输入 (A1) — 用户输入 url_internal /
  url_external 时, default radio 智能跟随; 用户手动选过后停止自动跟随; 编辑态
  尊重历史选择不自动改
- backend: `url_default` 字段接受空字符串 `''` (允许 "无 default" 状态, 配合
  A1 nullable 语义, 用空字符串语义代替 SQLite nullable migration)
- frontend: `Card` type union 扩 `'' | 'internal' | 'external'`
- ci: GitHub Actions 升级 9 actions (checkout@v6, setup-node@v6 Node 22,
  setup-go@v6, docker/* 全升, softprops@v3)
- tests: cardUrl D3 WAN-strict 4 新 vitest case (currentMode / fallback)

### Changed

- frontend: emptyForm `url_default` 默认值 `'internal'` → `''` (新建态不预选)
- frontend: mobile @media `.cards-cell__icons { display: none }` 隐双 URL
  icons (free ~60px for title)

### Fixed

- frontend: mobile 320 viewport admin Cards title 可用宽 0px silent bug (B.4
  Flag #34, 量化推断 + A patch 3 行修)
- tests: cardUrl D3 WAN invariant 改变后 2 旧 vitest case regression (A.6
  Flag #31, makeCard 加 `url_default: 'external'` 显式禁用 fallback)

### Internal

- 12 flag 全 surface, 0 擅自决定 (#27–#38)
- 6 次工程纪律真实救场:
  - #30 nullable 推翻 → X2 空字符串语义 (4 行 vs 30-50 行 + DB migration risk)
  - #31 vitest 旧 case regression A.6 抓到 + makeCard 修
  - #33 emptyForm protocol prefix 实质空判断式复用 (spec 救场)
  - #34 mobile silent bug 量化推断 + A patch 3 行修
  - Phase C Task 0 grep 整合 (#36 nspace 重叠 / #37 README 无英文 / #38
    CLAUDE.md inline 互补) 全 scope 验证
- 7 连 0-patch streak 候选 (v0.2.18 起算, 真实数字以 git log v0.2.17–23 verify)
- 4 新 memory/feedback_*.md (`spec_placeholder` / `silent_render_failure` /
  `browser_api_limits` / `unverified_prod_work`) + 1 existing 扩
  (`naiveui_deep_override` 加 NSpace v2 baseline section)

## [0.2.23] - 2026-05-16

### Added

- **Auto network detection** — Home page now automatically detects whether
  the client is on the local LAN or remote network and switches dual-URL
  cards (cards with both internal and external URLs) to the appropriate
  target. Detection uses an admin-configurable probe URL (new
  `network.probe_url` setting) with fallback to a sampled card's internal
  URL. The probe runs on mount, on browser `online`/`offline` events, and
  every 60 seconds while the page is visible (paused when tab is hidden,
  resumed on `visibilitychange`).

- **NetworkSwitcher status indicator** — When in "自动检测" mode, a small
  dot indicates the current detection state: green for LAN, orange for
  WAN, grey pulsing for detecting. A 1×1 spinner appears next to the
  switcher during active probing. Mobile dropdown trigger shows a small
  dot in the bottom-right corner with the same color semantics.

- **Dual-URL badge — visual state** — Cards with both internal and
  external URLs now show their current routing target in the badge:
  blue with 🏠 icon when going LAN, orange with 🌐 icon when going WAN,
  grey with spinner when detecting.

- **WAN-strict routing for LAN-only cards** — When detection determines
  the client is on WAN, cards with only an internal URL become
  non-clickable (`card-item--disabled`) with tooltip "外网环境无法访问此
  卡片（仅有内网 URL）". This prevents the dead-link behavior where users
  on mobile data accidentally tap LAN-only services. The new
  `resolveWANStrict` helper in `cardUrl.ts` enforces the asymmetric
  fallback rule (LAN falls back to external, WAN does not fall back to
  internal).

- **Session-scoped manual override** — Users can temporarily force LAN
  or WAN mode in NetworkSwitcher; this override persists for the current
  browser session only (sessionStorage, key
  `moon-panel.session-override`). On refresh, behavior returns to
  auto-detection. The existing permanent `internal`/`external`
  localStorage preferences (the "global mode" setting) are unaffected
  and coexist.

- **Admin setting: 内网检测 URL** — New "网络检测" card in SiteSettings
  for configuring an explicit probe URL. Validates http(s):// prefix and
  length ≤ 200 in both frontend and backend. Empty value triggers
  auto-sampling from the first card with an internal URL. The setting
  is exposed to anonymous Home clients via `/api/public/panel`
  (`site.network.probe_url`).

### Changed

- **NetworkSwitcher "自动" mode semantics** — Previously, "自动 · 跟随
  各卡默认" meant "follow each card's `url_default` field". v0.2.23
  redefines it as "自动检测" — detect and switch globally based on
  network reachability. The `url_default` field is retained for backward
  compatibility but is no longer read by `cardUrl.ts`. The default
  network mode (`'auto'`) is unchanged from v0.2.22; existing users keep
  whatever localStorage value they previously set.

- **`/api/health` Cache-Control** — Now returns `Cache-Control: no-store`
  to prevent stale probe responses from being cached by browsers or
  intermediate proxies (previously had no Cache-Control header).

### Fixed

- **`NInput` not registered in SiteSettings.vue** — The `NInput`
  component was used in `SiteSettings.vue` template but not imported
  from `'naive-ui'`. Browsers silently rendered `<NInput>` as an
  unknown 0×0 element. v0.2.23 adds the missing import, which also
  restores the "站点名称" (site title) input field that has been
  silently broken since v0.2.0 — users couldn't change the panel title
  for the entire v0.2.x series. This was discovered during v0.2.23
  visual-gate testing of the new `内网检测 URL` input.

### Internal

- **Network probe via `<img>` tag** — The auto-detection probe uses an
  `<img>` element rather than `fetch()`. `<img>` is unconstrained by
  CORS or redirect modes, so any server response (image, HTML, 301,
  404, JSON) triggers `onload` or `onerror` and counts as reachable.
  Only `setTimeout` firing first signals genuine unreachability. This
  sidesteps Chrome's `fetch(mode:'no-cors')` restriction that forbids
  `redirect:'manual'`, and avoids the `redirect:'follow'` flake where
  a LAN reverse-proxy 301→https follow-up request fails for unrelated
  reasons (false-WAN false-positive).
- **Zero new npm dependencies** — Detection uses browser-native APIs
  only (`Image`, `setTimeout`, `sessionStorage`); no new entries in
  `package.json`.
- **Zero database migration** — `model.Setting` generic key→value table;
  `bootstrapNetworkSettings()` follows the same pattern as
  `bootstrapWidgetSettings/DefaultEngines/SessionFloor/TrustedIPs`.
- **`cardUrl.ts` refactor** — `effectiveURL(card, state)` now accepts
  `CardURLState {override, effectiveMode}` and returns
  `EffectiveURL {side, fallback}` rich object instead of a plain string.
  The richer return value enables the dual-URL badge to display the
  current routing target and the tooltip to differentiate "no URL set"
  from "WAN environment, internal only". 14 vitest cases cover the new
  matrix, including 2 dedicated WAN-strict assertions.
- **E2E mock alignment** — `phase-3c-2`, `phase-4a`, and
  `readme-screenshots` fixtures now include `site.network.probe_url`.
  Tests that load Home (`phase-3c-2`, `readme-screenshots`) pin
  `sessionStorage['moon-panel.session-override'] = 'lan'` via
  `addInitScript` before navigation to avoid probe-failure flake in
  the headless environment.
- **Bundle delta** — +2.3 KB gzipped (JS +1.9, CSS +0.4) across
  Home/SiteSettings/NetworkSwitcher chunks. No new chunk-size warnings.

## [0.2.22] - 2026-05-11

### Changed
- **NaiveUI Input/Select/Button 主题切换适配** (Bevan daily UX 反馈, 截图对比):
  "不管在什么主题下, 图片里的搜索框和自动跟随框都是默认, 能否随着主题变化?"
  扩展 frontend/src/stores/ui.ts themeOverrides computed, 添加 Input /
  InternalSelection / Button 字段, 让全 app NaiveUI 表单元素跟随主题切换 (Risen
  暖棕金 / Moon 蓝灰). NaiveUI cssr inject CSS var 字符串, 浏览器 paint 时
  解析 + data-theme 改变自动 invalidate (D.3.b 决策, 0 reactivity hack).
  - `frontend/src/stores/ui.ts`: themeOverrides computed 扩展 ~+40-50 行,
     添加 Input / InternalSelection / Button 字段.
  - `--mp-search-*` dead token (Moon + Risen 各 3) 重启用 (跟 v0.2.20
    `.mp-acrylic-light` 规则保留同思路).
  - Focus 状态 fallback: `var(--mp-search-bg-focus, var(--mp-search-bg))` —
    future-proof, focus token 未来加时 ui.ts 不需改 (γ 方案).
  - main.css 不动: 严守 D.3 single source of truth + 现有 3 token 复用.
  - 全 app 影响 (B.1): NInput / NSelect / 非 primary NButton 全跟主题, 包括
    Admin Cards/Groups/Settings forms, WallpaperPicker, CityPicker, Modals 等.
  - Login primary button (type="primary") 保留 colorPrimary 联动, 不被
    surface override 影响.

### Engineering
- V2 教训实战第六次延续 (e2e test 0 影响, NaiveUI themeOverrides 是 NaiveUI
  API 标准做法, 不破 e2e BEM selector).
- ★ Claude Code 主动 flag 累积模式 第 4+5 次成功 ★:
  * 第 4 次 (D.3 决策, Task 2.2 实施前): spec 草案 getCssVar + reactive hack
    时序脆性 (4 步 async 链), 主动提供 D.3.b (CSS var 字符串 cssr inject) 更优解.
  * 第 5 次 (D.3.b spec, Task 2.2 实施前): spec 草案 2 个客观技术错误 (Issue 1:
    Button.colorActive 不存在, 应 colorPressed; Issue 2: --mp-search-bg-focus
    token 未定义), verify NaiveUI 源码 + main.css 抓出. γ 方案 fallback 解决.
  * 跟 v0.2.19/2.20/2.21 累积 root cause flag 一脉相承 (现在 5 次累积).
- Claude Code 主动 root cause 诊断: R1 + R3 混合 root cause — themeOverrides
  仅响应 themePrimary 不响应 themePreset + 6 个 `--mp-search-*` dead token 已存在
  但无消费方.
- Task 2.1 前置 verify (跨文件 grep hh-icon-button + NInput/NSelect/NButton):
  避免 themeOverrides Button 字段覆盖现有自定义 button, 第 2 次累积应用.

## [0.2.21] - 2026-05-11

### Changed
- **Home 用户端 group 容器扁平化** (Bevan daily UX 反馈, 截图对比):
  "卡片栏只保留这些卡片的样式, AI流之类的标题不需要再外框和背景, 这样
  看起来感觉会更舒服和整洁、直观". 删 `.home-group` 容器 background /
  border / border-radius / padding (PC + mobile), 保留 margin-bottom 作
  group 间距. `.home-group__title` (folder icon + 分组名 + border-bottom
  divider) 保留作分组识别. 跟 v0.2.13 HomeCard 扁平化 + v0.2.20 Weather
  删 acrylic 同理念 (去除多余视觉层级, 更整洁直观).
  - `frontend/src/views/Home.vue`: `.home-group` PC + 2 个 mobile @media
    规则简化, 删 bg/border/border-radius/padding. 净 -16 行 CSS.
  - `.home-group` class 保留 (e2e 10 处 selector 不破, V2 教训应用第五次延续)
  - `--mp-group-*` token 保留在 main.css (备用给未来组件, 跟 v0.2.20
    `.mp-acrylic-light` 规则保留同思路)

### Engineering
- V2 教训实战第五次延续 (e2e test 0 影响, 主动保留 `.home-group` class
  避免 10 处 selector 同 commit 修).
- Claude Code 主动 Bevan 反馈解读分析 (跟 v0.2.19/2.20 主动 flag root cause
  一脉相承): 你反馈 "标题不需要外框和背景", 实际是 `.home-group` 容器删
  (容器 contains title, title 本身无 bg/border). X1 修法精准对应期望.

## [0.2.20] - 2026-05-11

### Changed

- **Weather card 重新布局** (P0 a, Bevan backlog 累积反馈): HomeHero 弃 flex,
  改 CSS Grid 显式列数 (PC 3-up + max-width 900px 居中, Mobile 2-up). 5 cities
  时第 2/3 行 cards 默认左对齐 (Grid 默认行为).
  - `frontend/src/components/HomeHero.vue`: `grid-template-columns: repeat(3, 1fr)`
    (PC) + `repeat(2, 1fr)` (mobile @media), 删 flex-wrap / :deep(.cw) 旧代码.
    净 -17 行 (CSS 简化).

- **Weather card 颜色 (purple cast → Risen gold)** (P0 b, Bevan 长期反馈
  "purple → warmer gold for Risen"): Root cause 修正 (Claude Code 主动 flag,
  v0.2.x 第 2 次累积): purple cast 不是 weather token 问题 (token 已是 Risen
  gold), 是 `.mp-acrylic-light` global class CSS specificity 覆盖, 加上
  backdrop-filter saturate(180%) 透出 wallpaper purple cast. 修法极简:
  CityWidget.vue 行 112 删 `mp-acrylic-light` class (1 字符), 让 weather
  token 直接生效 (Risen gold / Moon 白透, 跟主题切换彻底).
  - `frontend/src/components/CityWidget.vue` 行 112:
    `<div class="cw mp-acrylic-light">` → `<div class="cw">`
  - Moon 主题一并改 (token 跟随, 视觉接近原 acrylic)
  - main.css 不动 (token 已设计完整, .mp-acrylic-light 规则保留供别处用)

### Engineering
- V2 教训实战第四次延续 (e2e test 0 影响, Task 1 grep audit 5 类 selector
  全清, 0 处验证 grid 列数 / acrylic background).
- Claude Code 主动 flag spec 假设错误 (第 2 次累积, 跟 v0.2.19 一脉相承):
  P0 b spec 假设 weather token 是 purple, 实际 grep 揭示 token 已 gold, 是
  acrylic class CSS specificity 覆盖. 修法从"加新 token"简化为"删 1 字符 class".
- Task 2.1 前置 verify (跨文件 grep `mp-acrylic-light`): 仅 CityWidget 1 处使用,
  main.css 3 处仅规则定义保留. 安全删, 不影响别处.

## [0.2.19] - 2026-05-10

### Fixed

- **新建卡片/分组/搜索引擎默认放在底部** (Bevan v0.2.18 真机验收 daily UX
  反馈: "新建卡片默认是在第一行, 我建议新建卡片放在最后一行").
  根因: backend createHandler 已有 max+10 fallback (v0.2.0+ 设计如此),
  但 frontend emptyForm 传 `sort: 0` 短路 nil check, 导致新建项 sort=0
  插入列表顶部.
  修法 (Y1, 后端 nil check 改进, scope 极简 ~3 行 Go 跨 3 文件):
  `if req.Sort != nil` → `if req.Sort != nil && *req.Sort > 0` (sort=0
  视为"未提供" → 走 max+10 fallback). 应用 3 栏目一致 (Cards/Groups/
  Search Engines) daily UX 期望.
  - `backend/internal/api/card.go` (createCard nil check)
  - `backend/internal/api/group.go` (createGroup nil check)
  - `backend/internal/api/search_engine.go` (createSearchEngine nil check,
    不动 updateHandler 行 221 — update 时 sort=0 是用户明确意图)

### Engineering
- V2 教训实战第三次延续 (e2e test 0 影响, Task 1 grep audit 5 类 selector
  全清, 0 处验证新建顺序). Backend 改动模式跟 v0.2.16 search_engine reorder
  API 一致. 0 patches 一次到位预期 (跟 v0.2.14/2.17/2.18 模式).

## [0.2.18] - 2026-05-09

### Added

- **`<SortableTable>` 抽象组件** (Rule of Three 时机, v0.2.15+v0.2.16 累积 3 用例
  成熟): `frontend/src/components/SortableTable.vue` (~165 行) 抽象 admin Cards /
  Groups / Search Engines 三栏目共性 80% (handle / drag UX / animation / disabled /
  group-name / show-group-headers / itemFilter / .sortable-table BEM), 差异 20% 由
  `#item` slot 调用方自己渲染 cells. 数据结构统一 `[{id, name, items}]` (single
  list 1 行 computed 包装). 后续新栏目接入快 (10-20 行调用).

### Changed

- **三栏目接入 SortableTable**:
  - `frontend/src/views/admin/Cards.vue` 接入 (保留 P0 b localStorage 分组记忆 +
    跨分组拖 + search disable + NTag(group) + dual-URL + sort + actions). cardsByGroup
    nested 字段 `cards` → `items` (跟 SortableTable interface 直接对齐). itemFilter
    prop 替代原 v-show search filter.
  - `frontend/src/views/admin/Groups.vue` 接入 (保留 groupsStore.invalidate +
    ID + sort + actions, single list 包装 `[{id:0, items: groups.value}]`).
  - `frontend/src/views/admin/SiteSettings.vue` 搜索引擎接入 (保留 ⭐/☆ Star +
    URL 模板 PC + sort + actions, single list 包装).
  - BEM 完全迁移 `.sortable-table__*` (跟 v0.2.17 V2 教训实战延续).

- **E2E test selector swap (V2 教训实战第二次延续, 同 commit 修)**: 5 处
  `.engines-list__item` → `.sortable-table__item` (phase-3c-2 + phase-4b),
  应用 feedback_e2e_test_sync_v2.md + accumulated_lesson_e2e_chore_pattern.md
  教训, 不进 chore commit 循环 (跟 v0.2.17 V2 实战首次模式一致).

### Stats
- Net: ~-50 行 (5 文件 +175/-390 + 1 新 SortableTable.vue ~165 行)
- Bundle minified: -3.62kB total (Cards/Groups/SiteSettings CSS 大幅减小, BEM 抽走)
- Build: 6.99s ★ 历史最快 ★

## [0.2.17] - 2026-05-09

### Changed

- **E2E test 第三方 internal class 迁移自有 BEM** (V2 教训应用第一次实战, 应用
  v0.2.16 chore memory feedback_e2e_test_sync_v2.md +
  feedback_e2e_test_avoid_third_party_internal.md): audit-logs/security 7 处
  `.n-data-table-tr` (NaiveUI internal class) 迁移自有 BEM, 防 NaiveUI 升级 /
  主题改造时 e2e 测试再触发 stale (跟 v0.2.15+v0.2.16 累积 chore 模式不同, 这次
  预先彻底修).
  - audit-logs (6 处): `.n-data-table-tr` → `.al__table--desktop tbody tr`
    (复用 AuditLog.vue 已有 BEM wrapper)
  - security (1 处): `.n-data-table-tr` → `.sec__table tbody tr`
    (Security.vue 加 `class="sec__table"` wrapper, 跟 .al__table--desktop
    平行 BEM)
  - 空状态测试: `.n-data-table` → `.al__table--desktop` (整 wrapper, 因空
    状态 tbody 内无 tr)
  - phase-4b.spec.ts:60 过期注释删 (顺手清理 v0.2.17 ship 后的过期文档).

## [0.2.16] - 2026-05-08

### Changed

- **Admin Groups 主表格直接拖拽** (P0 a, X3 方案, Bevan v0.2.15 反馈"分组的排序
  还是需要手动调整, 无法拖拉调整"): 弃 NDataTable, 改 div + vuedraggable inline
  drag (.groups-list BEM, 跟 v0.2.15 Cards 一致). 拖动 ⋮⋮ handle 调序 (lucide
  GripVertical icon). 拖完 @end 立即 PUT /admin/groups/reorder (auto-save).
  groupsStore.invalidate() 同步 Cards 下拉列表. 编辑表单删 NInputNumber 排序
  权重字段 (跟 v0.2.13 admin Cards 一致, 单一 sort UX).

- **搜索引擎主表格 PC + Mobile 直接拖拽** (P0 b, X3 + 后端新加): 弃 NDataTable
  (PC) + 弃 .engines-mobile-list (Mobile v0.2.14), 改统一 .engines-list BEM
  (跟 Groups + Cards 一致). PC + Mobile 都加 ⋮⋮ inline drag handle. ★ Backend
  新加 PUT /api/admin/search-engines/reorder ★ (跟 group reorder 模板, +57
  行 Go, max items 100). frontend reorderSearchEngines API client. ⭐/☆ Star
  icon 默认保留 (v0.2.14 模式, PC + Mobile 统一). URL 模板 PC 显示 (mobile
  @media 隐藏).

- **Home 卡片 dual-URL 标识** (Patch P1, M1.A.D4, Bevan v0.2.16 视觉门反馈
  "内外网都填入时, 没有显示"): CardItem.vue networkBadge computed 加 'both'
  state, template 加 v-else-if 渲染 "双网" badge (复用 homeBadgeIcon +
  globeBadgeIcon, .card-item__badge--both BEM 跟 --internal/--external 同
  pattern, 灰色 affordance hint). 修复 v0.1.0 起的 design 缺失 (双 URL 时
  无视觉标识, 用户无 affordance). 不是 v0.2.x 退化, git log --all 验证从
  initial release 就这样 design.

### Removed

- **GroupsSortModal.vue 整文件删除** (-150 行): X3 inline drag fully replace
  modal (跟 v0.2.15 CardsSortModal 删除一致). "调整顺序" 按钮 + sortOpen ref
  + import 全清理.

- **e2e test 07-groups-sort-modal 删除** (phase-4a.spec.ts:166, -22 行): 同
  commit 跟 product 改 (应用 v0.2.15 教训 feedback_e2e_test_sync.md, 不重复
  v0.2.15 Patch A 单独 chore commit 修).

## [0.2.15] - 2026-05-08

### Changed

- **Admin Cards 主表格直接拖拽** (P0 a, X3 方案, Bevan 反馈"卡片管理排序需要在
  界面内直接调整, 不要点按钮 → 弹 modal"): 弃 NDataTable, 改 div + vuedraggable
  统一 PC + mobile 模板 (.cards-list, @media 切 column 显示). PC 主表格行直接
  拖动 ⋮⋮ handle 调序 (lucide GripVertical icon). ★ 跨分组拖 ★ (B.1b, Bevan
  视觉门 Patch 1 反馈"卡片要从一个分组直接拖到另一个分组"): 所有 group 同
  :group="'cards'", 卡片可拖到任意分组. 拖完 @end 收集所有 group 当前状态
  (cardsByGroup nested array, vuedraggable 跨容器自动 splice + push), 重算
  sort + group_id, 立即 PUT /admin/cards/reorder (auto-save C.1, 失败 reload
  rollback). backend reorderCards API 已原生支持 group_id? 跨分组参数 (Task 1
  grep 确认), 零后端改动. 拖完 reload 同步 cards.value 内 group_id 字段. search
  时 disable draggable (handle 灰色 + tooltip "清空搜索后可拖动").

- **新建卡片分组字段记忆** (P0 b, Bevan 反馈"建立多个卡片时, 自动选择最后选择
  的分组"): 上次新建卡片时选择的分组 ID 存 localStorage (key:
  moon.admin.cards.last_group_id, 符合 moon.* 命名约定 — see
  memory/feedback_localstorage_naming.md). 下次 openCreate 读取, 自动预选
  (vs v0.2.14 永远 reset 第一个分组). 提交成功后 (仅新建路径) 存最新值; 编辑
  模式不动 localStorage. 验证 lastGroupId 仍在 groupsStore 内 (防分组被删后
  无效, fallback groups[0]?.id ?? 0).

### Removed

- **CardsSortModal.vue 整文件删除** (-221 行): 主表格直拖 fully replace modal
  功能, 单一 sort UX 来源. "调整顺序" 按钮 + sortOpen ref + import + 用法
  全部清理 (Cards.vue grep 0 残留).

## [0.2.14] - 2026-05-08

### Changed

- **Admin 顶栏 mobile 右对齐**: ☰ + 🏠 button 漂到右侧 (vs v0.2.13 紧贴 title 左侧).
  mobile @media 单 line CSS: .admin-header { justify-content: space-between }.
  Root cause: PC 用 .admin-header__menu { flex: 1 } 充当 spacer, mobile 隐藏后
  spacer 消失. PC 端不受影响 (flex:1 占满空间, justify-content 失效). Bevan
  v0.2.13 真机反馈"☰ + 🏠 mobile 想要在最右边而不是紧贴 title".

- **搜索引擎 mobile 卡片列表**: 复用 v0.2.13 admin Cards Patch 2/3 1 行 pattern.
  PC ≥769px 保持 NDataTable 6 列 (图标/名称/URL 模板/默认/排序/操作);
  ≤768px 切换为紧凑 1 行卡片 (icon 22 + name flex + ⭐/☆ 默认 + 编辑/删除).
  "默认" 列从 NTag/Button 改 lucide Star icon (实心金色 = is_default,
  空心 = 设为默认), 节省横向 ~52px → name 可用 ~77px (vs naive 复用 25px).
  renderEngineIcon size 参数化 (PC 24, mobile 22). URL 模板 + 排序数字 mobile
  不显示 (进编辑表单看). Bevan v0.2.12 真机反馈"搜索引擎 mobile 看不见编辑/删除".

## [0.2.13] - 2026-05-08

### Changed

- **Admin Cards 移动端**: 新增 mobile-only 卡片列表 (mirror v0.2.6 AuditLog 模式).
  PC ≥769px 保持 NDataTable 5 列 (标题/分组/内外网/排序/操作); ≤768px 切换为
  极致紧凑 1 行卡片. **Patch 1+2+3 (Bevan 三轮迭代)**: 从 3 行布局逐步压缩到
  1 行 flex (icon 22 + title 0.85rem + 内外网 + 编辑+删除), 删除 mobile 分组 NTag
  显示, padding 6 上下 / 10 左右, line-height 1.2. 高度 ~40-42px (vs 原 3 行
  ~150-180px, **-75%**), 一屏可见 ~13-15 张卡片. NButton small (28px) +
  dual-URL (28×28) 是高度物理天花板. PC NDataTable + getGroupName + NTag
  import 全保留, renderIconThumb 加 size 参数 (PC 默认 28, mobile 22), 零回归.

- **Admin Cards 编辑表单**: 删除"排序权重" NInputNumber 字段. CardsSortModal 已
  通过 vuedraggable 提供拖拽排序, 数字输入是 10 年前的过时方案. editorForm.sort
  保留在 reactive 定义中维持 backend 协议向后兼容.

- **PC + Mobile HomeCard 扁平化** (CardItem.vue, Patch 1): padding 14px →
  10px 14px (上下省 8px), .card-item__title line-height 1.3 (默认 ~1.5).
  marquee 动画 translateX 横向不依赖, 完整保留 v0.2.12 Patch F 无缝循环.
  ≤480px @media padding override 同步统一.

- **PC 首页天气卡片**: 撤销 v0.2.9 的 1Hz 秒级时钟, 回到 1/min HH:MM 显示
  (Bevan: 秒变化制造视觉噪音). HomeHero tick 1000→60000ms; CityWidget Intl
  删除 second: '2-digit'. 未还原 minute-aligned setTimeout (保留 v0.2.9 简化).

## [0.2.12] - 2026-05-06

### Fixed

- **Mobile CardItem 2-row layout for compact card discrimination**:
  Card name and "仅内网" (internal-only) badge now stack vertically
  on mobile (≤768px) — name on row 1 (claiming full row width),
  badge on row 2 left-aligned. Previously names competed inline with
  badge causing aggressive truncation on small viewports. Bevan's
  real-device feedback: long names like "Open WebUI Server" were
  indistinguishable as "Ope..." across many cards. PC default
  (≥769px) keeps inline layout — wider cards have room for both.

- **Mobile CardItem long-name seamless marquee scroll**: Names that
  overflow the mobile container width now scroll horizontally in a
  seamless infinite loop. Implementation reuses v0.2.9 CityWidget's
  JS detection pattern (~80% logic reuse: ResizeObserver +
  scrollWidth detection + ref binding + cleanup) but upgrades two
  UX dimensions: (1) per-loop dwell phase (CSS keyframes 0%-25%
  static, 25%-100% scroll) lets users read the static name comfortably
  each cycle; (2) doubled inner content with nbsp separator and
  translateX(-50%) endpoint creates seamless visual loop without the
  jump-back artifact common to single-copy marquees. Total duration
  auto-scales: scroll_time = clamp(2s, distance × 0.04, 8s); total =
  scroll_time / 0.75 (the 25% dwell ratio). Speed remains constant
  distance × 0.04 px per second regardless of loop overhead.

- **PC CardItem long-name tooltip via title attribute**: Names that
  are CSS-truncated with ellipsis on PC now show the full string on
  ~500ms hover via the HTML title attribute. Lightweight (no NTooltip
  component overhead), browser-native rendering, zero runtime cost.

### Engineering Lessons Applied

- **NaiveUI :deep() override safety (memory feedback v0.2.10 chore)**:
  All marquee-related styling targets component-owned classes
  (.card-item__title, .card-item__title__inner) — no :deep() override
  of NaiveUI internal BEM. Avoids the v0.2.10 Task 2.13 wrap regression
  pattern from modifying internal box calculations.

- **Mobile total width pre-implementation audit (memory feedback
  v0.2.10 chore)**: Spec phase quantified single-card widths at
  480px (~151px) and 768px (~160-200px) viewport widths to validate
  2-row layout before implementation. Prevents v0.2.10 Task 2.13
  style wrap-as-alignment misdiagnosis.

- **v0.2.9 CityWidget marquee pattern reuse**: ResizeObserver +
  scrollWidth detection + watch + cleanup logic reused ~80%. Marquee
  is becoming a reusable UX primitive — v0.2.13 backlog includes
  upgrading v0.2.9 CityWidget itself to match v0.2.12's seamless
  + dwell standard (reverse direction reuse).

### Visual Gate Iteration

Six patch iterations during visual gate, all UX-driven (vs v0.2.10's
mix of UX and misdiagnosis-driven):
- Task 2: Initial implementation (template + JS + CSS)
- Patch A: inner span display: inline-block + white-space: nowrap
  (root cause for marquee not triggering — inline scrollWidth doesn't
  measure overflow content)
- Patch B: .card-item__title explicit white-space: nowrap (defensive)
- Patch C: onMounted setTimeout 100ms re-check (grid auto-fill timing)
- Patch D: animation-delay 1.5s initial (reverted — only first iteration)
- Patch E: keyframes 0%-25% dwell phase (each loop dwells)
- Patch F: doubled content + translateX(-50%) (seamless infinite loop)

No regressions, no misdiagnosis-driven reverts (only Patch D revert
was a UX upgrade, not a fix).

## [0.2.11] - 2026-05-06

### Fixed

- **Mobile Home page header — brand title visible at compact size**:
  "Moon Panel" text (.home-header__title) restored to mobile (≤768px)
  at font-size 12px (was hidden in v0.2.10 to free horizontal space).
  Combined with v0.2.11's button size unification, 4 elements now fit
  comfortably in 375px viewport with brand identity preserved.
  Quantified total width: ~346px < 375px (49px buffer, 17% safe margin).

- **Mobile Home page header — Network + Settings buttons unified to
  44x44 box style**: Both buttons now mirror admin/Layout.vue's
  hamburger + view-home buttons (44x44 + 1px solid var(--mp-card-border)
  + border-radius 8px + theme-aware hover via var(--mp-card-bg-hover)).
  Settings NButton tagged with .hh-icon-button component class,
  NetworkSwitcher's .network-switcher--narrow root class extended.
  Cross-page visual unity: all mobile top bar icon buttons now share
  the same box style across Home + Admin pages.

- **Mobile Home page header — SearchBox compact width**: NInput width
  reduced 160→100px in mobile (≤768px) and 120→80px in small mobile
  (≤480px), proportional ~62%/67% scaling. Combined with brand text
  restoration and button enlargement, total mobile top bar fits within
  viewport without NSpace wrap.

- **Mobile Settings icon vertical baseline (resolves v0.2.10 Known
  Issue)**: Settings circle button no longer offset ~5px below other
  elements. Resolved as side-effect of v0.2.11's button unification —
  all icon buttons now share 44x44 box dimensions, naturally aligning
  via NSpace's `align="center"` flex behavior.

### Engineering Lessons Applied

- **NaiveUI :deep() override safety**: All Task 2 button size changes
  use component root class CSS (`.hh-icon-button`,
  `.network-switcher--narrow`) instead of `:deep(.n-button__content)`
  overrides. Applies feedback_naiveui_deep_override.md (v0.2.10 chore
  memory) — avoids the v0.2.10 Task 2.13 wrap regression that occurred
  when modifying NaiveUI internal BEM box properties.

- **Mobile total width pre-implementation audit**: Spec phase
  quantified 346 < 375 (768) and 326 < 480 (480) before implementation.
  Applies feedback_mobile_layout_total_width_audit.md (v0.2.10 chore
  memory) — prevents the v0.2.10 Task 2.13 wrap-vs-alignment misdiagnosis.

## [0.2.10] - 2026-05-06

### Fixed

- **Mobile Home page header — admin button overlap eliminated**:
  "管理后台" text button replaced with Settings circle icon
  (lucide-vue-next), saving horizontal space on mobile (≤768px).
  Tooltip + aria-label preserve "管理后台" semantic.

- **Mobile Home page header — brand title hidden to fit controls**:
  "Moon Panel" text (.home-header__title) hidden in mobile (≤768px),
  keeping only the moon/sun logo (.home-header__logo) for brand
  identity. Frees ~100px horizontal space — combined with the admin
  icon shrink (above), all 4 top bar elements (logo + search +
  network + admin) now fit in the mobile 375px viewport without
  NSpace wrap. PC keeps full "Moon Panel" text (zero regression).

- **Mobile Admin page top bar — quick "view home" relocation**:
  "查看主页" moved out of hamburger dropdown back to the top bar,
  placed right of ☰ as a Home icon button mirroring hamburger style
  (44x44 + same border + theme-aware hover background) for visual
  consistency. Reverses v0.2.6 absorption: daily mobile use proved
  single-tap access > 2-tap (open dropdown + select).

- **Mobile Home page header — element vertical alignment**: NSpace
  items now inline-flex with center alignment in mobile (≤768px).
  Fixes NaiveUI NSpace v2's default n-space-item baseline alignment
  that misaligned rectangular NInput against circle buttons. Same
  pattern as v0.2.7 AuditLog NSpace fix.

- **Mobile Home page header — search box compact sizing**: NInput
  height 24px + font 12px in mobile (≤768px), reducing search box
  visual weight to match circle buttons. PC keeps default sizing
  (zero regression).

- **StatusBar breakpoint unification**: 720→768 to match v0.2.9
  HomeHero + v0.2.7 AuditLog. All layout breakpoints now 100%
  unified at 768/769 across the codebase (component-level 480 and
  NModal max-width values intentionally distinct).

### Known Issues

- **Mobile Home Settings icon visual baseline**: Settings circle
  button sits ~5px below SearchBox/NetworkSwitcher visual center on
  mobile. NaiveUI NButton circle's internal icon alignment requires
  deeper layout rework. Tracked for v0.2.11 mobile Home header
  redesign.

## [0.2.9] - 2026-05-06

### Fixed

- **Mobile/tablet (≤768px) weather card layout — 3 cards per row**:
  City weather cards now display 3 per row at flex 33.333% width,
  replacing the v0.2.x 2+1 split (two cards at 50% then third card
  stretching full row). Phone screens shouldn't be dominated by a
  temperature widget.

- **CityWidget template rebuild — 5-element single-row horizontal**:
  2-row layout (city/time on top, icon+temp/date on bottom) consolidated
  to single-row 5-element horizontal layout (city + emoji + temp + date
  + time). Information density significantly increased — same data in
  less than half the vertical space.

- **Mobile (≤768px) hides date+time — phone status bar already shows them**:
  Mobile cards display only city + emoji + temperature (3 elements).
  Date and time hidden via `display: none` since phone OS already shows
  them in the status bar — no need to duplicate inside widget.

- **PC desktop (≥769px) time displays seconds (HH:MM:SS)**: Time format
  extended from `HH:MM` to `HH:MM:SS` for real-time feel matching
  desktop expectations. HomeHero timer simplified from minute-aligned
  setTimeout+setInterval combo to a single 1Hz setInterval — net code
  reduction (~10 lines simpler) plus the seconds feature. Performance
  impact: ~1ms/sec total across all 5 widgets (negligible).

- **CityWidget typography normalization**: All visible elements share
  0.95rem font-size on PC with city name slightly bolder (weight 600).
  Replaces inverted/competing hierarchy where time (1.4rem) dominated
  visually and city name (0.9rem) was smallest.

- **CityWidget compact spacing**: Card padding tightened from 12px 16px
  to 8px 12px, enabling the single-row layout to render compactly.

- **Mobile (≤768px) font-size scaled down ~15%**: city/emoji/temp
  reduced to 0.8rem/0.9rem/0.8rem so 3-element layout fits comfortably
  in ~120px (375px viewport ÷ 3 cards). Fixes Los Angeles 21°C truncation.

- **City name marquee scroll for long names (mobile only)**: When city
  name content exceeds 60px container width (e.g., "阿姆斯特丹" 5 chars
  or "San Francisco" 13 chars), JS adds `--overflow` modifier and inner
  span scrolls left at adaptive speed (overflow distance × 0.04s,
  clamped 2-8s). Container-width-triggered detection via ResizeObserver
  is language-agnostic and naturally PC-disabled (no max-width on PC).
  PC hover pauses scroll; mobile has no hover so animation runs
  continuously.

- **HomeHero breakpoint unification**: Replaced split 720px/420px
  breakpoints with single 768px breakpoint matching v0.2.7 AuditLog,
  eliminating the 721-768px gray zone left by v0.2.8 PC rule.

## [0.2.8] - 2026-05-06

### Fixed

- **PC desktop (≥769px) weather card layout**: City weather cards
  (Xiamen / New York / Tokyo) now display at fixed 280px base width
  and center-aligned on desktop instead of stretching across the full
  1500px container with large empty middle areas. Uses `:deep(.cw)`
  in HomeHero scoped CSS to target CityWidget root via Vue scoped
  boundary (same pattern as v0.2.7 AuditLog NCard fix). Mobile
  (≤720px) and intermediate (721-768px gray zone) layouts unchanged.

### Chore

- Add `paths-ignore` to `.github/workflows/ci.yml` for `memory/**`,
  `**/*.md`, and `docs/**` paths. Markdown-only commits (memory chore,
  docs updates, CHANGELOG fixes) no longer trigger full backend +
  frontend CI build, saving ~3 min per such commit. Discovered during
  v0.2.7+chore(memory) ship cycle when memory chore commit triggered
  full CI despite being markdown-only.

## [0.2.7] - 2026-05-06

### Fixed

- **Mobile (≤768 px) admin audit log header**: NCard header now stacks
  vertically — title row separate from filter row, prevents "审计日志"
  word-break and "查询/重置" buttons overlapping the title. Filter
  controls now stretch to 50%-each width with wrap on narrow screens.
  Implementation reaches into NaiveUI's internal BEM via `:deep()`
  (`.n-card-header`, `__main`, `__extra`); NSpace v2 children are
  targeted with `> *` rather than `> .n-space-item` because Space.mjs
  renders children with `class: itemClass` (default undefined) and in
  useGap mode omits the wrapper div entirely — `> *` is the only
  selector that works in both modes. Desktop ≥769 px is byte-identical
  (rules live entirely inside the existing `@media (max-width: 768px)`
  block).

### Chore

- Add `*.tsbuildinfo` to `.gitignore`.
- Remove previously tracked `frontend/tsconfig.tsbuildinfo` and
  `frontend/tsconfig.node.tsbuildinfo` from the git index. Files remain
  on disk and continue to be regenerated by tsc on every type-check,
  but no longer pollute `git status`.

## [0.2.6] - 2026-05-05

Mobile polish round 2. Real-phone testing of v0.2.5 surfaced three admin
sub-pages that were still cramped or unusable below 769 px:
1) the right-side header actions (查看主页 + admin▾) wrapped under the
hamburger and consumed two rows on phones; 2) the audit-log
`NDataTable` overflowed horizontally with key fields clipped; 3) the
站点设置 cities row pushed the 移除 button off-screen because en + tz
+ coords occupied the full row width. v0.2.6 absorbs the header
actions into the hamburger menu, swaps the audit table for a card list
≤768 px, and re-lays the cities row as a 2-row CSS Grid. Plus a
theme-aware bottom-pad token so the risen StatusBar stops overlapping
the last NCard on mobile.

### Added

- `--mp-content-bottom-pad-mobile` CSS token in both `:root[data-theme]`
  blocks: moon = `24 px` (StatusBar hidden, normal breathing room),
  risen = `70 px` (clears the ~30-34 px StatusBar plus a 12-14 px
  margin so the last NCard isn't clipped). Consumed by the
  `@media (max-width: 768px)` rule on `.admin-content` in
  `Layout.vue` scoped style.

### Changed

- `views/admin/Layout.vue`: desktop right-side `NButton` ("查看主页")
  + `NDropdown` ("admin▾") are wrapped in
  `.admin-header__actions-desktop`, hidden via `display: none` below
  769 px. The hamburger dropdown gains a divider plus three new
  hand-rolled `<button>` rows — 查看主页 / 修改密码 / 退出登录 —
  styled to read pixel-equivalent to the existing nav-link rows.
  `NDropdown`-inside-mobile-menu would have teleported out of the
  click-outside scope, so the action items are flat buttons with
  inline `closeMobileMenu()` handlers. Desktop ≥769 px is unchanged.
- `views/admin/AuditLog.vue`: `NDataTable` is hidden ≤768 px and
  replaced by a vertical card list — five rows per entry (timestamp +
  status `NTag` / 友好动作名 + monospace key / 操作者 + IP /
  ellipsis-clipped UA / 查看详情 button). Card chrome uses
  `--mp-card-bg` + `--mp-card-border` so risen automatically inherits
  warm-brown surfaces. Pagination row stacks vertically on mobile.
- `views/admin/SiteSettings.vue`: cities list `.ws__city` swaps from a
  single-line flex to a `display: grid` 2-row layout below 769 px —
  `grid-template-areas` puts the drag handle and 移除 button as
  vertically-centred bookends spanning both rows, with 中文名 / 拼接的
  detail (en · tz · lat,lon) stacked between them. The desktop spans
  (`.ws__en`, `.ws__tz`, `.ws__coords`) hide on mobile; a new
  `.ws__detail` mobile-only span carries the merged ellipsis-clipped
  text. Desktop ≥769 px is byte-identical (NPopconfirm wrapped in a
  `.ws__remove` span as a stable grid-area target — flex-child width
  unchanged).

### Notes

- Desktop layouts (≥769 px) for all four touched files are pixel-equal
  to v0.2.5. Every visual change rides a `@media (max-width: 768px)`
  block.
- The bottom-pad token only applies on `<= 768px`; desktop continues
  to use the existing `padding: 2rem 1.5rem` rule, so the moon = risen
  desktop fingerprint is untouched.

## [0.2.5] - 2026-05-05

Mobile polish: the admin header title overflowed the 56 px header on
phones — a problem under risen's serif 1.55 rem in particular. Plus a
small loose-end on the weather-card loading pulse so it follows the
active theme's accent instead of flashing brand-blue under risen.

### Added

- `--mp-brand-font-size-mobile` CSS token in both `:root[data-theme]`
  blocks: moon = `1.0rem`, risen = `1.15rem`. Maintains the moon < risen
  visual hierarchy on phones (15% gap) while bringing both inside the
  56 px header. Picked up by a new `@media (max-width: 768px)` rule on
  `.admin-header__title` in `Layout.vue` scoped style.

### Changed

- `CityWidget.vue` weather-card loading-pulse gradient stops swapped
  from hardcoded `rgba(91, 141, 239, 0.6)` to
  `color-mix(in srgb, var(--mp-brand-accent) 60%, transparent)`. Moon
  resolves to the same brand-blue as before; risen resolves to warm
  golden, so the loading pulse now matches whichever theme is active.

## [0.2.4] - 2026-05-05

Mobile usability hotfix. v0.2.3 left `/admin` unreachable below the
default desktop overview on phones — the horizontal NMenu's
`responsive` prop collapsed the entire 6-item bar at narrow widths
and offered no replacement, so phone users could not switch to
分组 / 卡片 / 站点设置 / 审计日志 / 安全管理. v0.2.4 swaps that menu
for a hand-rolled desktop nav at >=769px and a hamburger-driven
dropdown at <=768px.

### Fixed

- `views/admin/Layout.vue`: dropped `NMenu mode="horizontal" responsive`
  in favour of a `<nav class="admin-header__menu admin-nav-desktop">`
  containing direct `RouterLink`s. The `.admin-header__menu` class is
  preserved so the existing `phase-3d-2` e2e selector still anchors.
  Below 768px the desktop nav hides and a 44×44 hamburger button (using
  `lucide-vue-next`'s `Menu` icon) reveals a fixed-position dropdown
  pinned under the 56px header. Each menu row is ≥44px tall for thumb
  targets, the current route highlights via `--mp-brand-primary`, and
  the menu auto-closes on (a) selecting an item, (b) route change,
  (c) a click anywhere outside the menu/button, and (d) crossing the
  769px breakpoint upward (rotating tablet case).
- All hamburger / mobile-menu colours bind to `--mp-*` tokens, so moon
  reads cool-blue and risen reads warm golden without a second pass.

### Notes

- Pixel-equivalent on PC (>=769px): the new desktop nav uses the same
  flex/gap/height rhythm as the v0.2.3 horizontal NMenu, so a desktop
  user sees no diff.
- Other mobile-responsiveness gaps (天气卡布局, 色调微调, loading-bar)
  are intentionally out of scope and will land in v0.2.5.

## [0.2.3] - 2026-05-05

Risen-theme follow-up: 5 new starry / luxurious wallpapers tuned for the
warm golden palette, plus a UX touch where switching theme auto-swaps
the wallpaper to the matching default — but only for builtin wallpapers
(custom uploads are preserved).

### Added

- 5 new builtin SVG wallpapers (~10 KB total):
  `starlit_dunes` (night sky over a warm dune ridge with low moon halo),
  `velvet_galaxy` (deep-purple radial space with a golden core and
  spiral arms), `golden_aurora` (dark-brown sky with two soft golden
  aurora bands, blur baked once via `feGaussianBlur` in the SVG), 
  `sunset_haze` (wine-red → gold horizon with a setting-sun disk),
  `obsidian_stars` (near-black sky with a dense starfield + a few
  golden glow-stars). Total builtin count: 14.
- Theme → recommended wallpaper auto-swap. When the user changes the
  theme preset (`moon` ↔ `risen`) and they're currently on a builtin
  wallpaper, the wallpaper switches to the theme's recommended one in
  the same `/admin/settings` PUT so the panel reads visually coherent
  right away. moon → `builtin:night`, risen → `builtin:starlit_dunes`.
  Custom uploads are intentionally NOT overridden — that's a personal
  choice we don't second-guess.

### Notes

- v0.2.2 `CityWidget.vue` color theming was reported as missing but
  was actually already in place from v0.2.2. Verified 6/6 colours
  (`bg`, `border`, name, time, temp, date) are bound to
  `--mp-weather-*` tokens; no new edit needed.

## [0.2.2] - 2026-05-05

Completes the theme system started in v0.2.1. v0.2.1 shipped just brand
typography + status-bar visibility under `data-theme`; the rest of the
panel (cards, weather, group containers, badges) was still hardcoded
moon-blue. v0.2.2 routes those colors through the same token layer so a
risen-theme deployment actually looks risen end-to-end, while moon-theme
deployments remain pixel-equivalent to v0.2.0/v0.2.1.

### Changed

- Expanded `:root[data-theme]` token set in `main.css` to cover the full
  visible surface: card bg / border / hover / ring / text, weather card
  bg / border / time / temp / date, group container bg / border / title /
  divider / icon, internal/external badges, and global text levels.
- `CardItem.vue`: `--mp-card-bg` / `-bg-hover` / `-border` / `-hover-ring`
  / `-text` and `--mp-badge-bg` / `-text` plus `--mp-text-tertiary` for
  the description line. Moon resolves to the literal v0.2.0 rgba strings;
  risen swaps to warm-brown bg + golden 1px border + golden hover ring +
  cream text + golden badges.
- `CityWidget.vue`: weather card bg / border / city name / time / temp /
  date all bind to `--mp-weather-*` tokens.
- `Home.vue`: header logo glyph, brand title color, and the home-group
  container (bg / border / title / divider / icon) bind to tokens. Header
  logo glyph picks up `--mp-brand-accent` so the moon-icon SVG matches
  the active theme's accent (blue for moon, golden for risen).
- `admin/Layout.vue`: brand title color binds to `--mp-brand-primary`
  (replaces the v0.2.1 transient `--mp-brand-color` that's been removed).

### Notes

- NaiveUI components (NCard / NInput / NDataTable / NDropdown / NSelect
  etc.) retain their built-in dark theme styling — fully re-theming
  NaiveUI internals would require dynamic `themeOverrides` passes through
  `NConfigProvider`, which is its own follow-up. The user-visible
  Moon-Panel-authored chrome is fully themed; surrounding NaiveUI form
  controls stay neutral and look correct under both presets.
- Moon theme tokens were chosen to literally equal the v0.2.0 hardcoded
  rgba strings (e.g. `--mp-card-text` = `rgba(255,255,255,0.92)` matches
  the previous `.card-item__title` color), so a moon-theme user post-
  v0.2.2 sees zero visual drift from v0.2.0.

## [0.2.1] - 2026-05-05

Theme system: the panel now ships two visual presets, with the v0.2.0
default ("moon") preserved exactly and a new opt-in alternative
("risen", warm serif) selectable from admin → Site Settings.

### Added

- `site.theme_preset` setting ("moon" | "risen") returned from
  `/api/public/panel`. Stored in the existing `Setting` key/value table
  — no migration. Backend falls back to "moon" when unset or invalid.
- New CSS variable layer in `main.css` keyed by `:root[data-theme="..."]`.
  All theme-aware styles read `--mp-brand-font`, `--mp-brand-font-size`,
  `--mp-brand-color`, `--mp-status-bar-display` etc. through the
  cascade. The "moon" ruleset reproduces v0.2.0's hardcoded values
  exactly, so default-theme users see no visual change at all.
- `App.vue` watches `ui.themePreset`, sets `<html data-theme="...">`,
  and lazy-loads serif fonts only when the user actually flips to
  "risen" — `await import('@fontsource/playfair-display/700.css')`
  inside the conditional means moon-theme users never make a font
  network request. Failure to load fonts (offline, CDN miss) falls
  back silently to the system serif stack via the multi-family
  `--mp-brand-font` value.
- `StatusBar.vue` (mounted on home + admin) renders a small fixed
  bottom bar with version / cards / groups / uptime. Visibility is
  CSS-gated by `--mp-status-bar-display` (moon=`none`, risen=`flex`)
  so the component is mounted on both themes but only paints under
  risen. Polls `/api/site/stats` once on mount and every 5 minutes.
- `GET /api/site/stats` (public, no auth) — lightweight stats
  endpoint feeding the status bar. Distinct from
  `/api/admin/stats` (auth-gated, 4 fields incl. audit count).
- Admin `ThemePicker.vue` component embedded in the "Site Information"
  card on `/admin/site-settings`. Two thumbnail tiles
  (Moon / Risen) with one-click apply (no save button — same UX as
  the wallpaper picker). Thumbnails come from
  `/assets/themes/{moon,risen}-preview.svg`, embedded into the binary
  via `assets.ThemeFS()` + a new `r.GET("/assets/themes/:name", ...)`
  handler that mirrors the wallpaper-preview pattern.

### Changed

- `WallpaperPicker.vue` info alert now reads `内置 {N} 张` from
  `ui.builtins.length` instead of the hardcoded `3`. The blur-slider
  hint sentence (defunct since v0.1.7) is removed.
- New runtime npm dependencies (lazy-loaded only): `@fontsource/playfair-display`
  (Latin serif), `@fontsource/noto-serif-sc` (Chinese serif). Bundle
  sizes only land when "risen" is selected.

## [0.2.0] - 2026-05-05

First feature release after the v0.1.x stabilization line. v0.1.x ended at
`v0.1.7` once the wallpaper-layer paint cascade was rooted out; v0.2.0
turns to user-facing additions: a customizable site title, four new
builtin wallpapers, a more compact weather hero, and drag-to-reorder for
the hero's city list. The defunct blur slider (UI of a feature that
v0.1.7 removed) is hidden.

### Added

- `site.title` setting and an admin input for it. Empty value falls back
  to "Moon Panel". The browser tab title and both home / admin headers
  read live from the new `ui.siteTitle` store slot, so a homelab can
  rename the panel to anything (e.g. "Foo Family Hub") without code
  changes. Persisted via the existing `/api/admin/settings` endpoint —
  no schema migration.
- 6 new builtin SVG wallpapers shipping inside the binary:
  `galaxy` (deep-space galactic core + scattered stars), `ocean`
  (sunrise over water), `sunset` (warm sky with clouds and silhouette),
  `mountain` (cold dawn over snow-capped peaks), `meadow` (rolling
  hills under pale sky with layered grass undulations), `forest`
  (warm-dusk sky with three-layer evergreen-tree silhouette). Each
  is hand-tuned pure SVG (~1-3 KB), gradient-based, no
  `<feGaussianBlur>` filters (lessons from v0.1.7) — they composite
  cheap and stay sharp at any resolution. The original `night` /
  `aurora` / `graphite` set is kept as-is. Total builtin count: 9.
- Drag-to-reorder for the hero city list in
  `admin/site-settings`. Cities are stored as a JSON array in a single
  `widget.cities` setting row, so the reorder is a pure-frontend
  shuffle persisted by the existing save path — no API change.
  `vuedraggable` was already a dependency (used by the groups / cards
  sort modals).

### Changed

- Hero `CityWidget` re-laid out from a 4-row stack (~190 px tall) to a
  2-row compact layout (~90 px tall): top row pairs city name with
  current time, bottom row pairs weather emoji + temperature with the
  date. Hero now lets the card grid be the page's visual focus instead
  of dominating the fold. Acrylic surface (`.mp-acrylic-light`) and
  loading-pulse bar are unchanged.
- Hidden the wallpaper blur slider in admin settings. v0.1.7 removed
  the CSS filter on the wallpaper layer (the root cause of the global
  continuous-repaint regression), so the slider had no visible effect
  anymore. The `ui.wallpaper_blur` setting + backend column are kept
  for schema stability; a future release can decide whether to bake
  blur into the uploaded wallpaper at canvas-compress time.

## [0.1.7] - 2026-05-05

Final card-perf fix in the v0.1.x line, and the most surprising one.
v0.1.4-v0.1.6 hunted card-side paint causes (NDropdown, drop shadow,
group backdrop-filter) — each helped, but Bevan's home + admin pages
still felt laggy from page load. v0.1.7 root-caused it to a different
layer entirely: the wallpaper itself.

### Changed

- Dropped the `filter: blur(${ui.blur}px)` binding on `.wallpaper-layer`
  (and the companion `transform: translateZ(0) scale(1.05)` that
  compensated for blur-edge bleed). The original design ran the loaded
  wallpaper image through a 9 px Gaussian blur on every frame; with a
  4K background that was continuous GPU work behind every interaction,
  cascading to feel like jank everywhere — including admin pages that
  share the same fixed wallpaper layer. Console-disabling the filter
  alone instantly returned 60 fps in Bevan's diagnostic.
- The `ui.wallpaper_blur` setting (admin slider 0-20 px) is still
  persisted in the database; it just has no visual effect for now. A
  follow-up release can decide whether to bake the blur into the
  uploaded wallpaper at canvas-compress time, or remove the slider.
- Visual: built-in wallpaper detail (e.g. `night.svg` starfield, aurora
  gradient bands) now renders sharp instead of soft-blurred. Cards
  retain their own translucent fills for legibility.

## [0.1.6] - 2026-05-05

Final card-hover-perf hotfix in the v0.1.x line. Bevan's Paint Flashing
trace showed all five home-page cards flashing green *as one region*
when the cursor moved across the grid — not five independent flashes,
one. The cause: `.home-group`'s `backdrop-filter: blur(6px)` made the
entire group a single composite region, so any child card's hover
transition forced the whole group to re-composite. v0.1.5's
`contain: layout paint style` on the card prevents paint from escaping
the card box, but doesn't stop the parent from re-compositing when
child output changes — that's a different mechanism.

### Changed

- Dropped `backdrop-filter` from `.home-group` in `Home.vue`. The 5b-4
  decision to keep it was based on a "small blur is cheap" assumption
  that this version's evidence overturned. Cards now paint as
  independent regions; group loses the frosted-glass effect but keeps
  the `rgba(255,255,255,0.025)` translucent fill and 1 px border for
  visual grouping over the wallpaper.

## [0.1.5] - 2026-05-03

Continuation of the card hover-perf hunt. v0.1.4 (NDropdown lazy mount)
helped, but Bevan's F12 Performance still showed Frames mostly-red on
mouse-over. Re-diagnosed and root-caused: the v0.1.5 fix is unrelated to
the v0.1.4 NDropdown work — both were genuine but distinct issues.

### Changed

- `CardItem.vue` hover: dropped the outer drop-glow line of the v5b-3
  hover `box-shadow`. The drop glow used a 20 px blur radius + 6 px
  y-offset, so its painted region extended ~26 px past the card box.
  That overshoot landed inside the parent `.home-group`'s
  `backdrop-filter: blur(6px)` region, and the browser had to re-sample
  the home-group's backdrop on every animated hover frame. With 5 cards
  transitioning in/out as the cursor moved across the grid, paint
  became the bottleneck. Hover signal still has the inner 1 px brand-blue
  ring, the `translateY(-1px)` lift, and the background-color brighten
  — only the soft glow is gone.
- `CardItem.vue` base style adds `contain: layout paint style`. Future
  shadow / overlay additions stay clipped inside the card box and can't
  silently regress this fix by re-triggering parent backdrop sampling.

## [0.1.4] - 2026-05-03

Pure perf hotfix: card hover/click lag root-caused to NDropdown over-eager
mounting, not the wallpaper / acrylic stack we suspected through 5b-3/5b-4.

### Changed

- `CardItem.vue` lazy-mounts its NDropdown (`v-if="dropdownMounted"`).
  Pre-v0.1.4 every card kept a fully-instantiated NDropdown alive with
  `:show="false"` — NaiveUI spins up VBinder + popper.js listeners +
  ResizeObserver per instance regardless of show state, so 5 cards on
  the home page = 5 popper machines doing nothing on every paint and
  every layout recalc. First right-click on a card now flips the v-if;
  subsequent right-clicks toggle `:show` only. Cards that never get
  right-clicked never pay the popper cost.
- Verified hover state remains 100% CSS-driven (no `@mouseenter` /
  `@mouseleave` / JS hover refs in `CardItem.vue`). The earlier 5b-3
  hover transition rework is still in effect; this release is purely
  about the unrelated NDropdown overhead.

## [0.1.3] - 2026-05-03

Hotfix release for two functional bugs surfaced testing v0.1.2 in
production, plus a long-standing placeholder filled in.

### Fixed

- Version badge no longer renders as `vundefined`. The frontend
  `getVersion()` helper in v0.1.2 typed the axios call as `<VersionInfo>`
  and read `.data` as if it were the inner payload — but the backend
  wraps every response in `{code, msg, data}`, and the http client (plain
  axios) doesn't unwrap. Result: `version.value.version` was reading the
  envelope's nonexistent `version` field, returning `undefined`. The
  LDFLAGS injection itself was working correctly the whole time. Fixed
  to follow the same `data.data!` pattern as `panel.ts`.
- Version-badge popover preview now skips heading / hr / code-fence /
  link-reference lines and takes the first genuine prose paragraph
  (joined up to 2 lines, truncated at 120 chars). v0.1.2's preview
  returned the heading line stripped of `##`, which read as a redundant
  date repeat (e.g. `[0.1.2] - 2026-05-02`).

### Added

- `GET /api/admin/stats` (auth-gated) returning `groups_count`,
  `cards_count`, `engines_count`, and the count of audit-log entries
  written in the last 7 days. Drives the admin Overview page —
  previously hardcoded to `0`.
- Admin Overview now displays the four real counters instead of the
  three zero-filled placeholders. The "本页未来会显示..." NAlert is
  removed; the page is now an actual overview.

## [0.1.2] - 2026-05-02

Adds an in-app version indicator so deployments can see at a glance whether
they're behind upstream, plus the build-time wiring (LDFLAGS) to embed real
version metadata into the released binary.

### Added

- Version badge in the bottom-left corner of the home page and admin
  layout. Click to open a popover showing the running version + build
  date + short commit, the most recent 3 GitHub releases (tag, date,
  one-line preview), and a "View all on GitHub" link.
- `GET /api/version` (public, no auth) returning the binary's
  `{ version, build_date, commit }`. Frontend reads this on every page
  load; backend value is set at `go build` time via `-ldflags -X` so it
  reflects the actual published image, not a hardcoded constant.
- Recent releases are pulled from the public
  `api.github.com/repos/.../releases` endpoint (no auth, 60 req/h is
  plenty for click-to-open) and cached in `localStorage` for 30 minutes
  per session. Network / 429 errors fall back to stale cache or hide the
  releases section gracefully — the current-version display always
  works.

### Changed

- `Dockerfile`: backend build stage now accepts `VERSION`, `BUILD_DATE`,
  `COMMIT` build args and feeds them into `-ldflags -X` overrides for
  `internal/api.{Version,BuildDate,Commit}`. Local `docker build` without
  the args keeps the dev defaults; release builds get real values.
- `.github/workflows/release.yml`: passes the tag-derived version,
  workflow start time, and full commit SHA into the build via
  `--build-arg`. No retag required to refresh metadata — the next tag
  push fills in fresh values.

## [0.1.1] - 2026-05-02

Hotfix release covering two functional bugs surfaced in v0.1.0 production
deployment, an entrypoint robustness pass, and a substantial deployment
documentation expansion.

### Fixed

- Icon autocomplete now correctly commits the selected option's value
  (`lucide:<name>` for Lucide icons, full CDN URL for dashboard-icons)
  instead of the bare name. NAutoComplete's default behavior writes the
  option's `label` into the v-model on selection — that's correct for
  picker UIs where label === value, but our options use display name as
  label and prefixed/qualified strings as value. The fix overrides the
  v-model in `nextTick()` after select so the prefix isn't lost; saves
  no longer trip the icon-format validation warning.
- Default builtin wallpapers now appear in the admin Site Settings
  picker on private-mode (`MOON_PUBLIC_MODE=false`) deployments. The
  initial `ui.ensureLoaded()` call hits a 401 before login (the panel
  endpoint requires auth in private mode) and silently bails, leaving
  `ui.builtins` as `[]`. App.vue now watches `auth.authenticated` and
  re-fetches the panel on the false→true transition — covers regular
  login, TOTP verification, and first-time admin init in a single watch.

### Changed

- `docker/entrypoint.sh`: chown the data directory by numeric
  `$PUID:$PGID` rather than via resolved user/group names, so a
  corrupted `/etc/passwd` or a silent `addgroup`/`adduser` failure
  doesn't leave files root-owned. Added an explicit startup log line
  (`[entrypoint] chown'd /data to ...`) and fail-fast on chown error
  (a clear FATAL message instead of crash-looping with an opaque
  permission-denied later in the stack).

### Added

- README: full Deployment / Updating / Common Issues sections in both
  English and Chinese. Walks through PUID/PGID determination per
  platform (Linux / Synology / Unraid / TrueNAS), step-by-step compose
  setup, expected log sequence, the `:0.1` minor-track upgrade pattern,
  and troubleshooting for the most common failure modes (permission
  denied on jwt.key, bind-mount path missing, restart loops).

## [0.1.0] - 2026-05-02

Initial public release. Self-hosted dashboard / start page with single-password
admin, intranet ↔ extranet URL switching, and frosted-glass UI — built as a
lightweight Sun-Panel alternative for personal use on NAS / VPS.

Single binary, embedded frontend, pure-Go SQLite (no CGO). Runs on a Raspberry
Pi or Synology with the same image.

### Added

#### Core
- Cards organized in groups; per-card title, description, icon, internal URL,
  external URL, and tags
- One-click intranet ↔ extranet switch on the home page (NetworkSwitcher) — the
  same card resolves to your LAN IP at home and your public domain on mobile data
- Public mode: home page accessible without login (default); private mode hides
  it behind auth
- Search bar on home page filters across groups, cards, descriptions, and URLs
- Configurable search engines (Google / Bing / DuckDuckGo / 百度 seeded; admin
  can add / remove / reorder / pick default)

#### UI / Customization
- Naive UI dark theme with primary-color override (5 presets + HSL color picker)
- Wallpaper system: 3 builtin SVG gradients (night / aurora / graphite) embedded
  in the binary, plus user uploads (auto-compressed client-side to 1920×1080
  WebP). Per-wallpaper backdrop blur 0–20 px
- Acrylic frosted-glass surfaces (Win11 / macOS Big Sur style) on cards, modals,
  and login — gated behind `body.has-wallpaper` so default-dark theme is
  unchanged when no wallpaper is set
- Stateful 4-state input fields (idle → opened → editing → modified) across all
  admin editors — click-to-clear semantics with revert affordance
- Lucide icon library + dashboard-icons catalog with autocomplete picker;
  per-card icon supports URL / `upload:hash` / `lucide:name` / fetched-from-URL
- Drag-and-drop reordering for groups, cards, and search engines (vuedraggable)
- HomeHero: configurable city list (up to 5) showing local time + weather
  emoji; °C / °F toggle; Open-Meteo backend, no API key required
- Mobile-responsive layout with long-press to open card target picker

#### Auth & Security
- Single-password admin login (no email, no signup, no SSO — by design for
  personal-panel scope)
- Bcrypt password hashing with 8-character minimum enforced at every
  bootstrap / change path (defense-in-depth)
- TOTP 2FA enrollment with QR code + 8 single-use backup codes; separate TOTP
  rate-limit independent of password lockout
- IP-based login lockout (5 password attempts / 15 min → 30 min lock, 7 TOTP
  attempts / 10 min → 15 min lock); CIDR allowlist for trusted networks (home
  LAN, fixed office IP) bypasses lockout but still appears in audit log
- Audit log of admin mutations (login / logout / 2FA / password / cards /
  groups / settings / backups), with recursive secret redaction; 90-day
  retention with opportunistic cleanup
- Session invalidation floor (`auth.session_floor`) — stamping a global
  cutoff revokes all in-flight cookies without restarting the container
- "Remember me" sessions (7 d default / 30 d remembered) via httpOnly cookie
- SSRF defense for icon-fetch endpoint: block private IP ranges, schema
  whitelist, optional host allowlist
- ZIP path-traversal guard + 50 MiB size cap on backup-restore upload

#### Backup & Restore
- JSON export of all groups / cards / search engines / settings (excluding
  password hash, TOTP secret, session floor, audit log)
- ZIP export bundles `uploads/` (icons + wallpapers) alongside metadata.json
  for full-restore portability
- Restore replaces existing content atomically in one transaction; preserves
  user / 2FA state on the new instance; auto-fallback for orphaned wallpaper
  references when the backup target file is missing

#### Deployment
- Single static Go binary with frontend embedded via `go:embed`
- Pure-Go SQLite (`modernc.org/sqlite`) — no CGO, cross-compiles in seconds
- Multi-architecture Docker images: `linux/amd64` + `linux/arm64`
- LinuxServer.io-style PUID/PGID env vars for NAS deployments (Synology,
  Unraid, TrueNAS) — data files end up owned by the host user
- Configurable Alpine mirror (`ALPINE_MIRROR` build arg) for CN builders;
  defaults to official `dl-cdn.alpinelinux.org`
- One-volume design: everything under `/data` (SQLite db + uploads/ + jwt.key)

#### Developer Experience
- Local dev workflow with hot reload (Go [air](https://github.com/air-verse/air)
  + Vite HMR), no Docker rebuild loop required for daily iteration
- One-line PowerShell launchers (`backend/dev.ps1`, `frontend/dev.ps1`) for
  Windows developers; auto-detects `go` / `air` install location
- Comprehensive [DEV.md](docs/DEV.md) covering env setup, port conventions,
  data migration via the backup feature, PowerShell ↔ Bash syntax, and
  common pitfalls (shadow configs, NaiveUI cssr classes, etc.)
- Dev / prod data isolation: dev uses `./data-dev` and port 3001, leaves
  production `./data` and port 3000 untouched

[Unreleased]: https://github.com/bevanho777-max/moon-panel/compare/v0.2.30...HEAD
[0.2.30]: https://github.com/bevanho777-max/moon-panel/compare/v0.2.29...v0.2.30
[0.2.29]: https://github.com/bevanho777-max/moon-panel/compare/v0.2.28...v0.2.29
[0.2.5]: https://github.com/bevanho777-max/moon-panel/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/bevanho777-max/moon-panel/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/bevanho777-max/moon-panel/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/bevanho777-max/moon-panel/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/bevanho777-max/moon-panel/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.7...v0.2.0
[0.1.7]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.6...v0.1.7
[0.1.6]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/bevanho777-max/moon-panel/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/bevanho777-max/moon-panel/releases/tag/v0.1.0
