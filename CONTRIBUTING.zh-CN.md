# 贡献指南

感谢考虑给 Moon Panel 贡献代码！这份文档说明工作流、约定和 review checklist，
目的是让 `main` 分支始终处于"可发布"状态。

[English](CONTRIBUTING.md) · [中文版](CONTRIBUTING.zh-CN.md)

如果不确定一个想法是否合适 — 先开 issue 讨论。Moon Panel 设计上就是
单用户、无账号、有取舍的面板；常见的功能请求（多用户 / SSO / 邮箱注册）
**按设计**就不在范围内。README 的 Roadmap 章节是计划中功能的权威来源。

## 开发环境搭建

完整本地开发流程（Go + air + Vite 热重载、端口约定、数据隔离、常见排错）见
[docs/DEV.md](docs/DEV.md)。简版：

```bash
# 后端 — Terminal A (端口 3001)
cd backend
go mod tidy
.\dev.ps1                   # Windows PowerShell 一键脚本（air 模式）
# 通用写法：
MOON_PORT=3001 MOON_ADMIN_PASSWORD=devdev99 MOON_DATA_DIR=./data-dev air

# 前端 — Terminal B (端口 5173)
cd frontend
npm install
npm run dev                 # 自动 proxy /api/* 和 /uploads/* 到 :3001
```

浏览器开 http://localhost:5173，用 `admin` + 启动时设的 `MOON_ADMIN_PASSWORD`
登录。

必需工具：**Go 1.23+**、**Node 20+**。推荐：`air` 后端热重载
（`go install github.com/air-verse/air@latest`）。

## 分支与 PR

- Fork 仓库，在你 fork 上建 topic branch。
  `feat/widget-foo`、`fix/login-redirect`、`docs/contributing-tweak` 这种
  好读的名字 — 描述性即可。
- 每个分支只做一件事。多个小 PR 比一个巨 PR review 起来更快。
- Push 后向 `main` 开 PR。PR 模板会自动加载 — 填完，里面的 checklist 是
  review 的入门门槛。
- CI 必须全绿才能 review（Frontend job: typecheck + vitest + Playwright
  e2e；Backend job: vet + build）。

## Commit 信息规范 — Conventional Commits

我们遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <祈使句简短描述>
```

允许的 type：

| Type | 用途 |
|---|---|
| `feat` | 新的用户可见功能 |
| `fix` | bug 修复 |
| `docs` | 仅文档（README / CONTRIBUTING / 注释） |
| `refactor` | 重构，不改变行为 |
| `test` | 加测试 / 改测试 |
| `chore` | 工具链 / 依赖 / dotfile |
| `ci` | CI/CD 配置 |
| `perf` | 性能优化 |

scope 可选，但代码区域较大时建议加：
`feat(cards): bulk delete`、`fix(auth): handle empty cookie`。

短标题用单 `git commit -m "..."`。需要展开 **why** 时（超过 70 字符）再加
第二个 `-m ""`。第一行是 `git log --oneline` 显示的内容，要写到位。

示例：

```
fix(security): per-IP login lockout, not global

之前的全局计数让一个攻击者就能 DoS 锁住所有用户。改为按源 IP 计数，
15 分钟窗口，第 3 次锁定后指数退避。CIDR 信任白名单内的源跳过计数器
但仍写审计日志。

手动用两个源 IP 跑 curl 验证独立计数器。CI vet + build 覆盖布线。
```

## PR review checklist

请求 review 前自查：

- [ ] CI 全绿（Frontend ✓ + Backend ✓）。
- [ ] PR 描述说清楚 **改了什么** 和 **为什么**（不只是 what）。
- [ ] UI 可见的改动，附 before/after 截图。如果改了移动端布局，桌面 +
  移动各一张。
- [ ] 合理范围内补测试。这是个人面板，不是银行系统 — 但 auth / 安全 /
  数据变更路径值得加测试。
- [ ] 不带无关 formatting 抖动。diff 越小越好。
- [ ] 没把 secret 提进去（密码、token、真实 IP、API key）。万一漏了，
  merge 前 force-push 干净版本可以，但要在 PR 里说一下让 reviewer 知道
  泄露的值要 invalidate。

Reviewer 预期：48 小时内给出第一轮 review。如果 reviewer 提了改动建议，
在同一分支上加新 commit — review 期间**不要 force-push**，除非是 rebase
准备 merge。

## 不接受的改动

为了节省大家时间，下面这些直接拒绝：

- **多用户 / RBAC** — Moon Panel 设计上就是单用户。
- **邮箱注册 / 密码找回 / OAuth / SSO** — 同上。
- **遥测、分析、"phone-home" 功能** — local-only 是 feature，不是 bug。
- **没事先讨论的大功能 PR** — 先开 issue 谈方向。直接 PR 改架构的可能
  不 merge 直接关。
- **vendor 依赖** — 用 `go mod` / `package.json` 走正常路径。

## Issue 模板

开 issue 时挑最合适的模板：

- **Bug report** — 文档说能用但实际不行的。附复现步骤、预期 vs 实际、
  环境信息。
- **Feature request** — 你觉得有用的新能力。说明 use case 和你考虑过的
  替代方案。

两个模板在 [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/) 下。PR 模板
[.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md)
开 PR 时自动加载。

## 协议

提交贡献即同意你的代码以项目使用的 [MIT License](LICENSE) 发布。
