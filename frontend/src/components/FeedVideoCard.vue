<script setup lang="ts">
import { Heart, Play } from '@lucide/vue'
import type { FeedVideoItem } from '@/api/types'
import { Button } from '@/components/ui/button'

defineProps<{ item: FeedVideoItem; canLike: boolean; busy?: boolean }>()
const emit = defineEmits<{ (event: 'toggle-like', item: FeedVideoItem): void }>()
</script>

<template>
  <article class="group grid overflow-hidden rounded-xl border border-border bg-card md:grid-cols-[13rem_1fr]">
    <RouterLink :to="`/video/${item.id}`" class="relative aspect-video overflow-hidden bg-black md:aspect-auto">
      <img :src="item.cover_url" :alt="item.title" class="size-full object-cover transition duration-300 group-hover:scale-105" loading="lazy" />
      <span class="absolute inset-0 grid place-items-center bg-black/15 opacity-0 transition group-hover:opacity-100">
        <span class="grid size-11 place-items-center rounded-full bg-black/65 text-white"><Play class="size-5 fill-current" /></span>
      </span>
    </RouterLink>
    <div class="flex min-w-0 flex-col justify-between gap-4 p-4">
      <div>
        <RouterLink :to="`/video/${item.id}`" class="line-clamp-2 font-semibold hover:text-primary">{{ item.title }}</RouterLink>
        <p class="mt-2 text-sm text-muted-foreground">@{{ item.author.username }}</p>
        <p v-if="item.description" class="mt-3 line-clamp-2 text-sm leading-6 text-foreground/70">{{ item.description }}</p>
      </div>
      <div class="flex items-center justify-between gap-3">
        <span class="text-xs text-muted-foreground">{{ new Date(item.create_time * 1000).toLocaleDateString() }}</span>
        <Button
          v-if="canLike"
          :variant="item.is_liked ? 'default' : 'outline'"
          size="sm"
          :disabled="busy"
          @click="emit('toggle-like', item)"
        >
          <Heart class="size-4" :class="item.is_liked ? 'fill-current' : ''" />
          {{ item.likes_count }}
        </Button>
      </div>
    </div>
  </article>
</template>
