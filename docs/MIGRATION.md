# Moon Panel 升级与数据迁移

本文档记录跨版本升级时的数据迁移行为, 以及升级前的安全建议.

## 通用建议: 升级前 backup

不管什么版本升级, ★ 先 backup 你的 `moon.db` ★:

```bash
# Synology DSM / Linux NAS
docker compose stop moon-panel
cp /path/to/moon-panel/data/moon.db /path/to/moon-panel/data/moon.db.bak.$(date +%Y%m%d)
docker compose up -d
```

Moon Panel 也内置了 admin → 备份/恢复, 可导出 JSON 或 zip (含上传的图标/壁纸).

---

## v0.2.28 — A.5 multi-user roadmap R1 (地基)

### 用户感知: **无**

v0.2.28 是 multi-user 改造的第一步, **不引入任何用户可见的功能改动**:

- 登录页仍只输密码 (无 username 字段)
- 主页 / admin 后台界面跟 v0.2.27 完全一致
- 现有的分组 / 卡片 / 主题 / 壁纸 一切不变
- API 行为不变 (后续 R3 才引入 query 隔离)

### Schema 变更

`groups` 表 和 `cards` 表新增字段:

| 字段 | 类型 | 默认 | 含义 |
|---|---|---|---|
| `owner_id` | `INTEGER` | `0` (然后 migration 改成 admin.ID) | 数据归属用户 ID, 引用 `users.id` |

### 启动时自动 migration

容器启动时 `store.MigrateOwnerID(db)` 跑一次:

1. 找现有 admin user (`WHERE username = "admin"`)
2. 把所有 `groups.owner_id = 0` 和 `cards.owner_id = 0` 的行更新为 admin.ID
3. 在 transaction 内执行, 失败整体 rollback
4. **幂等**: 第二次启动 0 行匹配, 静默跳过

日志输出示例:

```
R1 migration: stamped owner_id=1 on 5 existing groups
R1 migration: stamped owner_id=1 on 23 existing cards
```

如果 admin 还未初始化 (空数据库 + 未设 `MOON_ADMIN_PASSWORD`), migration 静默跳过.

### 软回退到 v0.2.27 是安全的

如果升级 v0.2.28 后想回 v0.2.27 (e.g. 撞到非预期问题):

- ✓ 直接拉 v0.2.27 镜像即可
- ✓ v0.2.27 的 GORM 不认识 `owner_id` 字段, **直接忽略**, 数据完整
- ✓ 在 v0.2.27 期间新建的卡片 `owner_id = 0`; 回升 v0.2.28 时 migration 会再次填补
- ⚠️ 但不建议反复横跳, 升级前 backup 仍是首选保障

### Backup 恢复行为

恢复 v0.2.28 之前的 backup (不含 `owner_id`):

- 数据按 backup 原样恢复, `owner_id` 字段为 0
- 恢复完成后, `MigrateOwnerID` 自动跑一次, 把 0 改成当前 admin.ID
- 日志会显示 backfill 数量

恢复 v0.2.28+ 的 backup (含 `owner_id`):

- 直接保留原 `owner_id` 值
- `MigrateOwnerID` 跑但 0 行匹配, 静默

### 已知工程债

- `groups.owner_id` / `cards.owner_id` 没有 `NOT NULL` 约束 (AutoMigrate 在 SQLite 加 NOT NULL 不便). 应用层保证非 0 — R2/R3 强制 query 过滤时会校验.
- master admin user 删除会 cascade 删该 user 的所有 cards/groups. R1 阶段不开放删 user 接口, R6 才加 master admin 保护逻辑.

---

## A.5 后续 release 预告

| Release | 内容 | 用户感知 |
|---|---|---|
| v0.2.28 (R1) | Schema + migration 地基 | 无 |
| v0.3.0-alpha.1 (R2) | 登录加 username + admin 用户管理 API | admin 后台新增"用户管理"入口 |
| v0.3.0-alpha.2 (R3) | Card/Group 数据隔离 + 越权测试 | 多 user 各自看自己的卡片 (MVP) |
| v0.3.0-beta.1 (R4) | per-user 主题/壁纸 + 切换器挪首页 | 主题切换在首页 header |
| v0.3.0-beta.2 (R5) | SearchEngine/CityWidget 决策 | — |
| v0.3.0-rc.1 (R6) | Admin UI polish + 密码重置 + impersonate | admin 体验完善 |
| v0.3.0 (R7) | 开源文档 + 安全须知 | 正式版 |
