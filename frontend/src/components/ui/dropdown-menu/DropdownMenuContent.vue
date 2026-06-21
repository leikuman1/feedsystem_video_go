<script setup lang="ts">
import { DropdownMenuContent, DropdownMenuPortal } from 'reka-ui'
import { cn } from '@/lib/utils'

defineProps<{ class?: string; sideOffset?: number }>()
</script>

<template>
  <DropdownMenuPortal>
    <DropdownMenuContent
      :side-offset="$props.sideOffset ?? 8"
      :class="cn('z-50 min-w-40 overflow-hidden rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-xl outline-none', $props.class)"
    >
      <slot />
    </DropdownMenuContent>
  </DropdownMenuPortal>
</template>
