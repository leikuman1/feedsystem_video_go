<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import { Flame, RefreshCw, Trophy } from '@lucide/vue'

import { ApiError } from '@/api/client'
import * as feedApi from '@/api/feed'
import * as likeApi from '@/api/like'
import type { FeedVideoItem } from '@/api/types'
import AppShell from '@/components/AppShell.vue'
import FeedVideoCard from '@/components/FeedVideoCard.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { useToastStore } from '@/stores/toast'

const toast = useToastStore()
const state = reactive({
  loading: false,
  error: '',
  items: [] as FeedVideoItem[],
  hasMore: false,
  limit: 10,
  asOf: 0,
  nextOffset: 0,
})
const likeBusy = reactive<Record<string, boolean>>({})

async function loadHot(reset: boolean) {
  if (state.loading) return
  state.loading = true
  state.error = ''
  try {
    const response = await feedApi.listByPopularity({
      limit: state.limit,
      as_of: reset ? 0 : state.asOf,
      offset: reset ? 0 : state.nextOffset,
    })
    state.hasMore = response.has_more
    state.asOf = response.as_of
    state.nextOffset = response.next_offset
    state.items = reset ? response.video_list : state.items.concat(response.video_list)
  } catch (error) {
    state.error = error instanceof ApiError ? error.message : String(error)
  } finally {
    state.loading = false
  }
}

async function toggleLike(item: FeedVideoItem) {
  const key = String(item.id)
  if (likeBusy[key]) return
  likeBusy[key] = true
  try {
    if (item.is_liked) await likeApi.unlike(item.id)
    else await likeApi.like(item.id)
    item.is_liked = !item.is_liked
    item.likes_count = Math.max(0, item.likes_count + (item.is_liked ? 1 : -1))
  } catch (error) {
    toast.error(error instanceof ApiError ? error.message : String(error))
  } finally {
    likeBusy[key] = false
  }
}

onMounted(() => void loadHot(true))
</script>

<template>
  <AppShell>
    <Card class="mx-auto max-w-5xl">
      <CardHeader>
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <span class="mb-3 grid size-11 place-items-center rounded-xl bg-primary/15 text-primary"><Flame class="size-5" /></span>
            <CardTitle class="text-2xl">实时热榜</CardTitle>
            <CardDescription class="mt-2">Redis 分钟窗口聚合热度，使用快照游标保持分页稳定。</CardDescription>
          </div>
          <div class="flex items-center gap-2">
            <Input v-model.number="state.limit" type="number" min="1" max="50" class="w-20" :disabled="state.loading" />
            <Button variant="outline" :disabled="state.loading" @click="loadHot(true)">
              <RefreshCw class="size-4" />
              刷新
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div v-if="state.error" class="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">{{ state.error }}</div>
        <div v-else-if="state.loading && state.items.length === 0" class="py-16 text-center text-sm text-muted-foreground">正在计算热榜…</div>
        <div v-else-if="state.items.length === 0" class="py-16 text-center text-sm text-muted-foreground">暂无热视频</div>

        <div v-else class="grid gap-4">
          <div v-for="(item, index) in state.items" :key="item.id" class="grid gap-3 sm:grid-cols-[3.5rem_1fr]">
            <div
              class="grid size-12 place-items-center rounded-xl border border-border bg-background font-mono font-semibold"
              :class="index < 3 ? 'border-primary/40 bg-primary/10 text-primary' : 'text-muted-foreground'"
            >
              <Trophy v-if="index < 3" class="size-4" />
              <span>{{ index + 1 }}</span>
            </div>
            <FeedVideoCard :item="item" :can-like="true" :busy="!!likeBusy[String(item.id)]" @toggle-like="toggleLike" />
          </div>
          <Button v-if="state.hasMore" variant="outline" :disabled="state.loading" class="mt-2" @click="loadHot(false)">
            {{ state.loading ? '加载中…' : '加载更多' }}
          </Button>
        </div>
      </CardContent>
    </Card>
  </AppShell>
</template>
