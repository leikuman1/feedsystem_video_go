<script setup lang="ts">
import { ProgressIndicator, ProgressRoot } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{ modelValue?: number; class?: string }>(), { modelValue: 0 })
</script>

<template>
  <ProgressRoot
    :model-value="Math.max(0, Math.min(100, props.modelValue))"
    :class="cn('relative h-2 w-full overflow-hidden rounded-full bg-secondary', props.class)"
  >
    <ProgressIndicator
      class="h-full w-full bg-primary transition-transform duration-300"
      :style="{ transform: `translateX(-${100 - Math.max(0, Math.min(100, props.modelValue))}%)` }"
    />
  </ProgressRoot>
</template>
