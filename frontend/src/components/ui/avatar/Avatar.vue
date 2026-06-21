<script setup lang="ts">
import { computed } from 'vue'
import { AvatarFallback, AvatarImage, AvatarRoot } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  src?: string
  alt?: string
  fallback: string
  class?: string
}>()

const initials = computed(() => props.fallback.trim().slice(0, 2).toUpperCase())
</script>

<template>
  <AvatarRoot :class="cn('relative flex size-10 shrink-0 overflow-hidden rounded-full border border-border bg-muted', props.class)">
    <AvatarImage v-if="props.src" :src="props.src" :alt="props.alt ?? props.fallback" class="aspect-square size-full object-cover" />
    <AvatarFallback class="grid size-full place-items-center text-xs font-semibold text-muted-foreground">
      {{ initials }}
    </AvatarFallback>
  </AvatarRoot>
</template>
