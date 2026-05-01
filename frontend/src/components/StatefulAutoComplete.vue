<script setup lang="ts">
// StatefulAutoComplete — Phase 4c
//
// Same A/B/C/D state machine as StatefulInput, but wrapping NAutoComplete
// for fields that combine free-text input with suggestions. The icon field
// in Cards.vue (jellyfin → dashboard-icons suggestions, lucide:* → Lucide
// suggestions) is the primary use case. Behavior contract is identical to
// StatefulInput; visual layout adds a dropdown of options.
//
// Differences from StatefulInput:
//   - NAutoComplete owns its own dropdown (we pass through `options` and
//     `render-label`); we don't try to reproduce that.
//   - When the user picks a suggestion via @select, that's a commit just
//     like typing — hasCommitted=true, modelValue updates.
//   - Revert button rendered next to the input via NInputGroup since
//     NAutoComplete doesn't expose a #suffix slot identical to NInput.
//     We position the button as a sibling so click semantics stay clean.

import { computed, ref } from 'vue'
import { NAutoComplete, NButton } from 'naive-ui'

interface Props {
  modelValue: string
  originalValue: string
  options?: unknown[]
  renderLabel?: (option: unknown) => unknown
  placeholder?: string
  disabled?: boolean
  loading?: boolean
}
const props = withDefaults(defineProps<Props>(), {
  options: () => [],
  placeholder: '',
  disabled: false,
  loading: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
  (e: 'select', v: string | number): void
  (e: 'focus'): void
  (e: 'blur'): void
}>()

const focused = ref(false)
const hasCommitted = ref(props.modelValue !== props.originalValue)

type Stage = 'A' | 'B' | 'C' | 'D'
const stage = computed<Stage>(() => {
  if (focused.value) {
    if (!hasCommitted.value) {
      return props.originalValue !== '' ? 'B' : 'C'
    }
    return 'C'
  }
  if (props.modelValue !== props.originalValue) {
    return 'D'
  }
  return 'A'
})

const displayValue = computed<string>({
  get() {
    return stage.value === 'B' ? '' : props.modelValue
  },
  set(v: string) {
    hasCommitted.value = true
    emit('update:modelValue', v)
  },
})

const effectivePlaceholder = computed(() =>
  stage.value === 'B' ? props.originalValue : props.placeholder,
)

function onFocus() {
  focused.value = true
  emit('focus')
}
function onBlur() {
  focused.value = false
  if (!hasCommitted.value && props.modelValue !== props.originalValue) {
    emit('update:modelValue', props.originalValue)
  }
  emit('blur')
}
function onSelect(v: string | number) {
  // Selecting a suggestion is a commit. NAutoComplete sets value automatically;
  // the displayValue setter will run with the suggestion's value.
  hasCommitted.value = true
  emit('select', v)
}

function revert() {
  hasCommitted.value = false
  emit('update:modelValue', props.originalValue)
}

const showRevert = computed(
  () => stage.value === 'D' && props.modelValue !== props.originalValue,
)
const emptyAfterCommit = computed(
  () => stage.value === 'D' && props.modelValue === '',
)
</script>

<template>
  <div class="sac" :data-stage="stage">
    <NAutoComplete
      :value="displayValue"
      :options="options as any"
      :render-label="renderLabel as any"
      :placeholder="effectivePlaceholder"
      :disabled="disabled"
      :loading="loading"
      :clear-after-select="false"
      @update:value="displayValue = $event"
      @focus="onFocus"
      @blur="onBlur"
      @select="onSelect"
    />
    <NButton
      v-if="showRevert"
      class="sac__revert"
      :class="{ 'sac__revert--highlight': emptyAfterCommit }"
      size="tiny"
      tertiary
      title="还原原值"
      @click.stop="revert"
      @mousedown.stop
    >
      ↺
    </NButton>
  </div>
</template>

<style scoped>
.sac {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
}
.sac > :deep(.n-auto-complete) {
  flex: 1;
}
.sac__revert {
  flex-shrink: 0;
  width: 28px;
  font-size: 0.95rem;
}
.sac__revert--highlight {
  --n-color: rgba(224, 184, 90, 0.18) !important;
  --n-color-hover: rgba(224, 184, 90, 0.3) !important;
  --n-text-color: #e0b85a !important;
  animation: sac-pulse 1.4s ease-in-out infinite;
}
@keyframes sac-pulse {
  0%, 100% { opacity: 0.85; }
  50%      { opacity: 1; }
}
</style>
