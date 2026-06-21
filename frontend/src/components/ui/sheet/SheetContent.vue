<script setup lang="ts">
import { X } from '@lucide/vue'
import {
  DialogClose,
  DialogContent,
  DialogOverlay,
  DialogPortal,
} from 'reka-ui'
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{ side?: 'left' | 'right'; class?: string }>(), { side: 'right' })
const sideClass = computed(() => props.side === 'left' ? 'left-0 border-r' : 'right-0 border-l')
</script>

<template>
  <DialogPortal>
    <DialogOverlay class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm" />
    <DialogContent
      :class="cn('fixed inset-y-0 z-50 w-[min(88vw,24rem)] border-border bg-card p-6 shadow-2xl outline-none', sideClass, props.class)"
    >
      <slot />
      <DialogClose class="absolute right-4 top-4 rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground">
        <X class="size-4" />
        <span class="sr-only">关闭</span>
      </DialogClose>
    </DialogContent>
  </DialogPortal>
</template>
