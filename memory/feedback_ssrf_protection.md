---
name: SSRF 防护规则（Phase 2.5b 背景图 fetch + Phase 3 icon fetch 共用）
description: 后端任何"用户输入 URL 然后 fetch 资源"的端点必须走 SSRF 校验：DNS 解析 + IP 段拒绝 + 复用 IP 防 rebinding
type: feedback
---

## 何时适用

任何"用户在 admin 输入 URL → 后端 fetch 该 URL → 存到本地"的端点，包括但不限于：
- Phase 2.5b: `POST /api/admin/upload/background`（如果支持 URL 输入）
- Phase 3: `POST /api/admin/icons/fetch`
- 未来任何"代理下载"功能

**不适用**于纯文件上传（multipart form 直接传文件，无后端 fetch）。

## 拒绝列表（默认硬性禁止）

后端解析用户输入的 URL → DNS 查 → 拿到 IP → 检查是否落入：

**IPv4 私网**：
- `10.0.0.0/8`
- `172.16.0.0/12`
- `192.168.0.0/16`
- `169.254.0.0/16`（含 link-local + 云元数据 `169.254.169.254`）

**IPv4 回环**：
- `127.0.0.0/8`

**IPv6 回环**：
- `::1/128`

**IPv6 链路本地**：
- `fe80::/10`

**多播 / 保留**：
- `224.0.0.0/4`（多播）
- `0.0.0.0/8`（保留）

落入任一段 → 返回 `400` 拒绝，不发任何对外请求。

## 允许的 override

两个环境变量（默认都不开）：

1. **`MOON_ALLOW_PRIVATE_FETCH=true`**：整体放行所有上述拒绝段。给"我 NAS 内部就要 fetch 内网图标"的场景。带这个开关时**仍然不允许 169.254.169.254**（云元数据永远拒绝，避免 cred 泄露）。

2. **`MOON_ALLOWED_FETCH_HOSTS=cdn.example.com,static.example.com`**：逗号分隔白名单。即使不开 PRIVATE_FETCH，列在白名单的主机也能走（即使它解析到内网 IP）。给"我 NAS 上跑了个本地 CDN"场景。

优先级：白名单 > PRIVATE_FETCH 开关 > 默认拒绝。

## DNS 防 rebinding

只校验 URL 字符串里的 IP 字面值不够——攻击者可以传 `http://attacker.com`，后端解析时 DNS 返回 `1.2.3.4`（公网，校验通过），fetch 时 DNS 再解析返回 `127.0.0.1`（DNS rebinding）。

正确做法：
1. URL parse 拿 hostname
2. `net.LookupIP(hostname)` 解析得到 IP 列表
3. 校验**所有**返回 IP 都不在拒绝段（任一落黑名单则拒绝整个 URL）
4. **fetch 时复用解析得到的 IP**——构造自定义 `http.Transport.DialContext`，把 hostname 替换成已校验的 IP。这样第二次 DNS 解析（fetch 那一刻）的结果如果变化也用不上。

Go 实现要点（Phase 2.5b/3 写代码时参考）：

```go
ips, err := net.LookupIP(host)
if err != nil { return err }
for _, ip := range ips {
    if isBlocked(ip, cfg) { return ErrSSRF }
}
// Build a transport that pins to ips[0] (or pick best)
transport := &http.Transport{
    DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
        // addr is "host:port"; replace host with pinned IP
        _, port, _ := net.SplitHostPort(addr)
        return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
    },
    TLSClientConfig: &tls.Config{ ServerName: host }, // keep SNI = original host for cert validation
}
```

## 其他防御

- **HTTP redirect 也要走同一套校验**：原始 URL 通过 SSRF check，但 301 跳转到 `http://192.168.1.1/` 就 bypass 了。自定义 `CheckRedirect` 函数，对每次 redirect 的 URL 重新走校验。
- **下载大小限制**：用 `io.LimitReader` 限制读取字节数（图标 5MB / 背景图 20MB？）。防止 attacker 喂巨大文件耗内存。
- **Content-Type 白名单**：图标只接 `image/png|jpg|webp|svg+xml`；svg 还要进一步过滤防 XSS（svg 可以含 `<script>`）。
- **超时**：`http.Client.Timeout = 10s`，避免 fetch 慢端点堵线程。

## How to apply

- Phase 2.5b 实现 `/api/admin/upload/background` URL 输入分支时（如果有）必须套这套
- Phase 3 实现 `/api/admin/icons/fetch` 时同款
- 抽出公共函数 `internal/security/ssrf.go`，两端复用
- 单元测试：构造黑名单 IP 的 hostname mock，验证拒绝
- 用户授权 `MOON_ALLOW_PRIVATE_FETCH` 时启动日志 WARN 一行，提醒攻击面扩大
