import { onUnmounted } from 'vue'

export interface LongPressEvent {
  clientX: number
  clientY: number
  preventNativeMenu: () => void
}

/**
 * Returns touch event handlers a component can spread onto a target element.
 * Triggers `callback` after `delayMs` of continuous touch on the same spot.
 * Movement > 8px or early touchend cancels.
 *
 * Mobile-only: doesn't observe mouse events (desktop uses @contextmenu).
 */
export function useLongPress(
  callback: (e: LongPressEvent) => void,
  delayMs = 500,
) {
  let timer: number | null = null
  let startX = 0
  let startY = 0
  const MOVE_TOLERANCE = 8

  function clear() {
    if (timer !== null) {
      clearTimeout(timer)
      timer = null
    }
  }

  function onTouchStart(e: TouchEvent) {
    const t = e.touches[0]
    if (!t) return
    startX = t.clientX
    startY = t.clientY
    clear()
    timer = window.setTimeout(() => {
      timer = null
      callback({
        clientX: startX,
        clientY: startY,
        preventNativeMenu: () => {
          // Best-effort prevention of native long-press menu. iOS Safari
          // requires CSS -webkit-touch-callout:none on the element to fully
          // suppress; this only stops further propagation of this event.
          e.preventDefault()
        },
      })
    }, delayMs)
  }

  function onTouchMove(e: TouchEvent) {
    const t = e.touches[0]
    if (!t) {
      clear()
      return
    }
    const dx = Math.abs(t.clientX - startX)
    const dy = Math.abs(t.clientY - startY)
    if (dx > MOVE_TOLERANCE || dy > MOVE_TOLERANCE) clear()
  }

  function onTouchEnd() {
    clear()
  }

  function onTouchCancel() {
    clear()
  }

  onUnmounted(clear)

  return { onTouchStart, onTouchMove, onTouchEnd, onTouchCancel }
}
