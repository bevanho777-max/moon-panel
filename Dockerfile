# syntax=docker/dockerfile:1.7

# ---- Stage 1: build frontend ----
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --no-audit --no-fund
COPY frontend/ ./
RUN npm run build
# Output lands at ../backend/web/dist per vite.config.ts; promote it to a known path.
RUN mkdir -p /out && cp -r ../backend/web/dist /out/dist

# ---- Stage 2: build backend ----
# Multi-arch: pin builder to native BUILDPLATFORM and cross-compile via Go's
# GOOS/GOARCH instead of running an emulated arm64 toolchain under QEMU.
# Because modernc.org/sqlite is pure Go (no CGO), cross-compilation is just
# `GOARCH=arm64 go build` — runs at native amd64 speed targeting any arch.
# Without --platform=$BUILDPLATFORM, buildx would launch an emulated arm64
# golang container and a single CI build would balloon from ~30s to ~10min.
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS backend
ARG TARGETOS
ARG TARGETARCH
# v0.1.2: build-time version metadata, injected into the binary via
# -ldflags -X overrides. release.yml fills these from the pushed git tag,
# the workflow run timestamp, and the commit SHA. Local `docker build`
# without --build-arg keeps the dev defaults defined in api/version.go.
ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG COMMIT=unknown
WORKDIR /src
# GOPROXY: try CN mirror first for fast local builds; fall through to default.
# CI runners outside CN reach proxy.golang.org via the `direct` fallback in
# milliseconds, so this remains universal.
ENV GOPROXY=https://goproxy.cn,direct
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /out/dist ./web/dist
# go mod tidy resolves any new imports added since the cached layer (e.g. when
# Phase 3d-3 adds pquerna/otp without a pre-computed go.sum entry). The cached
# `go mod download` above still warms most of the module graph; tidy fills in
# the missing entries — fast in steady state, correct after any dep bump.
RUN go mod tidy
ENV CGO_ENABLED=0
# v0.1.2: -X overrides inject the build-time version vars declared in
# internal/api/version.go. Single long line for the -ldflags value: nesting
# Dockerfile line continuations (\) inside the quoted string confuses the
# parser into treating the next line as a new instruction.
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath \
    -ldflags "-s -w -X github.com/moon-panel/moon-panel/internal/api.Version=${VERSION} -X github.com/moon-panel/moon-panel/internal/api.BuildDate=${BUILD_DATE} -X github.com/moon-panel/moon-panel/internal/api.Commit=${COMMIT}" \
    -o /out/moon-panel ./cmd/server

# ---- Stage 3: alpine runtime with PUID/PGID support (LinuxServer.io style) ----
# Why alpine instead of distroless: NAS users (Synology, Unraid, TrueNAS) expect
# PUID/PGID env vars so the container's data files end up owned by the host user.
# distroless has no shell, so it can't chown the data volume at startup —
# users would have to manually `chown -R 65532:65532 ./data` for every deployment.
# Trade: ~25MB vs ~17MB. On a NAS that's noise.
FROM alpine:3.20
# Optional Alpine mirror override for slow networks. Default is the official
# dl-cdn.alpinelinux.org (universal, works in CI). CN builders override locally:
#   docker build --build-arg ALPINE_MIRROR=mirrors.tuna.tsinghua.edu.cn .
# Empty default means the sed below is a harmless identity rewrite.
ARG ALPINE_MIRROR=dl-cdn.alpinelinux.org
RUN sed -i "s|dl-cdn.alpinelinux.org|${ALPINE_MIRROR}|g" /etc/apk/repositories \
 && apk add --no-cache su-exec ca-certificates tzdata
WORKDIR /app
COPY --from=backend /out/moon-panel /app/moon-panel
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENV MOON_DATA_DIR=/data \
    MOON_PORT=3000 \
    MOON_PUBLIC_MODE=true \
    MOON_ENV=production \
    PUID=1000 \
    PGID=1000
EXPOSE 3000
VOLUME ["/data"]
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/app/moon-panel"]
