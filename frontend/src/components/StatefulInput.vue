<script setup lang="ts">
// StatefulInput — Phase 4b
//
// Wraps NInput with a 4-state UX: A (idle, shows original) → B (open, cleared
// with placeholder=original) → C (editing, has committed) → D (committed,
// shows revert button). Replaces the default "click field, original value
// auto-selected, hit something to delete" experience that plagues many
// admin tools.
//
// Contract:
//   v-model="form.title"            - committed value the parent saves on submit
//   :original-value="orig.title"    - the value at the moment the form opened
//                                     (parent should snapshot this in a
//                                     separate "original" object on form open)
//   The component derives current state from (v-model, originalValue, focus,
//   hasCommitted) — NO state stored in parent beyond modelValue.
//
// State transitions (also see project_phase4_stateful_input_spec.md):
//   - focus + originalValue empty → C (normal editing)
//   - focus + originalValue non-empty + !committed → B (display cleared, placeholder = original)
//   - any keystroke / clear-button click → hasCommitted = true → C (while focused)
//   - blur + !committed → emit originalValue → A
//   - blur + committed → D (input shows committed value, revert button visible)
//   - revert button click → emit originalValue + hasCommitted=false → A
//
// Edge cases handled:
//   - originalValue empty: no B state (focus → C directly), placeholder is the user-supplied prop
//   - D state with empty value (user typed then cleared): revert button rendered HIGHLIGHTED
//     so user understands "this is committed empty, not auto-restored"
//   - clearable X click: hasCommitted=true, currentValue='', stays in current focus state
//   - Repeat clicks on same field: stays in B (don't reset hasCommitted)
//   - modelValue ≠ originalValue at mount: hasCommitted starts true (treats as pre-committed D)
//
// Not done in 4b (deferred to backlog):
//   - NAutoComplete fields (e.g. card icon picker) — needs a parallel component
//   - First-use UX hint toast ("点击即可重新输入...")

import { computed, ref } from 'vue'
import { NInput } from 'naive-ui'

interface Props {
  modelValue: string
  originalValue: string
  placeholder?: string
  disabled?: boolean
  type?: 'text' | 'password' | 'textarea'
  showPasswordOn?: 'click' | 'mousedown'
  maxlength?: number
  // Disable the component's wrapping; pass-through enables for testing flexibility.
  /** Disable clearable X; default true (X is shown in C/D states). */
  clearable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '',
  disabled: false,
  type: 'text',
  showPasswordOn: 'click',
  clearable: true,
})

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
}>()

const focused = ref(false)
// hasCommitted starts true if parent injected a non-original modelValue
// (rare, but supports forms that pre-fill changed values for the user).
const hasCommitted = ref(props.modelValue !== props.originalValue)

type Stage = 'A' | 'B' | 'C' | 'D'
const stage = computed<Stage>(() => {
  if (focused.value) {
    if (!hasCommitted.value) {
      return props.originalValue !== '' ? 'B' : 'C'
    }
    return 'C'
  }
  // Not focused: D iff value diverged from original. If user touched the
  // field but value matches original (typed then deleted same chars, or
  // originalValue was empty all along), treat as A — no revert button
  // needed when there's nothing to revert.
  if (props.modelValue !== props.originalValue) {
    return 'D'
  }
  return 'A'
})

// What NInput shows:
//   - In B: empty (placeholder fills with originalValue)
//   - Otherwise: modelValue
const displayValue = computed<string>({
  get() {
    return stage.value === 'B' ? '' : props.modelValue
  },
  set(v: string) {
    // Every user-driven update commits. Programmatic emits (blur restore,
    // revert) bypass this setter by emitting directly to the parent.
    hasCommitted.value = true
    emit('update:modelValue', v)
  },
})

const effectivePlaceholder = computed(() =>
  stage.value === 'B' ? props.originalValue : props.placeholder,
)

function onFocus() {
  focused.value = true
}

function onBlur() {
  focused.value = false
  if (!hasCommitted.value) {
    // User entered B but didn't commit — restore to original value.
    // The emit ensures the parent sees the restored value too (matters
    // when modelValue was somehow != originalValue at focus time, e.g.,
    // external reset during the brief focus window).
    if (props.modelValue !== props.originalValue) {
      emit('update:modelValue', props.originalValue)
    }
  }
}

function revert() {
  hasCommitted.value = false
  emit('update:modelValue', props.originalValue)
  // Defer focus loss so the click is fully processed; don't force blur
  // — natural focus loss happens because the click was outside the input
  // proper. State recomputes to A on next tick.
}

const showRevert = computed(
  () => stage.value === 'D' && props.modelValue !== props.originalValue,
)
const emptyAfterCommit = computed(
  () => stage.value === 'D' && props.modelValue === '',
)
</script>

<template>
  <div class="si" :data-stage="stage">
    <NInput
      :value="displayValue"
      :placeholder="effectivePlaceholder"
      :disabled="disabled"
      :type="type"
      :show-password-on="showPasswordOn"
      :maxlength="maxlength"
      :clearable="clearable && stage !== 'B'"
      @update:value="displayValue = $event"
      @focus="onFocus"
      @blur="onBlur"
    >
      <template #suffix v-if="showRevert">
        <span
          class="si__revert"
          :class="{ 'si__revert--highlight': emptyAfterCommit }"
          role="button"
          tabindex="-1"
          title="还原原值"
          @click.stop="revert"
          @mousedown.stop
        >
          ↺
        </span>
      </template>
    </NInput>
  </div>
</template>

<style scoped>
.si {
  width: 100%;
}
.si__revert {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  font-size: 0.95rem;
  cursor: pointer;
  color: rgba(255, 255, 255, 0.55);
  border-radius: 3px;
  user-select: none;
  transition: color 120ms ease, background 120ms ease;
}
.si__revert:hover {
  color: rgba(255, 255, 255, 0.9);
  background: rgba(255, 255, 255, 0.08);
}
/* Empty-after-commit state — make the revert call to action obvious so users
   don't mistake "I deleted everything and clicked away" for "auto-restored". */
.si__revert--highlight {
  color: #e0b85a;
  background: rgba(224, 184, 90, 0.12);
  animation: si-pulse 1.4s ease-in-out infinite;
}
.si__revert--highlight:hover {
  color: #f0c870;
  background: rgba(224, 184, 90, 0.22);
}
@keyframes si-pulse {
  0%, 100% { background: rgba(224, 184, 90, 0.12); }
  50%      { background: rgba(224, 184, 90, 0.28); }
}
</style>
