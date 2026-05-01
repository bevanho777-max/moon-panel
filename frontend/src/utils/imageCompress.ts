// Client-side image compression for icon uploads.
// Resizes to max 256x256 (preserving aspect ratio) and re-encodes as WebP at
// quality 0.85. Falls back to PNG if the browser can't produce WebP.
// GIFs are passed through unchanged (canvas only captures the first frame —
// preserving animation matters more than file size for emoji-style icons).

export interface CompressOptions {
  maxDimension?: number
  quality?: number
}

export interface CompressResult {
  blob: Blob
  filename: string
  mime: string
  originalSize: number
  compressedSize: number
}

const DEFAULT_MAX = 256
const DEFAULT_QUALITY = 0.85

export async function compressImage(
  file: File,
  opts: CompressOptions = {},
): Promise<CompressResult> {
  const maxDim = opts.maxDimension ?? DEFAULT_MAX
  const quality = opts.quality ?? DEFAULT_QUALITY

  // GIF: skip compression to preserve animation
  if (file.type === 'image/gif') {
    return {
      blob: file,
      filename: 'icon.gif',
      mime: 'image/gif',
      originalSize: file.size,
      compressedSize: file.size,
    }
  }

  if (!file.type.startsWith('image/')) {
    throw new Error(`不支持的文件类型: ${file.type}`)
  }

  const img = await loadImage(file)

  const scale = Math.min(1, maxDim / Math.max(img.width, img.height))
  const targetW = Math.max(1, Math.round(img.width * scale))
  const targetH = Math.max(1, Math.round(img.height * scale))

  const canvas = document.createElement('canvas')
  canvas.width = targetW
  canvas.height = targetH
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('canvas 2d context unavailable')
  ctx.drawImage(img, 0, 0, targetW, targetH)

  // Try WebP first
  const webpBlob = await canvasToBlob(canvas, 'image/webp', quality)
  if (webpBlob && webpBlob.type === 'image/webp') {
    return {
      blob: webpBlob,
      filename: 'icon.webp',
      mime: 'image/webp',
      originalSize: file.size,
      compressedSize: webpBlob.size,
    }
  }

  // Fallback to PNG (universally supported)
  const pngBlob = await canvasToBlob(canvas, 'image/png')
  if (!pngBlob) throw new Error('canvas encode failed (both WebP and PNG)')
  return {
    blob: pngBlob,
    filename: 'icon.png',
    mime: 'image/png',
    originalSize: file.size,
    compressedSize: pngBlob.size,
  }
}

function loadImage(file: File): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const img = new Image()
    img.onload = () => {
      URL.revokeObjectURL(url)
      resolve(img)
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('图片解码失败'))
    }
    img.src = url
  })
}

function canvasToBlob(
  canvas: HTMLCanvasElement,
  type: string,
  quality?: number,
): Promise<Blob | null> {
  return new Promise((resolve) => {
    canvas.toBlob(resolve, type, quality)
  })
}
