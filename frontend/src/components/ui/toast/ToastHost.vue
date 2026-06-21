<script setup lang="ts">
import { computed } from 'vue'
import { CircleCheck, CircleX, Info } from '@lucide/vue'
import { useToastStore } from '@/stores/toast'

const toast = useToastStore()
const items = computed(() => toast.toasts)
</script>

<template>
  <div class="pointer-events-none fixed inset-x-4 top-4 z-[100] grid justify-items-end gap-2">
    <div
      v-for="item in items"
      :key="item.id"
      class="pointer-events-auto flex min-w-72 max-w-md items-start gap-3 rounded-xl border border-border bg-popover/95 p-4 text-sm shadow-2xl backdrop-blur"
    >
      <CircleCheck v-if="item.type === 'success'" class="mt-0.5 size-4 text-emerald-400" />
      <CircleX v-else-if="item.type === 'error'" class="mt-0.5 size-4 text-destructive" />
      <Info v-else class="mt-0.5 size-4 text-primary" />
      <span class="text-popover-foreground">{{ item.message }}</span>
    </div>
  </div>
</template>
