---
name: Gin 静态文件路由 + NoRoute SPA fallback 共存
description: r.Static() 注册的路由不会被 NoRoute 拦截，但前提是路由必须真的注册了；遗漏会导致静态资源 URL 返回 SPA HTML
type: feedback
---

## 现象（如果踩了）

访问 `/uploads/icons/abc.png`（或 `/static/foo.css` 等）浏览器拿到的是 `<html>` 文档而不是图片二进制。`<img src=...>` 显示成裂图，因为浏览器把 HTML 当图片解析失败。

## 原理

Gin 用 radix tree 做路由匹配。`r.Static("/uploads", dir)` 内部注册的是 `GET /uploads/*filepath` 通配符路由。**只有未注册的路径才会触发 r.NoRoute()**。所以：

- 注册了 `r.Static("/uploads", ...)` → `/uploads/x` 走静态 handler ✓
- 没注册 → `/uploads/x` 没路由 → NoRoute 触发 → 返回 index.html ✗

**严格说"声明顺序"不影响 Gin 路由——树是基于路径的，不是基于代码顺序。** 但代码可读性约定还是"具体路由先、NoRoute 最后"。

## 标准 main.go 顺序

```go
// 1. API 路由（最具体）
api := r.Group("/api")
api.RegisterHealth(apiGroup)
// ...各种 Register 调用

// 2. 静态文件（中等具体，前缀匹配）
uploadsDir := filepath.Join(cfg.DataDir, "uploads")
if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
    log.Fatalf("create uploads dir: %v", err)
}
r.Static("/uploads", uploadsDir)

// 3. SPA fallback（兜底，匹配剩下所有 GET）
fsys, _ := web.Sub()
r.NoRoute(func(c *gin.Context) {
    if strings.HasPrefix(c.Request.URL.Path, "/api/") {
        c.JSON(404, ...)
        return
    }
    web.SPAHandler(fsys)(c)
})
```

## 调试 checklist

`/uploads/foo.png` 返回 HTML 时按顺序查：

1. `r.Static("/uploads", absDir)` 是否真的调用了？grep main.go 一眼
2. `absDir` 路径是否真存在 + 有读权限？`docker exec moon-panel ls /data/uploads`
3. 文件名大小写、扩展名是否匹配？容器内 `ls /data/uploads/icons/` 看实际文件
4. 静态服务前缀和请求 URL 是否对应？`/uploads` vs `/upload`、有没有多余 `/`？

## 容易踩的额外坑

- **跨平台路径**：Windows 开发用 `\` 路径，Docker 容器是 Linux `/`。`r.Static` 接受 OS-native path，但 URL 总是 `/`。`filepath.ToSlash()` 用在 DB 存储 / API 返回值
- **挂载点冲突**：docker-compose 的 `./data:/data` 必须包含 uploads 子目录，或者代码里 `os.MkdirAll` 自动创建
- **权限**：alpine + PUID/PGID 模式下，`/data/uploads/*` 必须由容器内的 moon 用户拥有，否则写入失败。entrypoint.sh 里 `chown -R moon:moon /data` 已经覆盖整个 /data 树

## How to apply

新增任何"前缀路由"（`r.Static`、`r.GET("/foo/*x", ...)` 等）时：
1. 放在 NoRoute 之前**仅出于约定**
2. 真正要做的是确认**路径前缀 + 实际目录 + 权限**三件事
3. 测试时直接 `curl -I http://localhost:3000/uploads/<known-file>` 看返回 Content-Type，是 `image/*` 才对，是 `text/html` 就是踩了
