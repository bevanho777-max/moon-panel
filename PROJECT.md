# Moon Panel

自托管个人导航面板，Sun-Panel 的轻量替代品。

## 设计目标

- **零摩擦访问**：默认公开模式，主页无需登录
- **真单文件部署**：Go 二进制内嵌前端资源，单进程 + SQLite，NAS 友好
- **简单认证**：单密码（无邮箱注册），密码存 bcrypt
- **网络感知**：每张卡片支持内/外网双地址，前端一键切换
- **数据可携带**：JSON 一键导入导出，迁移无痛

## 技术栈

### 后端
- **Go 1.22+** + **Gin**（路由）
- **modernc.org/sqlite**（纯 Go 实现，**无需 CGO**，跨平台编译/Docker 镜像更简单）
- **GORM** 或 **sqlx**：建议 GORM（migration 自动化，CRUD 模板少）
- **golang-jwt/jwt** + httpOnly Cookie（不放 localStorage，防 XSS 拿 token）
- **embed**：把 `frontend/dist` 编进二进制，单文件即服务

### 前端
- **Vue 3** + **Vite** + **TypeScript**
- **Pinia** 状态管理 / **Vue Router** 路由
- **Naive UI**（Vue 3 原生，体积小，admin 页面够用）
- **vue-draggable-plus**（拖拽排序）
- **UnoCSS** 或 **Tailwind**（首页样式，二选一，建议 UnoCSS 更轻）

### 部署
- 多阶段 Dockerfile：`node:alpine` 构建前端 → `golang:alpine` 编译后端 → `gcr.io/distroless/static` 运行（最终镜像 ~15MB）
- docker-compose 一键起，挂载 `./data` 即可

### 与原方案的差异
| 你的方案 | 调整建议 | 原因 |
|---|---|---|
| `mattn/go-sqlite3` | `modernc.org/sqlite` | 纯 Go，免 CGO，免 gcc/musl 折腾 |
| token 默认 localStorage | httpOnly Cookie | 更安全，且 Admin 页面无需 SPA 跨域 |
| - | 前端嵌入二进制 | "单文件部署"才名副其实 |

---

## 项目结构

```
moon-panel/
├── backend/
│   ├── cmd/server/main.go          # 入口：装载 embed 资源、启 Gin
│   ├── internal/
│   │   ├── api/                    # Gin handler（按资源拆文件）
│   │   │   ├── public.go           # 公开接口
│   │   │   ├── auth.go             # 登录/改密
│   │   │   ├── group.go
│   │   │   ├── card.go
│   │   │   ├── search.go
│   │   │   ├── setting.go
│   │   │   ├── upload.go
│   │   │   └── backup.go
│   │   ├── middleware/             # auth、CORS、日志
│   │   ├── model/                  # GORM 模型 + 迁移
│   │   ├── service/                # 业务逻辑（薄）
│   │   ├── store/                  # DB 句柄、初始化
│   │   ├── auth/                   # JWT 签发/校验、bcrypt
│   │   └── config/                 # 启动参数、env
│   ├── web/embed.go                # //go:embed dist/*
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── api/                    # axios 封装 + 各资源 client
│   │   ├── components/
│   │   │   ├── home/               # 首页：CardGrid、SearchBar、UrlSwitcher
│   │   │   └── admin/              # 后台：GroupEditor、CardEditor、IconPicker
│   │   ├── views/
│   │   │   ├── Home.vue            # 公开首页
│   │   │   ├── Login.vue
│   │   │   └── admin/              # 嵌套路由：dashboard、cards、search、settings
│   │   ├── stores/                 # panel.ts、auth.ts、network.ts
│   │   ├── router/index.ts
│   │   ├── styles/
│   │   ├── App.vue
│   │   └── main.ts
│   ├── public/
│   ├── index.html
│   ├── vite.config.ts              # build outDir = ../backend/web/dist
│   ├── tsconfig.json
│   └── package.json
├── data/                           # 运行时数据（Docker 挂载点，git ignore）
│   ├── moon.db
│   └── uploads/
├── Dockerfile
├── docker-compose.yml
├── Makefile                        # make dev / make build / make docker
├── .gitignore
├── README.md
└── PROJECT.md
```

---

## 数据模型

SQLite 单库，所有时间戳 `DATETIME`（UTC）。下面用伪 SQL 表达，实际由 GORM AutoMigrate 生成。

### `users` —— 单密码模式下只有一行
```
id            INTEGER PK
username      TEXT UNIQUE   -- 默认 'admin'
password_hash TEXT          -- bcrypt
created_at    DATETIME
updated_at    DATETIME
```

### `groups` —— 卡片分组
```
id          INTEGER PK
name        TEXT NOT NULL
icon        TEXT             -- 可选，分组小图标
sort        INTEGER          -- 排序权重，越小越前
created_at  DATETIME
updated_at  DATETIME
```

