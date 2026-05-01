---
name: 国内网络镜像清单（Docker 构建管线）
description: GOPROXY、Alpine apk、gcr.io 三类镜像源在 Dockerfile 的位置，国外用户切回官方源的方法
type: feedback
---

项目目标用户主要在中国大陆，Docker 构建管线里所有"从境外仓库拉东西"的步骤都必须配镜像源，否则单步耗时从秒级飙到几十分钟（甚至超时失败）。

## 当前已配置的镜像（截至 2026-04-28）

### 1. Go module proxy
- **位置**: [Dockerfile](../Dockerfile) Stage 2 (golang:1.23-alpine 后)
- **配置**: `ENV GOPROXY=https://goproxy.cn,direct`
- **官方源**: `proxy.golang.org`（国内不通）
- **国外切回**: 注释掉 `ENV GOPROXY=...` 那行，Go 默认就走 proxy.golang.org

### 2. Alpine apk repositories
- **位置**: [Dockerfile](../Dockerfile) Stage 3 (alpine:3.20 后，apk add 前)
- **配置**: 
  ```
  RUN sed -i 's|dl-cdn.alpinelinux.org|mirrors.tuna.tsinghua.edu.cn|g' \
      /etc/apk/repositories
  ```
- **官方源**: `dl-cdn.alpinelinux.org`（国内实测 apk add 卡 47 分钟才完成）
- **国外切回**: 注释掉那条 `RUN sed` 行
- **可选替代镜像**:
  - `mirrors.aliyun.com/alpine`（阿里）
  - `mirrors.nju.edu.cn/alpine`（南大）
  - `mirrors.tuna.tsinghua.edu.cn/alpine`（清华，**当前使用**）

### 3. (历史) gcr.io distroless 镜像
- **位置**: 之前的 Stage 3（distroless 时代），现在已切到 alpine 不再用
- **如果未来切回 distroless**: `FROM gcr.nju.edu.cn/distroless/static-debian12:nonroot`
- **官方源**: `gcr.io/distroless/static-debian12:nonroot`（Google Container Registry，国内不通）

## 还没碰但可能要配的

### npm registry
- **当前**: 未配置，用 npm 默认 `registry.npmjs.org`
- **何时考虑**: 如果未来 npm install 单步耗时 > 5 分钟、或交付清单出现"npm install 卡死"反馈
- **加法**: Stage 1 (node:20-alpine) `npm install` 前加 `RUN npm config set registry https://registry.npmmirror.com`
- **现状**: 76 包 install 实测 30 秒，**没必要配**

## 诊断 pattern：怎么判断是不是镜像问题

某一步异常慢（>5 分钟），先看它在拉什么：
- `go: downloading ... proxy.golang.org` → GOPROXY 没生效
- `apk: ... dl-cdn.alpinelinux.org` → Alpine apk 镜像没配
- `Step X/Y : FROM gcr.io/...` → gcr.io 镜像没换
- `npm http GET https://registry.npmjs.org/...` 半天没响应 → npm 镜像可能要配

中国用户报告"build 卡了几十分钟最后成功了"——这个**最隐蔽**，因为最终成功了不会爆错，只是速度问题。我（Claude）很容易看不到这层延迟，依赖用户反馈才能发现。

## How to apply

- 加 / 改 Dockerfile 任何拉境外资源的步骤前，**主动检查**当前是否已配镜像源
- 新加的步骤（比如 stage 多一个 base image）涉及国外仓库，提前加镜像
- 每条镜像配置都要带"国外用户切回官方源"的注释
- 用户报告慢 build 时第一反应：定位是哪个仓库慢、加镜像源
- 镜像加完后**通过实测耗时验证**（让用户给 build 时间数字），不要靠猜
