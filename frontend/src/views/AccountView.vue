<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Heart, Settings, Users, Video as VideoIcon, X } from '@lucide/vue'

import { ApiError } from '@/api/client'
import * as accountApi from '@/api/account'
import * as likeApi from '@/api/like'
import type { Account, Video } from '@/api/types'
import * as videoApi from '@/api/video'
import AppShell from '@/components/AppShell.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useAuthStore } from '@/stores/auth'
import { useSocialStore } from '@/stores/social'

const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const accountId = computed(() => auth.claims?.account_id ?? 0)
const profile = ref<Account | null>(null)
const tab = ref('works')
const listOpen = ref(false)
const listType = ref<'followers' | 'following'>('followers')

const state = reactive({
  loading: false,
  error: '',
  works: [] as Video[],
  likes: [] as Video[],
})

const listItems = computed(() => listType.value === 'followers' ? social.followers : social.vloggers)
const listTitle = computed(() => listType.value === 'followers' ? '粉丝' : '关注')

async function load() {
  if (!accountId.value) return
  state.loading = true
  state.error = ''
  try {
    const [account, works, likes] = await Promise.all([
      accountApi.findById(accountId.value),
      videoApi.listByAuthorId(accountId.value),
      likeApi.listMyLikedVideos(),
      social.refreshMine(),
    ])
    profile.value = account
    state.works = works
    state.likes = likes
  } catch (error) {
    state.error = error instanceof ApiError ? error.message : String(error)
  } finally {
    state.loading = false
  }
}

function openList(type: 'followers' | 'following') {
  listType.value = type
  listOpen.value = true
}

onMounted(load)
</script>

<template>
  <AppShell>
    <div class="mx-auto grid max-w-6xl gap-6">
      <Card>
        <CardContent class="flex flex-col gap-6 p-6 md:flex-row md:items-center md:justify-between">
          <div class="flex items-center gap-4">
            <UserAvatar
              :username="profile?.username ?? auth.claims?.username ?? 'User'"
              :id="accountId"
              :src="profile?.avatar_url"
              :size="76"
            />
            <div>
              <h1 class="text-2xl font-semibold">@{{ profile?.username ?? auth.claims?.username }}</h1>
              <p class="mt-1 font-mono text-xs text-muted-foreground">ACCOUNT #{{ accountId }}</p>
              <p v-if="profile?.bio" class="mt-3 max-w-xl text-sm leading-6 text-foreground/70">{{ profile.bio }}</p>
            </div>
          </div>
          <Button variant="outline" @click="router.push('/settings')">
            <Settings class="size-4" />
            编辑资料
          </Button>
        </CardContent>
      </Card>

      <div class="grid gap-4 sm:grid-cols-3">
        <button class="metric-card" @click="openList('followers')">
          <Users class="size-5 text-primary" />
          <span class="text-2xl font-semibold">{{ social.followerCount }}</span>
          <span class="text-xs text-muted-foreground">粉丝</span>
        </button>
        <button class="metric-card" @click="openList('following')">
          <Users class="size-5 text-primary" />
          <span class="text-2xl font-semibold">{{ social.followingCount }}</span>
          <span class="text-xs text-muted-foreground">关注</span>
        </button>
        <div class="metric-card">
          <VideoIcon class="size-5 text-primary" />
          <span class="text-2xl font-semibold">{{ state.works.length }}</span>
          <span class="text-xs text-muted-foreground">作品</span>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>内容收藏</CardTitle>
          <CardDescription>查看自己发布和点赞过的视频。</CardDescription>
        </CardHeader>
        <CardContent>
          <div v-if="state.error" class="mb-4 rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">{{ state.error }}</div>
          <Tabs v-model="tab">
            <TabsList>
              <TabsTrigger value="works"><VideoIcon class="mr-2 size-4" />作品</TabsTrigger>
              <TabsTrigger value="likes"><Heart class="mr-2 size-4" />点赞</TabsTrigger>
            </TabsList>
            <TabsContent value="works">
              <div v-if="state.loading" class="py-12 text-center text-sm text-muted-foreground">加载中…</div>
              <div v-else-if="state.works.length === 0" class="py-12 text-center text-sm text-muted-foreground">尚未发布视频</div>
              <div v-else class="video-grid">
                <button v-for="video in state.works" :key="video.id" class="video-tile" @click="router.push(`/video/${video.id}`)">
                  <img :src="video.cover_url" :alt="video.title" />
                  <span>{{ video.title }}</span>
                </button>
              </div>
            </TabsContent>
            <TabsContent value="likes">
              <div v-if="state.loading" class="py-12 text-center text-sm text-muted-foreground">加载中…</div>
              <div v-else-if="state.likes.length === 0" class="py-12 text-center text-sm text-muted-foreground">尚未点赞视频</div>
              <div v-else class="video-grid">
                <button v-for="video in state.likes" :key="video.id" class="video-tile" @click="router.push(`/video/${video.id}`)">
                  <img :src="video.cover_url" :alt="video.title" />
                  <span>{{ video.title }}</span>
                </button>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>

    <div v-if="listOpen" class="fixed inset-0 z-50 grid place-items-center bg-black/70 p-4 backdrop-blur-sm" @click.self="listOpen = false">
      <Card class="max-h-[75vh] w-full max-w-lg overflow-hidden">
        <CardHeader class="flex-row items-center justify-between">
          <CardTitle>{{ listTitle }}</CardTitle>
          <Button variant="ghost" size="icon" @click="listOpen = false"><X class="size-4" /></Button>
        </CardHeader>
        <CardContent class="grid max-h-[60vh] gap-2 overflow-y-auto">
          <p v-if="listItems.length === 0" class="py-10 text-center text-sm text-muted-foreground">暂无数据</p>
          <button
            v-for="user in listItems"
            :key="user.id"
            class="flex items-center gap-3 rounded-xl border border-border bg-background/40 p-3 text-left hover:bg-accent"
            @click="router.push(`/u/${user.id}`); listOpen = false"
          >
            <UserAvatar :username="user.username" :id="user.id" :src="user.avatar_url" :size="40" />
            <span>@{{ user.username }}</span>
          </button>
        </CardContent>
      </Card>
    </div>
  </AppShell>
</template>

<style scoped>
.metric-card {
  display: grid;
  min-height: 8rem;
  align-content: center;
  gap: 0.4rem;
  border: 1px solid var(--border);
  border-radius: 0.75rem;
  background: var(--card);
  padding: 1.25rem;
  text-align: left;
  transition: 160ms ease;
}

button.metric-card:hover {
  border-color: color-mix(in oklab, var(--primary) 40%, var(--border));
  transform: translateY(-2px);
}

.video-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
  gap: 1rem;
}

.video-tile {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 0.75rem;
  background: var(--background);
  padding: 0;
  text-align: left;
}

.video-tile img {
  width: 100%;
  aspect-ratio: 9 / 12;
  object-fit: cover;
}

.video-tile span {
  display: block;
  overflow: hidden;
  padding: 0.75rem;
  font-size: 0.875rem;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