### `cards` —— 导航卡片（核心表）
```
id              INTEGER PK
group_id        INTEGER FK → groups(id) ON DELETE CASCADE
title           TEXT NOT NULL
description     TEXT
icon            TEXT             -- 在线 URL 或 'upload:<filename>'
icon_type       TEXT             -- 'url' | 'upload' | 'preset'
url_internal    TEXT             -- 内网地址，可空
url_external    TEXT             -- 外网地址，可空
url_default     TEXT             -- 'internal' | 'external'，新会话默认走哪个
open_in_new_tab BOOLEAN DEFAULT 1
sort            INTEGER
created_at      DATETIME
updated_at      DATETIME
```
约束：`url_internal` 和 `url_external` 至少有一个非空（应用层校验）。

### `search_engines` —— 搜索引擎
```
id            INTEGER PK
name          TEXT             -- 'Google'
url_template  TEXT             -- 'https://www.google.com/search?q={query}'
icon          TEXT
is_default    BOOLEAN          -- 仅一行为 true（应用层保证）
sort          INTEGER
```

### `settings` —— 站点级 K/V 配置
```
key    TEXT PK
value  TEXT             -- JSON 编码，前端按 schema 解析
```
内置 key：
- `site.title` —— 浏览器标题
- `site.greeting` —— 首页问候语模板（支持 `{time_of_day}`）
- `site.background` —— 背景图（URL 或 upload 引用）
- `site.public_mode` —— `true` = 主页公开 / `false` = 整站需登录
- `site.default_network` —— 默认内/外网（前端首次访问 fallback）
- `theme.mode` —— `light` | `dark` | `auto`

### 前端会话级状态（不入库）
- 当前选定网络（internal/external）：存 `localStorage`，前端控制
- 当前搜索引擎：存 `localStorage`

---

## API 设计

约定：
- 所有 JSON，请求体 `application/json`
- 统一响应 `{ code: 0, data: ..., msg: "" }`，非 0 即业务错
- 鉴权用 httpOnly Cookie（`moon_session`），管理路由 401 时前端跳 `/login`
- 上传用 `multipart/form-data`

### Public（无需登录，受 `site.public_mode` 控制）
| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/public/panel` | 返回首页全量数据：groups + cards + search_engines + 公开 settings |
| GET | `/uploads/:filename` | 上传图标静态资源（直接由 Gin Static 服务） |

### Auth
| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/auth/login` | `{username, password}` → 设置 Cookie |
| POST | `/api/auth/logout` | 清 Cookie |
| GET | `/api/auth/me` | 当前用户信息（含是否已初始化） |
| POST | `/api/auth/init` | 首次启动设置 admin 密码（仅当 users 表为空时可用） |
| PUT | `/api/auth/password` | 改密码（需鉴权） |

### Admin（全部需鉴权）
**Groups**
- `GET /api/admin/groups`
- `POST /api/admin/groups`
- `PUT /api/admin/groups/:id`
- `DELETE /api/admin/groups/:id`
- `PUT /api/admin/groups/sort` —— body: `[{id, sort}, ...]`

**Cards**
- `GET /api/admin/cards?group_id=`
- `POST /api/admin/cards`
- `PUT /api/admin/cards/:id`
- `DELETE /api/admin/cards/:id`
- `PUT /api/admin/cards/sort`

**Search Engines**
- `GET /api/admin/search-engines`
- `POST /api/admin/search-engines`
- `PUT /api/admin/search-engines/:id`
- `DELETE /api/admin/search-engines/:id`
- `PUT /api/admin/search-engines/sort`

**Settings**
- `GET /api/admin/settings` —— 全部 K/V
- `PUT /api/admin/settings` —— body: `{key: value, ...}`，批量

**Upload**
- `POST /api/admin/upload/icon` —— multipart，返回 `{filename, url}`
- `DELETE /api/admin/upload/:filename` —— 清理无引用图标

**Backup**
- `GET /api/admin/backup/export` —— 下载 `moon-panel-backup-{ts}.json`，包含全部表（不含 password_hash、不含上传文件）
- `POST /api/admin/backup/import` —— 上传 JSON，body: `{mode: "merge"|"replace", data: {...}}`
- `GET /api/admin/backup/export-full` —— 含 uploads 的 zip（可选，第二阶段做）

### 公开返回 payload 示例
```json
GET /api/public/panel
{
  "code": 0,
  "data": {
    "site": {
      "title": "我的导航",
      "greeting": "晚上好",
      "background": "/uploads/bg.jpg",
      "default_network": "internal"
    },
    "groups": [
      {
        "id": 1, "name": "影音", "icon": null, "sort": 0,
        "cards": [
          {
            "id": 10, "title": "Jellyfin",
            "icon": "/uploads/jellyfin.png", "icon_type": "upload",
            "url_internal": "http://192.168.1.10:8096",
            "url_external": "https://media.example.com",
            "url_default": "internal",
            "open_in_new_tab": true
          }
        ]
      }
    ],
    "search_engines": [
      {"id": 1, "name": "Google", "url_template": "...", "is_default": true}
    ]
  }
}
```

