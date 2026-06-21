<script setup lang="ts">
import { TabsTrigger } from 'reka-ui'
import { cn } from '@/lib/utils'

defineProps<{ value: string; class?: string }>()
</script>

<template>
  <TabsTrigger
    :value="$props.value"
    :class="cn('inline-flex items-center justify-center rounded-md px-3 py-1.5 text-sm font-medium outline-none transition data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm', $props.class)"
  >
    <slot />
  </TabsTrigger>
</template>
