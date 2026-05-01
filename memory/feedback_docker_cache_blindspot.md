---
name: Docker layer cache 对小幅 backend 改动的盲区
description: 增量加 handler/路由时 BuildKit 可能跳过 go build 层，导致镜像里仍是旧二进制
type: feedback
---

**症状：** Backend 改动很小（比如只新增一个 handler 文件、main.go 加一行 register），用户跑 `docker compose build` 甚至 `--no-cache` 都过了，但起容器后**新路由 404、行为跟旧代码完全一致**。

**根因：** Docker BuildKit 的 layer cache 在某些情况下会命中 `COPY backend/ ./` 那一步，跳过后续 `RUN go build`，导致镜像里塞的是上次缓存的旧二进制。`docker compose build --no-cache` 在某些场景下**不会清 BuildKit 自己的 cache 池**——这是 docker 已知盲区，不是用户操作错误。

可能触发条件（不全）：
- 源码在 SMB / NFS 等远程文件系统上，mtime 元数据不稳定
- BuildKit cache 池里有 layer 哈希匹配的旧条目
- COPY 步骤的输入文件集合 BuildKit 判定"未变"

**真正能根治的清法**（用户 2026-04-28 实测可用）：
```bash
sudo docker image rm moon-panel:latest -f
sudo docker builder prune -af
sudo docker compose build --no-cache --pull
```

三连：删旧镜像 → 清 BuildKit 全部缓存 → 拉新基础镜像并无缓存重 build。

**诊断手段**：怀疑塞旧二进制时，进容器看二进制 mtime：
```bash
sudo docker exec moon-panel ls -la /app/moon-panel
```
如果 mtime 比当前 build 早很多（甚至是上次 phase 的时间），就是这个问题。

## 加入 backend 交付清单

每次 backend 有**路由层 / handler 层**改动（新文件、新 register 调用、新接口）时，在交付文档"你跑的步骤"里加一条提示：

> **如果新路由仍 404**，是 Docker BuildKit cache 盲区（不是你做错），全清重 build：
> ```bash
> sudo docker image rm moon-panel:latest -f
> sudo docker builder prune -af
> sudo docker compose build --no-cache --pull
> sudo docker compose up -d
> ```

**纯前端改动 / 仅文档改动不需要这条提示**（前端在 frontend 阶段构建，命中 cache 也是正确的）。

## 同一类盲区：frontend package.json 加新依赖（2026-04-29 用户撞）

**症状：** frontend 加了新 npm 依赖（如 lucide-vue-next）→ 普通 `docker compose build` 后起容器，dist 里没有新依赖代码（lucide chunk 不生成、import 失败）。

**根因：** 同一个 BuildKit cache 机制——`COPY frontend/package.json` 和 `RUN npm install` 这两层在 BuildKit 看来"输入未变"（或哈希误判）就被命中跳过，新依赖根本没下载。

**判断本轮 frontend 改动是否动了依赖：**
- 改了 `frontend/package.json` 的 `dependencies` / `devDependencies` → **必须 `--no-cache`**（最少这条；保险起见走全清三连）
- 只动 `.vue` / `.ts` / `.css` / scripts 字段 → 普通 build 即可

**3a-1 复盘：我曾说"frontend 改动不需要全清三连"——这话只在"package.json 不动"时成立。3a-1 实际加了 lucide-vue-next，应当至少 `--no-cache`。**

## 交付时"你跑的步骤"判断 flowchart（**2026-04-30 第三次修正**）

```
本轮改了哪一层？
├─ 仅文档 / 仅 memory                  → 不需要重 build
├─ 仅修改既有 frontend 文件内容         → docker compose build --no-cache --pull
├─ frontend 增新文件（.vue/.ts/.json）  → 全清三连
├─ frontend/package.json 增删依赖       → 全清三连
├─ backend 任何 .go / go.mod           → 全清三连
└─ Dockerfile / docker-compose / 入口脚本 → 全清三连

全清三连 = sudo docker image rm moon-panel:latest -f
        && sudo docker builder prune -af
        && sudo docker compose build --no-cache --pull
```

**`--no-cache` 单独使用降级为"次级保险"**：实测 SMB + BuildKit 组合下，`--no-cache` 不能拦住所有 cache 误判（3a-3 实测：用户跑了 `--no-cache` build 报告成功但 binary 还是 stale，full prune 才修）。**Main flow 默认走全清三连**。

**演进史**（3 次踩雷）：
1. 3a-1：我说"frontend 改动不需要全清"踩雷（package.json 加 lucide-vue-next）
2. 3a-3 第一次：我说"`.vue/.ts/.css` 普通 build 即可"踩雷（新加 `src/data/*.json` 数据文件）
3. 3a-3 第二次：用户跑 `--no-cache` build 报告成功但 dist 仍是 stale，**`--no-cache` 都拦不住**，全清三连才行

每次都是 BuildKit 在 SMB 源上 mtime 误判 + 哈希误命中。**结论：SMB + BuildKit 这对组合本质不可信，不要赌 cache 准确性。**

## 拿不准就全清三连

多花的几分钟 vs 一轮 stale dist 排查，前者完胜。

**保守原则**：拿不准时走全清三连——多花几分钟比 user 撞 cache 盲区再 debug 一轮值。

## 诊断容器内是否含某 frontend chunk（**重要**：dist 在 binary 里不在文件系统）

Moon Panel 用 `//go:embed all:dist` 把 frontend dist 编进 Go 二进制。**容器里没有 `/app/web/dist/` 目录**——所有静态资源都在 `/app/moon-panel` binary 里，运行时由 embed FS 服务。

错误命令（不会有输出）：
```bash
ls /app/web/dist/assets/         # 路径不存在
```

正确命令（grep binary 的 printable strings）：
```bash
sudo docker exec moon-panel sh -c \
  'grep -aoE "<chunk-pattern>-[A-Za-z0-9_-]+\.js" /app/moon-panel | sort -u | head'
```

例：检查 dashboard-icons + lucide-icons chunks 是否在 binary：
```bash
sudo docker exec moon-panel sh -c \
  'grep -aoE "(dashboard-icons|lucide-icons)-[A-Za-z0-9_-]+\.js" /app/moon-panel | sort -u | head'
```

也可以查 binary mtime / size 来判断是否真重新 build 过：
```bash
sudo docker exec moon-panel ls -la /app/moon-panel
```

每次 build 后 mtime 会更新，size 会随 dist 大小变化。

**写诊断命令前先想一下**：用户问的是"容器里有没有 X 文件"——X 是源码 / config 在 `/app/`、是 frontend asset 在 binary embed FS、还是数据在 `/data/`？三个位置对应三种 inspect 方式。

## How to apply

后端改动 / 依赖改动交付时主动写明对应清法。每次交付前自查："本轮 diff 涉及哪一层？" 写进交付文档"你跑的步骤"里，不要让用户撞上。
