<script setup lang="ts">
import { TabsContent } from 'reka-ui'
import { cn } from '@/lib/utils'

defineProps<{ value: string; class?: string }>()
</script>

<template>
  <TabsContent :value="$props.value" :class="cn('mt-4 outline-none', $props.class)">
    <slot />
  </TabsContent>
</template>
