<script setup lang="ts">
import { computed, onMounted, reactive, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessageCircle, UserCheck, UserPlus, Video as VideoIcon } from '@lucide/vue'

import { ApiError } from '@/api/client'
import * as accountApi from '@/api/account'
import * as socialApi from '@/api/social'
import type { Account, Video } from '@/api/types'
import * as videoApi from '@/api/video'
import AppShell from '@/components/AppShell.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useAuthStore } from '@/stores/auth'
import { useSocialStore } from '@/stores/social'
import { useToastStore } from '@/stores/toast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()
const userId = computed(() => Number(route.params.id))
const isMe = computed(() => auth.claims?.account_id === userId.value)
const isFollowing = computed(() => social.isFollowing(userId.value))

const state = reactive({
  loading: false,
  error: '',
  user: null as Account | null,
  videos: [] as Video[],
  followerCount: 0,
  followingCount: 0,
})

async function load() {
  if (!Number.isFinite(userId.value) || userId.value <= 0) {
    state.error = '无效的用户编号'
    return
  }
  state.loading = true
  state.error = ''
  try {
    const [user, videos, followers, following] = await Promise.all([
      accountApi.findById(userId.value),
      videoApi.listByAuthorId(userId.value),
      socialApi.getAllFollowers(userId.value),
      socialApi.getAllVloggers(userId.value),
    ])
    state.user = user
    state.videos = videos
    state.followerCount = followers.follower_count
    state.followingCount = following.vlogger_count
  } catch (error) {
    state.error = error instanceof ApiError ? error.message : String(error)
  } finally {
    state.loading = false
  }
}

async function toggleFollow() {
  try {
    if (isFollowing.value) await social.unfollow(userId.value)
    else await social.follow(userId.value)
    await load()
  } catch (error) {
    toast.error(error instanceof ApiError ? error.message : String(error))
  }
}

watch(() => route.params.id, load)
onMounted(load)
</script>

<template>
  <AppShell>
    <div class="mx-auto grid max-w-6xl gap-6">
      <Card>
        <CardContent class="flex flex-col gap-6 p-6 md:flex-row md:items-center md:justify-between">
          <div class="flex items-center gap-4">
            <UserAvatar
              :username="state.user?.username ?? 'User'"
              :id="state.user?.id ?? userId"
              :src="state.user?.avatar_url"
              :size="76"
            />
            <div>
              <h1 class="text-2xl font-semibold">@{{ state.user?.username ?? '加载中' }}</h1>
              <p class="mt-1 font-mono text-xs text-muted-foreground">ACCOUNT #{{ userId }}</p>
              <p v-if="state.user?.bio" class="mt-3 max-w-xl text-sm leading-6 text-foreground/70">{{ state.user.bio }}</p>
            </div>
          </div>
          <div class="flex flex-wrap gap-2">
            <Button v-if="isMe" variant="outline" @click="router.push('/settings')">编辑资料</Button>
            <template v-else>
              <Button variant="outline" @click="router.push(`/messages/${userId}`)">
                <MessageCircle class="size-4" />
                私信
              </Button>
              <Button :disabled="state.loading" @click="toggleFollow">
                <UserCheck v-if="isFollowing" class="size-4" />
                <UserPlus v-else class="size-4" />
                {{ isFollowing ? '已关注' : '关注' }}
              </Button>
            </template>
          </div>
        </CardContent>
      </Card>

      <div class="grid gap-4 sm:grid-cols-3">
        <div class="profile-metric"><span>{{ state.followerCount }}</span><small>粉丝</small></div>
        <div class="profile-metric"><span>{{ state.followingCount }}</span><small>关注</small></div>
        <div class="profile-metric"><span>{{ state.videos.length }}</span><small>作品</small></div>
      </div>

      <Card>
        <CardHeader>
          <VideoIcon class="mb-2 size-5 text-primary" />
          <CardTitle>发布作品</CardTitle>
          <CardDescription>按发布时间倒序展示。</CardDescription>
        </CardHeader>
        <CardContent>
          <div v-if="state.error" class="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">{{ state.error }}</div>
          <div v-else-if="state.loading" class="py-16 text-center text-sm text-muted-foreground">加载中…</div>
          <div v-else-if="state.videos.length === 0" class="py-16 text-center text-sm text-muted-foreground">暂无作品</div>
          <div v-else class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4">
            <button v-for="video in state.videos" :key="video.id" class="overflow-hidden rounded-xl border border-border bg-background text-left" @click="router.push(`/video/${video.id}`)">
              <img :src="video.cover_url" :alt="video.title" class="aspect-[9/12] w-full object-cover" loading="lazy" />
              <div class="p-3">
                <p class="truncate text-sm font-medium">{{ video.title }}</p>
                <p class="mt-1 text-xs text-muted-foreground">♥ {{ video.likes_count }}</p>
              </div>
            </button>
          </div>
        </CardContent>
      </Card>
    </div>
  </AppShell>
</template>

<style scoped>
.profile-metric {
  display: grid;
  gap: 0.25rem;
  border: 1px solid var(--border);
  border-radius: 0.75rem;
  background: var(--card);
  padding: 1.25rem;
}

.profile-metric span {
  font-size: 1.5rem;
  font-weight: 600;
}

.profile-metric small {
  color: var(--muted-foreground);
}
</style>
