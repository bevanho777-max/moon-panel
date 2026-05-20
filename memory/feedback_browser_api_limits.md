---
name: browser API 跨域 / redirect 边界 — 实测优先, 不要叠加 workaround
description: 涉及 CORS / no-cors / redirect / fetch options 组合 / Permissions API / Service Worker 时, 先 F12 console 跑最小重现验证浏览器真实 error message. 不能在 broken fetch 上叠加 workaround, 失败时退一步重新选 API (例 fetch → <img>).
type: feedback
---

# browser API 跨域 / redirect 边界

## 背景

v0.2.23 Phase B 自动 LAN/WAN 检测探针经历 3 patch 才落地.

Phase A 初版用:
```js
fetch(url, { mode: 'no-cors', cache: 'no-store', signal })
```
→ 假阴性: 192.168.1.20 → 301 redirect, fetch 跟随 follow → 第二请求 fail
  (协议升级 / CORS / 自签证书) → reject → 误判 WAN

Phase B-patch-2 加 `redirect: 'manual'` 想绕过:
```js
fetch(url, { mode: 'no-cors', redirect: 'manual', signal })
```
→ ★ Chrome 强制规则 ★: `mode: 'no-cors'` 必须搭配 `redirect: 'follow'`
→ 浏览器直接 sync TypeError "Request mode is 'no-cors' but the redirect mode
  is not 'follow'"
→ 根本没发起 fetch, 比之前更糟糕

Phase B-patch-3 改 `<img>` tag 探测才彻底解决.

## 错误模式

```js
// ❌ Phase A: no-cors 跟随 redirect, 第二请求 fail
fetch(url, { mode: 'no-cors' })
  .then(() => 'lan')
  .catch(() => 'wan')

// ❌ Phase B-patch-2: 强制 manual, 立即 TypeError
fetch(url, { mode: 'no-cors', redirect: 'manual' })
```

## 正确做法

```js
// ✓ <img> tag 探测, 无 CORS / redirect 限制
function probeViaImg(url: string, timeoutMs: number): Promise<boolean> {
  return new Promise((resolve) => {
    const img = new Image()
    let resolved = false
    const t = setTimeout(() => {
      if (resolved) return
      resolved = true
      img.src = ''
      resolve(false)  // timeout = WAN
    }, timeoutMs)

    const finish = (ok: boolean) => {
      if (resolved) return
      resolved = true
      clearTimeout(t)
      resolve(ok)
    }

    img.onload = () => finish(true)   // server returned image = LAN
    img.onerror = () => finish(true)  // server returned but not image = LAN (responded)
    img.src = url + (url.includes('?') ? '&' : '?') + '_probe=' + Date.now()
  })
}
```

诊断流程:
1. F12 console 跑最小重现脚本验证假设
2. 失败先看浏览器 console 真实 error message (例 "Request mode is...")
3. 不能叠加 workaround 修 (patch-2 错示范), 退一步重新选 API (patch-3 `<img>`)

## 适用范围

- 任何涉及跨域请求 / iframe / CORS / Permissions API / Service Worker 场景
- 任何 fetch options 组合 (mode / redirect / credentials) 改动
- 任何 "应该 work" 的 browser API 行为假设
- 触发关键词: no-cors, CORS, opaque, redirect, mode, cross-origin

## 关联 reference

- CLAUDE.md "Browser API" section (onboarding checklist)
- @memory/feedback_unverified_prod_work.md
