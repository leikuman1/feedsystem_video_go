<script setup lang="ts">
import { X } from '@lucide/vue'
import {
  DialogClose,
  DialogContent as DialogContentPrimitive,
  DialogOverlay,
  DialogPortal,
} from 'reka-ui'
import { cn } from '@/lib/utils'

defineProps<{ class?: string }>()
</script>

<template>
  <DialogPortal>
    <DialogOverlay class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out" />
    <DialogContentPrimitive
      :class="cn('fixed left-1/2 top-1/2 z-50 grid w-[min(92vw,34rem)] -translate-x-1/2 -translate-y-1/2 gap-4 rounded-xl border border-border bg-card p-6 shadow-2xl outline-none', $props.class)"
    >
      <slot />
      <DialogClose class="absolute right-4 top-4 rounded-md p-1 text-muted-foreground transition hover:bg-accent hover:text-foreground">
        <X class="size-4" />
        <span class="sr-only">关闭</span>
      </DialogClose>
    </DialogContentPrimitive>
  </DialogPortal>
</template>
