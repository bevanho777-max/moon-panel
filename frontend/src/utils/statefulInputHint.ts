// Phase 4c: one-time UX hint shown the first time the user opens an editor
// modal containing StatefulInput fields. Storage uses the project-wide
// `moon.` localStorage prefix.

import type { MessageApiInjection } from 'naive-ui/es/message/src/MessageProvider'

const STORAGE_KEY = 'moon.statefulInputHintShown'
const HINT_MESSAGE =
  '💡 提示：点击字段可以重新输入；不打字点别处会恢复原值；改完出现 ↺ 还原按钮'

/**
 * Show the StatefulInput intro hint once per browser. Subsequent calls in
 * the same browser are no-ops (a localStorage flag is set on first call).
 *
 * Call from openCreate/openEdit functions in editors that contain
 * StatefulInput fields. The 6-second NaiveUI message is long enough for
 * a careful read but doesn't block interaction.
 */
export function showStatefulInputHintOnce(message: MessageApiInjection): void {
  try {
    if (localStorage.getItem(STORAGE_KEY) === 'true') return
    localStorage.setItem(STORAGE_KEY, 'true')
  } catch {
    // localStorage unavailable (private mode, etc.) — show every time.
    // Better than crashing the editor open.
  }
  message.info(HINT_MESSAGE, { duration: 6000, closable: true })
}
