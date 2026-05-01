---
name: 数据库重置后必须先在浏览器/init 接口设密码再做其他测试
description: rm moon.db 或清 users 表后系统回到首次启动态，admin 不存在；要先 init 密码再做 SSH/curl 测试
type: feedback
---

测试期间如果重置过数据库（`rm data/moon.db` 或 `DELETE FROM users`），系统回到**首次启动态**——`users` 表为空，admin 账户不存在。这时候 SSH 上去直接 curl `/api/auth/login` / 任何 `/api/admin/*` 都会失败（没有可登录的账户）。

**必须先做的事**（二选一）：

**方案 A（推荐，匹配真实用户流程）**：浏览器打开任意 `/admin` 或 `/login`，前端检测到 `initialized: false` 后会自动跳到"首次启动·设置管理员密码"表单，设完直接登录态进入。

**方案 B（仅自动化场景）**：curl 直接调 init 接口：
```bash
curl -sS -X POST -H "Content-Type: application/json" \
  -d '{"password":"test1234"}' http://localhost:3000/api/auth/init
```
返回 `{"code":0,"data":{"username":"admin"}}` 即创建成功，cookie 也已设上，后续可直接 curl `/api/admin/*`。

**Why:** 这是 Phase 2.2 测试时踩的坑——DB 重置后直接 SSH curl 测新功能，全部 401，浪费时间排查"为啥认证不通"才意识到 admin 没了。

**How to apply:**
- 任何提议"清测试数据"或"重置 DB"的命令时，**同时给出 init 命令或浏览器步骤**
- 比如交付清单里写 "rm /data/moon.db 后请到浏览器 /admin 重设密码再继续测试"，不要光给 rm 不给后续
- 检测端点 `GET /api/auth/me` 的 `initialized: false` 字段是判断当前是否首次启动态的可靠信号
