# Moon Panel

自托管个人导航面板。Sun-Panel 的轻量替代品。

> **Phase 1（当前）**：项目骨架 + 数据库 + 单密码认证。还没有卡片管理 UI —— 那是 Phase 2。

## 设计要点

- **默认主页公开**，未登录就能看；只有 `/admin` 要密码
- **单密码**，无邮箱、无注册、无多用户
- **会话 30 天**，每次访问自动续期（每 24 小时签发新 cookie）
- 配置开关 `MOON_PUBLIC_MODE=false` 可让主页也要求登录
- httpOnly Cookie 存 JWT —— 比 localStorage 抗 XSS
- 所有依赖开源免费、无需联网激活

## 先决条件

- **Go 1.22+** —— 后端编译和运行
- **Node 18+ / npm** —— 前端构建（你机器上是 Node 20.20.2 ✓）
- 不需要 gcc、不需要 CGO（用 `modernc.org/sqlite` 纯 Go SQLite）

> 如果机器上还没装 Go：从 [https://go.dev/dl/](https://go.dev/dl/) 下载 Windows installer，一路下一步即可。安装后**新开一个终端**让 PATH 生效，然后 `go version` 验证。

## 第一次运行

### 1. 装依赖

```bash
# 后端
cd backend
go mod tidy

# 前端
cd ../frontend
npm install
```

### 2. 起开发环境（两个终端）

**终端 A：后端**
```bash
cd backend
go run ./cmd/server
# 监听 :3000，CORS 默认允许 http://localhost:5173
```

**终端 B：前端**
```bash
cd frontend
npm run dev
# 监听 :5173，/api/* 自动代理到 :3000
```

> Mac/Linux 用户可以直接 `make dev` 一条命令同时起两个。Windows 因为 Make 的 `&` / `wait` 行为不一致，建议两个终端分开跑。

### 3. 浏览器访问

- 主页（公开）：[http://localhost:5173](http://localhost:5173) —— 应该看到 "Moon Panel" 占位卡片，无需登录
- 管理后台：[http://localhost:5173/admin](http://localhost:5173/admin) —— 第一次会跳转到 `/login`，提示**首次启动设置密码**
- 设置好密码后，自动进入管理后台占位页

## 验证清单

按顺序点一遍，全过即 Phase 1 验收通过：

1. **主页公开** —— 隐身窗口打开 `/`，能看到内容（不重定向到登录）
2. **首次密码** —— `/admin` 跳转到 `/login`，显示"首次启动·设置管理员密码"
3. **登录跳转** —— 设置密码后跳进 `/admin` 看到"概览"卡
4. **会话保持** —— 关闭浏览器 → 重开 → 直接访问 `/admin` 仍然是登录态
5. **退出** —— 点右上角"退出" → 跳回 `/login`，显示"管理员登录"（不再是"首次启动"）
6. **错误密码** —— 故意输错 → 提示 "invalid credentials"
7. **API 健康** —— 浏览器或 curl 访问 [http://localhost:3000/api/health](http://localhost:3000/api/health) 返回 `{"code":0,...}`
8. **私有模式** —— 停掉后端，加环境变量 `MOON_PUBLIC_MODE=false` 再起，访问 `/` 应该被拒（前端会拿到 401，最终 Phase 2 会处理跳转）

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `MOON_PORT` | `3000` | 监听端口 |
| `MOON_DATA_DIR` | `./data` | SQLite + 上传文件存放目录 |
| `MOON_PUBLIC_MODE` | `true` | `false` 时主页也要登录 |
| `MOON_ADMIN_PASSWORD` | _(空)_ | 仅在 users 表为空时使用，用于无人值守 Docker 部署 |
| `MOON_JWT_SECRET` | _(自动生成)_ | 留空则自动生成并存到 `data/jwt.key`，重启不会失效 |
| `MOON_TOKEN_TTL_DAYS` | `30` | Session 有效期天数 |
| `MOON_COOKIE_SECURE` | `false` | 生产 HTTPS 部署时设 `true` |
| `MOON_CORS_ORIGINS` | `http://localhost:5173` | 逗号分隔；生产同域部署时可清空 |

## 重置密码

不小心忘了密码：

```bash
# 删 users 表中那行（或整个 db）
rm data/moon.db
# 下次启动会重新进入"首次设置密码"流程
```

如果只想重置而不丢配置：

```bash
sqlite3 data/moon.db "DELETE FROM users;"
```

## 项目结构

```
moon-panel/
├── backend/
│   ├── cmd/server/main.go          # 入口
│   ├── internal/
│   │   ├── api/                    # HTTP handlers
│   │   ├── auth/                   # JWT + bcrypt
│   │   ├── config/                 # env vars
│   │   ├── middleware/             # auth gate, cookie helpers
│   │   ├── model/                  # GORM models
│   │   └── store/                  # DB open + migrate
│   └── web/                        # 前端 dist 嵌入位
├── frontend/
│   └── src/
│       ├── api/                    # axios + 各资源 client
│       ├── stores/                 # Pinia
│       ├── router/
│       └── views/                  # Home / Login / admin/*
├── data/                           # 运行时数据（gitignore）
├── PROJECT.md                      # 全阶段设计文档
└── README.md                       # 本文件
```

## 路线图

- Phase 1 ✅ 骨架 + 认证（你在这）
- Phase 2 分组与卡片 CRUD + 内/外网切换
- Phase 3 图标上传（自托管 Lucide / Iconify）+ 搜索引擎
- Phase 4 站点设置 + JSON 备份导入导出
- Phase 5 主题、Docker 单文件镜像、多架构构建

完整设计见 [PROJECT.md](PROJECT.md)。
