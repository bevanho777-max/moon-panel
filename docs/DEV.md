# 本地开发指南 (DEV)

Moon Panel 本地 dev 工作流 — **不依赖 docker rebuild**，前后端 hot reload，迭代周期亚秒级。

> Docker 全清三连仅用于**最终交付验收**（开源 release 前 / 部署到 NAS 前）。日常开发 / phase polish / bug 修复全程本地 dev。

---

## 1. 环境准备

| 工具 | 最低版本 | 验证命令 |
|---|---|---|
| Go | 1.23+ | `go version` |
| Node.js | 20 LTS+ | `node -v` |
| npm | 跟随 Node | `npm -v` |
| **可选**：[air](https://github.com/air-verse/air)（Go hot reload） | 任意 | `air -v` |

安装 air（一次性）：

```bash
go install github.com/air-verse/air@latest
```

第一次进项目，安装前端依赖：

```bash
cd frontend
npm install
```

---

## 2. 启动 dev（两个 terminal）

> **关于 dev 默认密码：** 下面用 `devdev99` 作为本地 dev 的 admin 密码。
> Phase 3d-1 加了 **8 字符强度下限**（[auth.go:68](../backend/internal/auth/auth.go#L68)
> `password too short (min 8 characters)`），dev 环境也走同样的校验路径
> （bootstrap 内部调 `HashPassword`），不能用 `dev` / `123` 这种短密码 —
> 启动会直接 fail。要换其他 8+ 字符密码可自由替换；记得用同一密码登录。

### Terminal A — 后端 (:3001)

**Windows 一行式（推荐）**：

```powershell
cd backend
.\dev.ps1                    # 默认 air 模式（hot reload）
.\dev.ps1 -NoAir             # 退化为 go run（air 没装时）
```

**Bash / macOS / Linux**：

```bash
cd backend
MOON_PORT=3001 MOON_ADMIN_PASSWORD=devdev99 MOON_DATA_DIR=./data-dev air         # 推荐
MOON_PORT=3001 MOON_ADMIN_PASSWORD=devdev99 MOON_DATA_DIR=./data-dev go run ./cmd/server   # 不用 air
```

**PowerShell 手写（不用脚本）**：

```powershell
cd backend
$env:MOON_PORT='3001'; $env:MOON_ADMIN_PASSWORD='devdev99'; $env:MOON_DATA_DIR='./data-dev'; air
```

启动成功输出：

```
moon-panel listening on :3001 (env=development public_mode=true ...)
[GIN-debug] Listening and serving HTTP on :3001
```

### Terminal B — 前端 (:5173)

**Windows 一行式**：

```powershell
cd frontend
.\dev.ps1
```

**通用**：

```bash
cd frontend
npm run dev
```

浏览器开 [http://localhost:5173](http://localhost:5173)。Vite 自动 proxy `/api/*`、`/uploads/*`、`/assets/wallpapers/*` 到 :3001 后端。

登录：用户名 `admin`，密码 `devdev99`（或你启动时传的 `MOON_ADMIN_PASSWORD`）。

---

## 3. 端口约定

| 端口 | 角色 | 备注 |
|---|---|---|
| **3000** | 生产 backend（NAS 容器） | 不动，prod 容器始终在 :3000 |
| **3001** | dev backend（本地 go run / air） | 跟 prod 隔离，可同时跑 |
| **5173** | dev frontend（Vite dev server） | proxy 到 :3001 |

---

## 4. 数据：dev / prod 完全隔离

| 路径 | 用途 |
|---|---|
| `backend/data-dev/` | dev 数据（moon.db / uploads / jwt.key）— 启动时 `MOON_DATA_DIR=./data-dev` |
| `<NAS>:/volume5/code/moon-panel/data/` | prod 数据 — docker 容器挂载 |

`data-dev/` 已在 `.gitignore`（如未配置见第 9 节）— 不入库，本地脏数据无所谓。

### 4.1 把 prod 数据导入 dev（首次开发某个真实场景时）

利用 Phase 4c 备份功能：

1. **Prod admin → 站点设置 → 备份与恢复 → 导出 ZIP** 拿到 `moon-panel-backup-YYYYMMDD-HHMMSS.zip`
2. 启动本地 dev backend（步骤 2 Terminal A）
3. 浏览器 [http://localhost:5173/admin/site-settings](http://localhost:5173/admin/site-settings) → 备份与恢复 → 从备份恢复
4. 上传那个 zip → 一键恢复（含 uploads/）
5. dev 现在跟 prod 数据完全一致；测试不会污染 prod

> ⚠️ 反向操作（dev → prod）**不要做**。dev 改动应该走代码层（git commit + docker rebuild），不是数据层。

---

## 5. Hot reload 机制

### 前端
Vite HMR 原生支持。改 .vue / .ts / .css 浏览器自动局部刷新，无需手动 reload。

### 后端
- 用 air：`.go` 改动自动 rebuild + 重启进程，约 1-3 秒，Listening 后继续测
- 不用 air：改完 `Ctrl+C` → 重跑 `go run`，约 3-5 秒

`air` 配置在 [backend/.air.toml](../backend/.air.toml)，watch `*.go`，忽略 `tmp/` `data-dev/` `web/dist/`。

---

## 6. 调试小贴士

### 6.1 admin 默认密码
`MOON_ADMIN_PASSWORD=devdev99` 启动 → 用户名 `admin` 密码 `devdev99`（**必须 ≥ 8 字符**，Phase 3d-1 强度校验）。如果 `data-dev/moon.db` 已有 admin 用户，env 被忽略（保护 prod 行为，dev 同样如此 — 想换密码先删 `data-dev/`）。

### 6.2 看 SQL
GIN debug 模式默认带 GORM SQL 日志（`[X.XXXms] [rows:N] SELECT ...`）。`MOON_ENV=production` 关日志（dev 不要设）。

### 6.3 重置 dev 数据库
```bash
rm -rf backend/data-dev
```
下次启动重新 bootstrap admin / 默认搜索引擎 / 默认城市等。

### 6.4 测壁纸 / 主题色
- 默认 dev 没壁纸 → admin 站点设置 → 背景壁纸 → 选 builtin night/aurora/graphite，或上传图片测 canvas 压缩
- 改 acrylic CSS 后浏览器 F12 → 跑：
  ```js
  document.querySelectorAll('.n-data-table *').forEach(el => {
    const bg = getComputedStyle(el).backgroundColor
    if (bg !== 'rgba(0, 0, 0, 0)') console.log(el.className, bg)
  })
  ```
  验证 transparency 选择器是否覆盖到所有渲染层。

### 6.5 跨设备测（手机 / iPad 上访问 dev）
默认 Vite 只监听 localhost。要让局域网其他设备访问：

```bash
cd frontend
npm run dev -- --host
```

Vite 输出会显示 `Network: http://192.168.x.x:5173` — 用那个地址在手机浏览器开。后端 :3001 默认 `0.0.0.0` 监听，proxy 自动跟着走。

---

## 7. 提交前自检（Optional）

```bash
# 后端
cd backend
go build ./...               # 编译检查
go vet ./...                 # 静态检查

# 前端
cd ../frontend
npm run build                # vue-tsc + vite build，模拟 docker 中要做的事
```

跑完没报错就大致 OK，可以 commit。

---

## 8. Docker 验收（最终交付前一次性做）

每次 phase 收尾 / 开源 release 前才做。**不在每次 commit 后做**。

```bash
# 全清三连（清旧镜像 + 清旧容器 + 清旧 volumes）
docker compose down -v
docker rmi moon-panel:latest
docker system prune -f

# 重新 build + 启
docker compose up -d --build

# 看 log 确认启动 OK
docker compose logs -f moon-panel
```

成功标志：`moon-panel listening on :3000 (env=production ...)`。

---

## 9. .gitignore 推荐内容

确保下面这些不入库：

```
# Go
backend/data-dev/
backend/tmp/
backend/build-errors.log
backend/dist/
backend/web/dist/

# Frontend
frontend/node_modules/
frontend/dist/
frontend/tsconfig.tsbuildinfo

# Editor / OS
.idea/
.vscode/
.DS_Store
Thumbs.db
```

---

## 10. Troubleshooting / 常见问题

### 10.1 启动失败类

**Q: `bootstrap admin: password too short (min 8 characters)`**
→ Phase 3d-1 强度校验：admin 密码至少 8 字符。本地 dev 也走同一校验路径（`HashPassword` defense-in-depth）。把 `MOON_ADMIN_PASSWORD=dev` 改成 `devdev99` 或任意 ≥ 8 字符的字符串再启。

**Q: backend 启动报 `address already in use :3001` / `bind: ...`**
→ 之前的 dev 进程没退干净。
- **Windows**: `netstat -ano | findstr :3001` 找 PID → `taskkill /PID <pid> /F`
- **Mac/Linux**: `lsof -i :3001` → `kill <pid>`

**Q: `'air' is not recognized` / `air: command not found`**
→ 用 `go install` 装的二进制在 `%USERPROFILE%\go\bin`（Win）/ `~/go/bin`（Mac/Linux），需要把这个目录加进 `PATH`。
- **Windows PowerShell（当前 session）**：
  ```powershell
  $env:Path += ";$env:USERPROFILE\go\bin"
  ```
- **Windows 永久**：系统属性 → 环境变量 → 用户变量 `Path` 加 `%USERPROFILE%\go\bin`，重开 terminal
- **Mac/Linux**（写入 `~/.zshrc` 或 `~/.bashrc`）：
  ```bash
  export PATH="$HOME/go/bin:$PATH"
  ```
- **没装可以现装**：`go install github.com/air-verse/air@latest`
- **不想装也能用**：`.\dev.ps1 -NoAir` 或直接 `go run ./cmd/server`，每次手动 ctrl+c 重启即可

**Q: 前端访问 `/api/*` 返回 404 HTML（Vite 兜底页）**
→ Vite proxy 没生效。检查：
1. [vite.config.ts](../frontend/vite.config.ts) `proxy['/api'].target` 是 `http://localhost:3001`
2. 后端确实在 `:3001` 监听（看 backend 启动日志最后一行）
3. 改了 `vite.config.ts` 后必须**重启 vite dev server**（HMR 不会重新加载配置文件）

**Q: API 请求走错端口（如 `/api/auth/me` 500，但 backend log 没看到这次请求）**
→ ★ shadow-config 陷阱：`frontend/` 目录里同时存在 `vite.config.ts` 和 `vite.config.js`。Vite resolution 顺序是 `js > mjs > ts > cjs > mts > cts`，**任何 `.js` 都会优先吃掉 `.ts`**，让你以为在跑新配置实际跑老的。

```bash
ls frontend/*.config.*    # 应该只有 vite.config.ts，没有 .js / .cjs / .mjs / .d.ts
```

如果有 `.js` / `.d.ts`：删掉它们（通常是过去某次 `tsc` 误编译的产物）。已通过 `.gitignore` 永久拦截 `frontend/*.config.{js,cjs,mjs,d.ts}` — 见 [.gitignore](../.gitignore)。同样的坑也适用于其他 `*.config.ts` 工具（vitest、eslint、tailwind 等）：发现某个配置改了不生效，先看是不是被同名 `.js` 遮蔽。

**Q: 改 .go 文件 air 没自动重启**
→ 看 `backend/build-errors.log` 是否有编译错误。air 编译失败会停在错误状态，修好语法错就会自动恢复。

**Q: 上传图标 / 壁纸返回 500**
→ 检查 `data-dev/uploads/` 目录权限。Win 一般没问题，Mac/Linux 启动用户对该目录要可写。

### 10.2 PowerShell vs Bash 语法对照

Windows 用户最容易踩坑的就是 env var 设法。下面三种等价：

| 场景 | Bash / WSL / Mac / Linux | PowerShell（Win 终端） | cmd.exe（Win） |
|---|---|---|---|
| 一行式 inline | `MOON_PORT=3001 air` | `$env:MOON_PORT='3001'; air` | `set MOON_PORT=3001 && air` |
| 多个 env | `A=1 B=2 air` | `$env:A='1'; $env:B='2'; air` | `set A=1 && set B=2 && air` |
| Persistent 当前 session | `export A=1` | `$env:A='1'` | `set A=1` |
| Persistent 永久 | 写 `~/.zshrc` | `[Environment]::SetEnvironmentVariable('A','1','User')` | 系统属性 GUI |
| 调用脚本 | `./script.sh` | `.\script.ps1` | `script.bat` |
| 路径分隔 | `/` | `\` 或 `/`（PS 也支持 `/`） | `\` |

**最大坑**：Bash 写法 `MOON_PORT=3001 air` 在 PowerShell 里**不会报错也不会生效** — PS 把 `MOON_PORT=3001` 当成位置参数传给 `air`，env var 没设上。所以 [DEV.md §2 启动 dev] 里 Win 用户必须用 `$env:VAR=...` 或调 `dev.ps1`。

### 10.3 视觉调试类

**Q: 改 acrylic CSS 后 F12 看 backgroundColor 还是不对**
→ NaiveUI cssr class 命名不规整（同一组件可能同时有 `.n-data-table-base-table` 和 `.n-data-table-table` 等相似名）。先 forEach 列实际渲染 class 再写选择器：

```js
document.querySelectorAll('.n-data-table *').forEach((el) => {
  const bg = getComputedStyle(el).backgroundColor
  if (bg !== 'rgba(0, 0, 0, 0)') console.log(el.className, bg)
})
```

修完 CSS 再跑一遍验证返回值符合预期。

---

## 11. 这套 workflow 的设计原则

1. **dev 速度优先**：本地 hot reload < 1s < air rebuild < 3s < docker rebuild ~5min。日常迭代不接受 docker。
2. **dev / prod 完全隔离**：不同端口（3001 / 3000）、不同数据目录（`data-dev/` / `<NAS>/data/`），互不污染。
3. **真实数据走备份桥**：用 Phase 4c 备份功能 prod → dev，避免手动拷数据库。
4. **docker 仅交付时用**：全清三连一次性验收，不进迭代循环。

> Phase 5 + 后续 polish 全程本地 dev。Docker 只在准备开源 release / 部署到 NAS 前用一次。
