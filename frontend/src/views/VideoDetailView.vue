<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  Heart,
  MessageCircle,
  Share2,
  UserPlus,
  Volume2,
  VolumeX,
} from '@lucide/vue'
import gsap from 'gsap'

import { ApiError } from '@/api/client'
import * as likeApi from '@/api/like'
import type { Video } from '@/api/types'
import * as videoApi from '@/api/video'
import AppShell from '@/components/AppShell.vue'
import CommentDrawer from '@/components/CommentDrawer.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/stores/auth'
import { useSocialStore } from '@/stores/social'
import { useToastStore } from '@/stores/toast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()
const videoElement = ref<HTMLVideoElement | null>(null)
const root = ref<HTMLElement | null>(null)
const muted = ref(true)
const commentsOpen = ref(false)
const id = computed(() => Number(route.params.id))

const state = reactive({
  loading: false,
  error: '',
  video: null as Video | null,
  isLiked: false,
  busy: false,
})

async function load() {
  if (!Number.isFinite(id.value) || id.value <= 0) {
    state.error = '无效的视频编号'
    return
  }
  state.loading = true
  state.error = ''
  try {
    const [video, liked] = await Promise.all([
      videoApi.getDetail(id.value),
      likeApi.isLiked(id.value).catch(() => ({ is_liked: false })),
    ])
    state.video = video
    state.isLiked = liked.is_liked
    await nextTick()
    videoElement.value?.play().catch(() => undefined)
    if (!window.matchMedia('(prefers-reduced-motion: reduce)').matches && root.value) {
      gsap.from(root.value.querySelectorAll('[data-detail-reveal]'), {
        opacity: 0,
        y: 18,
        duration: 0.5,
        stagger: 0.06,
        ease: 'power2.out',
      })
    }
  } catch (error) {
    state.error = error instanceof ApiError ? error.message : String(error)
  } finally {
    state.loading = false
  }
}

function togglePlay() {
  const video = videoElement.value
  if (!video) return
  if (video.paused) void video.play()
  else video.pause()
}

function toggleMute() {
  muted.value = !muted.value
  if (videoElement.value) videoElement.value.muted = muted.value
}

async function toggleLike() {
  if (!state.video || state.busy) return
  state.busy = true
  try {
    if (state.isLiked) {
      await likeApi.unlike(state.video.id)
      state.video.likes_count = Math.max(0, state.video.likes_count - 1)
    } else {
      await likeApi.like(state.video.id)
      state.video.likes_count += 1
    }
    state.isLiked = !state.isLiked
  } catch (error) {
    toast.error(error instanceof ApiError ? error.message : String(error))
  } finally {
    state.busy = false
  }
}

async function toggleFollow() {
  if (!state.video || state.busy) return
  state.busy = true
  try {
    if (social.isFollowing(state.video.author_id)) await social.unfollow(state.video.author_id)
    else await social.follow(state.video.author_id)
  } catch (error) {
    toast.error(error instanceof ApiError ? error.message : String(error))
  } finally {
    state.busy = false
  }
}

async function share() {
  if (!state.video) return
  const url = `${location.origin}/video/${state.video.id}`
  try {
    await navigator.clipboard.writeText(url)
    toast.success('链接已复制')
  } catch {
    window.prompt('复制链接', url)
  }
}

watch(id, () => void load())
onMounted(load)
</script>

<template>
  <AppShell full>
    <div ref="root" class="relative h-full overflow-hidden bg-black">
      <div v-if="state.loading" class="grid h-full place-items-center text-sm text-white/60">正在加载视频…</div>
      <div v-else-if="state.error" class="grid h-full place-items-center text-sm text-red-300">{{ state.error }}</div>

      <template v-else-if="state.video">
        <video
          ref="videoElement"
          class="absolute inset-0 size-full bg-black object-contain"
          :src="state.video.play_url"
          :poster="state.video.cover_url"
          :muted="muted"
          playsinline
          preload="metadata"
          loop
          @click="togglePlay"
        />
        <div class="pointer-events-none absolute inset-0 bg-gradient-to-t from-black via-transparent to-black/45" />

        <div class="absolute inset-x-0 top-0 z-10 flex items-center justify-between p-4 md:p-6">
          <Button variant="outline" class="border-white/15 bg-black/30 text-white" @click="router.push('/')">
            <ArrowLeft class="size-4" />
            返回
          </Button>
          <Button variant="outline" class="border-white/15 bg-black/30 text-white" @click="toggleMute">
            <VolumeX v-if="muted" class="size-4" />
            <Volume2 v-else class="size-4" />
            {{ muted ? '静音' : '有声' }}
          </Button>
        </div>

        <div class="absolute inset-x-0 bottom-0 z-10 flex items-end justify-between gap-6 p-5 pb-8 md:p-8">
          <div class="max-w-2xl pr-16">
            <RouterLink data-detail-reveal :to="`/u/${state.video.author_id}`" class="mb-4 inline-flex items-center gap-3">
              <UserAvatar :username="state.video.username" :id="state.video.author_id" :size="44" />
              <span class="font-medium text-white">@{{ state.video.username }}</span>
            </RouterLink>
            <h1 data-detail-reveal class="text-balance text-3xl font-semibold text-white md:text-5xl">{{ state.video.title }}</h1>
            <p v-if="state.video.description" data-detail-reveal class="mt-4 max-w-xl text-sm leading-6 text-white/70 md:text-base">
              {{ state.video.description }}
            </p>
          </div>

          <div data-detail-reveal class="grid shrink-0 gap-3">
            <button class="detail-action" :disabled="state.busy" @click.stop="toggleLike">
              <Heart class="size-5" :class="state.isLiked ? 'fill-primary text-primary' : ''" />
              <span>{{ state.video.likes_count }}</span>
            </button>
            <button class="detail-action" @click.stop="commentsOpen = true">
              <MessageCircle class="size-5" />
              <span>评论</span>
            </button>
            <button
              v-if="auth.claims?.account_id !== state.video.author_id"
              class="detail-action"
              :disabled="state.busy"
              @click.stop="toggleFollow"
            >
              <UserPlus class="size-5" />
              <span>{{ social.isFollowing(state.video.author_id) ? '已关注' : '关注' }}</span>
            </button>
            <button class="detail-action" @click.stop="share">
              <Share2 class="size-5" />
              <span>分享</span>
            </button>
          </div>
        </div>

        <CommentDrawer v-if="commentsOpen" :video="state.video" @close="commentsOpen = false" />
      </template>
    </div>
  </AppShell>
</template>

<style scoped>
.detail-action {
  display: grid;
  width: 3.5rem;
  height: 3.5rem;
  place-items: center;
  border: 1px solid rgb(255 255 255 / 0.15);
  border-radius: 9999px;
  background: rgb(0 0 0 / 0.35);
  color: white;
  backdrop-filter: blur(12px);
  transition: 160ms ease;
}

.detail-action:hover {
  transform: scale(1.05);
  background: rgb(255 255 255 / 0.14);
}

.detail-action span {
  font-size: 10px;
}
</style>
