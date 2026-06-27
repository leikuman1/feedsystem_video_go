<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Heart,
  Info,
  MessageCircle,
  Share2,
  UserPlus,
  Volume2,
  VolumeX,
} from '@lucide/vue'
import gsap from 'gsap'

import type { FeedVideoItem } from '@/api/types'
import AppShell from '@/components/AppShell.vue'
import CommentDrawer from '@/components/CommentDrawer.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useLikeFollow } from '@/composables/useLikeFollow'
import { useVideoFeed } from '@/composables/useVideoFeed'
import { useVideoPlayer } from '@/composables/useVideoPlayer'
import { useAuthStore } from '@/stores/auth'
import { useSocialStore } from '@/stores/social'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const root = ref<HTMLElement | null>(null)

const { tab, following, currentState, loadFollowing, ensureTabLoaded, loadMoreIfNeeded } = useVideoFeed()
const scroller = ref<HTMLDivElement | null>(null)
const { muted, activeIndex, videoMap, setVideoRef, scrollToIndex, onScroll, playActive, toggleMute, togglePlayPause } = useVideoPlayer(scroller)
const { likeBusy, followBusy, toggleLike, toggleFollow, share } = useLikeFollow(() => undefined)

const drawerVideo = ref<FeedVideoItem | null>(null)
const drawerOpen = ref(false)

const query = computed(() => (typeof route.query.q === 'string' ? route.query.q.trim().toLowerCase() : ''))
const filteredItems = computed(() => {
  const items = currentState.value.items
  if (!query.value) return items
  return items.filter((video) => (
    video.title.toLowerCase().includes(query.value)
    || video.author.username.toLowerCase().includes(query.value)
  ))
})
const activeItem = computed(() => filteredItems.value[activeIndex.value] ?? null)
const visibleRange = computed(() => ({
  start: Math.max(0, activeIndex.value - 1),
  end: Math.min(filteredItems.value.length - 1, activeIndex.value + 1),
}))
const myAccountId = computed(() => auth.claims?.account_id ?? 0)

function videoSource(item: FeedVideoItem, index: number) {
  return index === activeIndex.value ? item.play_url : undefined
}

function animateActiveMeta() {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches || !root.value) return
  const elements = root.value.querySelectorAll('.feed-slide.active [data-feed-reveal]')
  gsap.fromTo(elements, { opacity: 0, y: 18 }, {
    opacity: 1,
    y: 0,
    duration: 0.45,
    stagger: 0.05,
    ease: 'power2.out',
    overwrite: true,
  })
}

function openComments(item: FeedVideoItem) {
  drawerVideo.value = item
  drawerOpen.value = true
}

function closeDrawer() {
  drawerOpen.value = false
  drawerVideo.value = null
}

watch(activeItem, async () => {
  await nextTick()
  await playActive(activeItem.value?.id)
  await loadMoreIfNeeded(activeIndex.value)
  animateActiveMeta()
})

watch(() => tab.value, async () => {
  activeIndex.value = 0
  videoMap.clear()
  if (scroller.value) scroller.value.scrollTop = 0
  await ensureTabLoaded()
  await nextTick()
  await playActive(activeItem.value?.id)
  animateActiveMeta()
})

watch(query, async () => {
  activeIndex.value = 0
  if (scroller.value) scroller.value.scrollTop = 0
  await nextTick()
  await playActive(activeItem.value?.id)
})

watch(() => filteredItems.value.length, (length) => {
  if (length === 0) activeIndex.value = 0
  else if (activeIndex.value > length - 1) activeIndex.value = length - 1
})

watch(() => auth.isLoggedIn, async (loggedIn) => {
  if (tab.value === 'following' && loggedIn && following.items.length === 0) await loadFollowing(true)
})

function onKeydown(event: KeyboardEvent) {
  const target = event.target as HTMLElement | null
  if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') || drawerOpen.value) return
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    scrollToIndex(activeIndex.value + 1, filteredItems.value.length)
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    scrollToIndex(activeIndex.value - 1, filteredItems.value.length)
  } else if (event.key === ' ') {
    event.preventDefault()
    togglePlayPause(activeItem.value?.id)
  } else if (event.key.toLowerCase() === 'm') {
    event.preventDefault()
    toggleMute()
  } else if (event.key.toLowerCase() === 'c' && activeItem.value) {
    event.preventDefault()
    openComments(activeItem.value)
  }
}