---

## 开发阶段

### Phase 0 · 脚手架（0.5 天）
- [ ] 仓库初始化、`.gitignore`、`Makefile`
- [ ] 后端：`cmd/server` 起 Gin，挂 `/api/health`
- [ ] 前端：Vite + Vue 3 + TS + Naive UI 起空壳
- [ ] Vite `build.outDir` 指到 `backend/web/dist`，后端 `embed.FS` 服务前端
- [ ] `make dev` 同时跑前后端，前端代理 `/api` 到后端
- **验收**：`go run ./cmd/server` → 浏览器看到 Vue 默认页

### Phase 1 · 认证 + DB（1 天）
- [ ] GORM 接入 modernc.org/sqlite，AutoMigrate 全部表
- [ ] users 表为空时，前端进入"初始化设置密码"流程
- [ ] JWT + httpOnly Cookie，中间件 `RequireAuth`
- [ ] 登录页 + 改密页
- **验收**：首次访问要求设密码 → 登录 → 进入 admin 空壳页

### Phase 2 · 核心 CRUD（2 天）
- [ ] Groups CRUD + 拖拽排序
- [ ] Cards CRUD（含内外网双地址）+ 拖拽排序
- [ ] 公开首页：分组栅格 + 卡片网格（无图标先用占位）
- [ ] 网络切换器（顶栏开关，localStorage 记忆）
- **验收**：管理后台增删改查完整跑通，公开首页正常渲染并可切内外网

### Phase 3 · 图标 + 搜索（1.5 天）
- [ ] 图标上传（限制 2MB、png/jpg/svg/webp）
- [ ] 图标三态：URL / 上传 / 预设（预设打包几十个常见服务图标）
- [ ] 搜索引擎 CRUD
- [ ] 首页搜索框 + 引擎切换（默认引擎 + 下拉切换）
- **验收**：能传图标、能切搜索引擎并发起搜索

### Phase 4 · 站点设置 + 备份（1 天）
- [ ] 设置页：标题、问候语、背景、公开模式、默认网络、主题
- [ ] JSON 导出/导入（merge / replace 两种模式）
- [ ] 公开模式开关：关闭时整站要求登录
- **验收**：导出 → 清库 → 导入 → 数据完整恢复

### Phase 5 · 打磨 + 部署（1 天）
- [ ] 主题切换（明/暗/跟随系统）
- [ ] 卡片右键菜单（复制链接、编辑、删除）
- [ ] 多阶段 Dockerfile + docker-compose.yml
- [ ] README：部署、备份、升级、常见问题
- [ ] GitHub Actions：tag 触发构建多架构镜像（amd64 + arm64）
- [ ] **CI pipeline 必须包含 `cd frontend && npm run test`**（Phase 2.3 引入了 vitest 单元测试，回归发现回归靠这个）
- [ ] CI 还要含：`go vet ./...`、`go test ./...`（如果 backend 有 _test.go 的话）、前端 `npm run build` 通过
- **验收**：`docker compose up -d` 一行起，访问 `:3000` 看到首页

### 后续可选（不在 MVP）
- 多用户 / 权限分级
- iframe 服务状态探测（ping/http 健康检查显示绿点）
- 浏览器书签批量导入
- 全文搜索（卡片标题/描述）
- PWA（离线访问首页）

---

## 关键决策记录

1. **为何选 modernc.org/sqlite 而不是 mattn/go-sqlite3**
   纯 Go 实现免 CGO，Docker 镜像可以用 distroless/static，最终镜像 < 20MB；交叉编译到 ARM（NAS 常见）也免去 musl/gcc 折腾。代价是性能略低，但导航面板的读写量完全无所谓。

2. **为何前端嵌入二进制而不是分开部署**
   你要的是"单文件部署"。分开部署意味着 nginx + 静态文件 + 反代，复杂度爆炸。`go:embed` 把 `dist/` 打进去后，最终就是一个 `moon-panel` 可执行文件 + 一个 `data/` 目录。

3. **为何用 httpOnly Cookie 而不是 Bearer Token**
   localStorage 任何 XSS 都能拿到 token。httpOnly Cookie 至少 XSS 拿不到，配合 SameSite=Lax 也能挡掉绝大多数 CSRF。前后端同源（嵌入部署）天然没有跨域问题，无须 CORS。

4. **为何不做用户系统**
   你明确说了"不要邮箱注册那一套"。单密码 = 一行 users 记录，初始化时设置一次，改密码走 `/api/auth/password`。如果未来要多用户，加 `role` 字段即可，不破坏现有结构。

5. **内外网切换在前端做还是后端做**
   前端做。后端只返回两个字段，前端按 localStorage 里的偏好渲染 `href`。好处：服务器不需要识别访问者来源，逻辑简单；用户在公司点"内网"也能访问公司内网（如果他们 VPN 进来了）。

---

## 准备就绪后的第一步

确认这份 PROJECT.md 后，从 Phase 0 开始：建仓库、装脚手架、跑通 hello world。
