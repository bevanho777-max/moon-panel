import { http, type ApiResponse } from './client'

/** Single uploaded wallpaper as returned by GET /admin/wallpapers. */
export interface UploadedWallpaper {
  /** Hash without extension — stable id used as Vue :key in the picker grid. */
  hash: string
  /** Relative path under /uploads/, e.g. "public/wallpapers/abc.webp". */
  path: string
  /** Public URL ready to use as <img src>. */
  url: string
  /** Setting reference form, e.g. "upload:public/wallpapers/abc.webp". */
  wallpaper: string
  /** File size in bytes — shown in the picker for user info / cleanup hints. */
  size: number
}

export interface WallpaperUploadResult {
  path: string
  url: string
  wallpaper: string
  deduped: boolean
  size: number
  type: string
}

export async function listWallpapers(): Promise<UploadedWallpaper[]> {
  const { data } = await http.get<ApiResponse<{ items: UploadedWallpaper[] }>>('/admin/wallpapers')
  return data.data?.items ?? []
}

export async function uploadWallpaper(file: Blob): Promise<WallpaperUploadResult> {
  const form = new FormData()
  form.append('file', file)
  const { data } = await http.post<ApiResponse<WallpaperUploadResult>>(
    '/admin/wallpapers/upload',
    form,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return data.data!
}

export async function deleteWallpaper(hash: string): Promise<void> {
  await http.delete(`/admin/wallpapers/${encodeURIComponent(hash)}`)
}

/**
 * Resolve a setting-style wallpaper reference to a usable <img src> URL.
 * Returns null for null/empty/unknown — caller renders no background in that
 * case (default dark theme).
 *
 * - "builtin:night"           → "/assets/wallpapers/night.svg"
 * - "upload:public/.../X.webp" → "/uploads/public/.../X.webp"
 * - null / unknown            → null
 */
export function resolveWallpaperURL(spec: string | null | undefined): string | null {
  if (!spec) return null
  if (spec.startsWith('builtin:')) {
    return `/assets/wallpapers/${spec.slice('builtin:'.length)}.svg`
  }
  if (spec.startsWith('upload:')) {
    return `/uploads/${spec.slice('upload:'.length)}`
  }
  return null
}

/**
 * Downscale + re-encode an image File on the client to a webp Blob no larger
 * than 1920×1080. Aspect-preserving (fits inside the box). Quality 0.85 is the
 * sweet spot for photos — visually lossless, ~1/5 to 1/3 the JPEG size.
 *
 * Why client-side: keeps the Go binary free of an image library (NAS budget).
 * Browsers natively encode webp via canvas.toBlob since 2018, so this works
 * on every realistic deployment target.
 *
 * On environments that don't support webp encoding (rare — old Safari before
 * 14), we fall back to png. The server accepts both; size cap is 5 MiB.
 */
export async function compressWallpaperToWebp(file: File): Promise<Blob> {
  const TARGET_W = 1920
  const TARGET_H = 1080

  const dataUrl = await readFileAsDataURL(file)
  const img = await loadImage(dataUrl)

  // Fit inside TARGET_W × TARGET_H, preserve aspect. Don't upscale.
  const scale = Math.min(1, TARGET_W / img.naturalWidth, TARGET_H / img.naturalHeight)
  const w = Math.round(img.naturalWidth * scale)
  const h = Math.round(img.naturalHeight * scale)

  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('canvas 2d context unavailable')
  ctx.drawImage(img, 0, 0, w, h)

  const webp = await canvasToBlob(canvas, 'image/webp', 0.85)
  if (webp) return webp
  // Fallback: PNG. Always supported.
  const png = await canvasToBlob(canvas, 'image/png')
  if (!png) throw new Error('canvas.toBlob returned null')
  return png
}

function readFileAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('image decode failed'))
    img.src = src
  })
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality?: number): Promise<Blob | null> {
  return new Promise((resolve) => canvas.toBlob(resolve, type, quality))
}