onMounted(async () => {
  await ensureTabLoaded()
  await nextTick()
  await playActive(activeItem.value?.id)
  animateActiveMeta()
  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <AppShell full>
    <div ref="root" class="relative h-full overflow-hidden bg-black">
      <div class="absolute inset-x-0 top-0 z-20 flex items-center justify-between border-b border-white/10 bg-black/35 px-4 py-3 backdrop-blur-xl">
        <div class="flex items-center gap-2">
          <Button :variant="tab === 'recommend' ? 'default' : 'ghost'" size="sm" @click="tab = 'recommend'">推荐</Button>
          <Button :variant="tab === 'following' ? 'default' : 'ghost'" size="sm" @click="tab = 'following'">关注</Button>
          <Button :variant="tab === 'hot' ? 'default' : 'ghost'" size="sm" @click="tab = 'hot'">热度</Button>
        </div>
        <div class="flex items-center gap-2">
          <Button variant="outline" size="sm" class="border-white/15 bg-black/30" @click="toggleMute">
            <VolumeX v-if="muted" class="size-4" />
            <Volume2 v-else class="size-4" />
            <span class="hidden sm:inline">{{ muted ? '静音' : '有声' }}</span>
          </Button>
          <Button
            variant="outline"
            size="sm"
            class="border-white/15 bg-black/30"
            :disabled="!activeItem"
            @click="activeItem && router.push(`/video/${activeItem.id}`)"
          >
            <Info class="size-4" />
            详情
          </Button>
        </div>
      </div>

      <div ref="scroller" class="h-full snap-y snap-mandatory overflow-y-auto [scrollbar-width:none]" @scroll="onScroll">
        <div v-if="currentState.loading && currentState.items.length === 0" class="grid h-full place-items-center text-sm text-white/60">
          正在加载视频…
        </div>
        <div v-else-if="currentState.error && currentState.items.length === 0" class="grid h-full place-items-center text-sm text-red-300">
          {{ currentState.error }}
        </div>
        <div v-else-if="filteredItems.length === 0" class="grid h-full place-items-center text-sm text-white/60">
          没有匹配内容
        </div>

        <template v-for="(item, index) in filteredItems" :key="`${tab}-${item.id}`">
        <section
          v-if="index >= visibleRange.start && index <= visibleRange.end"
          class="feed-slide relative h-full snap-start overflow-hidden"
          :class="{ active: index === activeIndex }"
        >
          <video
            :ref="(element) => setVideoRef(item.id, element as HTMLVideoElement | null)"
            class="absolute inset-0 size-full bg-black object-contain"
            :src="videoSource(item, index)"
            :poster="item.cover_url"
            playsinline
            preload="none"
            loop
            @click="togglePlayPause(item.id)"
            @dblclick.prevent="toggleLike(item)"
          />
          <div class="pointer-events-none absolute inset-0 bg-gradient-to-t from-black via-black/5 to-black/25" />

          <div class="absolute inset-x-0 bottom-0 z-10 flex items-end justify-between gap-6 p-5 pb-8 md:p-8">
            <div class="max-w-2xl pr-16">
              <RouterLink
                data-feed-reveal
                :to="`/u/${item.author.id}`"
                class="mb-4 inline-flex items-center gap-3"
              >
                <UserAvatar :username="item.author.username" :id="item.author.id" :size="42" />
                <span class="font-medium text-white">@{{ item.author.username }}</span>
              </RouterLink>
              <h1 data-feed-reveal class="text-balance text-2xl font-semibold text-white md:text-4xl">{{ item.title }}</h1>
              <p v-if="item.description" data-feed-reveal class="mt-3 max-w-xl text-sm leading-6 text-white/70 md:text-base">
                {{ item.description }}
              </p>
              <div data-feed-reveal class="mt-4 flex flex-wrap gap-2">
                <Badge variant="outline" class="border-white/15 bg-black/25 text-white/75">↑ ↓ 切换</Badge>
                <Badge variant="outline" class="hidden border-white/15 bg-black/25 text-white/75 sm:inline-flex">M 静音 · C 评论</Badge>
              </div>
            </div>

            <div data-feed-reveal class="grid shrink-0 gap-3">
              <button
                class="grid size-14 place-items-center rounded-full border border-white/15 bg-black/35 text-white backdrop-blur transition hover:scale-105 hover:bg-white/15 disabled:opacity-50"
                :disabled="!!likeBusy[String(item.id)]"
                @click.stop="toggleLike(item)"
              >
                <Heart class="size-5" :class="item.is_liked ? 'fill-primary text-primary' : ''" />
                <span class="text-[10px]">{{ item.likes_count }}</span>
              </button>
              <button class="grid size-14 place-items-center rounded-full border border-white/15 bg-black/35 text-white backdrop-blur hover:bg-white/15" @click.stop="openComments(item)">
                <MessageCircle class="size-5" />
                <span class="text-[10px]">评论</span>
              </button>
              <button
                v-if="!myAccountId || myAccountId !== item.author.id"
                class="grid size-14 place-items-center rounded-full border border-white/15 bg-black/35 text-white backdrop-blur hover:bg-white/15 disabled:opacity-50"
                :disabled="!!followBusy[String(item.author.id)]"
                @click.stop="toggleFollow(item.author.id)"
              >
                <UserPlus class="size-5" />
                <span class="text-[10px]">{{ social.isFollowing(item.author.id) ? '已关注' : '关注' }}</span>
              </button>
              <button class="grid size-14 place-items-center rounded-full border border-white/15 bg-black/35 text-white backdrop-blur hover:bg-white/15" @click.stop="share(item)">
                <Share2 class="size-5" />
                <span class="text-[10px]">分享</span>
              </button>
            </div>
          </div>
        </section>
        </template>
      </div>

      <CommentDrawer v-if="drawerOpen" :video="drawerVideo" @close="closeDrawer" />
    </div>
  </AppShell>
</template>
