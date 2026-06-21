<script setup lang="ts">
import type { HTMLAttributes, InputHTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })
defineProps<{ class?: HTMLAttributes['class']; type?: InputHTMLAttributes['type'] }>()
const model = defineModel<string | number>()
</script>

<template>
  <input
    v-bind="$attrs"
    v-model="model"
    :type="$props.type"
    :class="cn('flex h-10 w-full rounded-lg border border-input bg-background/60 px-3 py-2 text-sm outline-none placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50', $props.class)"
  />
</template>
