<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import {
  Clapperboard,
  Compass,
  Bell,
  Flame,
  LogOut,
  Menu,
  MessageCircle,
  Network,
  Search,
  Settings,
  Upload,
  UserRound,
} from '@lucide/vue'

import * as accountApi from '@/api/account'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet'
import { useAuthStore } from '@/stores/auth'
import { useSocialStore } from '@/stores/social'
import { useToastStore } from '@/stores/toast'

defineProps<{ full?: boolean }>()

const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()
const router = useRouter()
const route = useRoute()
const search = ref(typeof route.query.q === 'string' ? route.query.q : '')

const navItems = [
  { to: '/', label: '发现', icon: Compass },
  { to: '/hot', label: '热榜', icon: Flame },
  { to: '/video', label: '发布', icon: Upload },
  { to: '/account', label: '我的', icon: UserRound },
  { to: '/messages', label: '私信', icon: MessageCircle },
  { to: '/notifications', label: '通知', icon: Bell },
  { to: '/architecture', label: '架构', icon: Network },
  { to: '/settings', label: '设置', icon: Settings },
]

const userLabel = computed(() => auth.claims?.username ?? '演示账号')

watch(
  () => route.query.q,
  (value) => {
    search.value = typeof value === 'string' ? value : ''
  },
)

watch(
  () => auth.isLoggedIn,
  (loggedIn) => {
    if (loggedIn) void social.refreshMine()
    else social.clear()
  },
  { immediate: true },
)

async function onSearch() {
  const query = search.value.trim()
  await router.push({ path: '/', query: query ? { q: query } : {} })
}

async function logout() {
  try {
    await accountApi.logout()
  } catch {
    // Token may already be invalid; local logout must still complete.
  }
  auth.clearTokens()
  social.clear()
  toast.info('已退出演示系统')
  await router.replace('/login')
}
</script>

<template>
  <div class="min-h-screen bg-background text-foreground">
    <aside class="fixed inset-y-0 left-0 z-40 hidden w-64 border-r border-border bg-card/90 p-4 backdrop-blur-xl lg:flex lg:flex-col">
      <RouterLink to="/" class="mb-8 flex items-center gap-3 px-2">
        <span class="grid size-10 place-items-center rounded-xl bg-primary text-primary-foreground">
          <Clapperboard class="size-5" />
        </span>
        <span>
          <span class="block text-sm font-semibold">FrameFlow</span>
          <span class="block text-xs text-muted-foreground">Video Feed System</span>
        </span>
      </RouterLink>

      <nav class="grid gap-1">
        <RouterLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-muted-foreground transition hover:bg-accent hover:text-foreground"
          active-class="bg-accent text-foreground"
        >
          <component :is="item.icon" class="size-4" />
          {{ item.label }}
        </RouterLink>
      </nav>

      <div class="mt-auto rounded-xl border border-border bg-background/50 p-3">
        <div class="mb-3 min-w-0">
          <p class="truncate text-sm font-medium">@{{ userLabel }}</p>
          <p class="text-xs text-muted-foreground">Private interview demo</p>
        </div>
        <Button variant="outline" size="sm" class="w-full" @click="logout">
          <LogOut class="size-4" />
          退出登录
        </Button>
      </div>
    </aside>

    <div class="lg:pl-64">
      <header class="sticky top-0 z-30 flex h-16 items-center gap-3 border-b border-border bg-background/80 px-4 backdrop-blur-xl md:px-6">
        <Sheet>
          <SheetTrigger as-child>
            <Button variant="ghost" size="icon" class="lg:hidden">
              <Menu class="size-5" />
            </Button>
          </SheetTrigger>
          <SheetContent side="left" class="pt-16">
            <nav class="grid gap-2">
              <RouterLink
                v-for="item in navItems"
                :key="item.to"
                :to="item.to"
                class="flex items-center gap-3 rounded-lg px-3 py-3 text-sm text-muted-foreground hover:bg-accent hover:text-foreground"
                active-class="bg-accent text-foreground"
              >
                <component :is="item.icon" class="size-4" />
                {{ item.label }}
              </RouterLink>
            </nav>
          </SheetContent>
        </Sheet>

        <form class="mx-auto flex w-full max-w-xl items-center gap-2" @submit.prevent="onSearch">
          <div class="relative flex-1">
            <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input v-model="search" class="pl-9" placeholder="搜索标题或作者" />
          </div>
          <Button variant="secondary" type="submit">搜索</Button>
        </form>

        <Button class="hidden sm:inline-flex" @click="router.push('/video')">
          <Upload class="size-4" />
          发布
        </Button>
      </header>

      <main :class="$props.full ? 'h-[calc(100vh-4rem)] overflow-hidden' : 'min-h-[calc(100vh-4rem)] p-4 md:p-6'">
        <slot />
      </main>
    </div>
  </div>
</template>
