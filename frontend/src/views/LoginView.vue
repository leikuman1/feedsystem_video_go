<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowRight, Boxes, Database, LockKeyhole, Radio, Sparkles } from '@lucide/vue'
import gsap from 'gsap'

import * as accountApi from '@/api/account'
import { ApiError } from '@/api/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'

const auth = useAuthStore()
const toast = useToastStore()
const route = useRoute()
const router = useRouter()
const root = ref<HTMLElement | null>(null)
const busy = ref(false)
const form = reactive({ username: '', password: '' })

const capabilities = [
  { icon: Database, label: 'MinIO multipart media storage' },
  { icon: Radio, label: 'RabbitMQ + SSE event pipeline' },
  { icon: Boxes, label: 'Redis hot feed and cache fallback' },
]

onMounted(() => {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches || !root.value) return
  gsap.from(root.value.querySelectorAll('[data-reveal]'), {
    opacity: 0,
    y: 28,
    duration: 0.7,
    stagger: 0.09,
    ease: 'power3.out',
  })
})

async function login() {
  if (busy.value) return
  const username = form.username.trim()
  const password = form.password
  if (!username || !password) {
    toast.error('请输入演示账号和密码')
    return
  }

  busy.value = true
  try {
    const response = await accountApi.login(username, password)
    auth.setTokens(response.token, response.refresh_token ?? '')
    toast.success('已进入演示系统')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect)
  } catch (error) {
    toast.error(error instanceof ApiError ? error.message : String(error))
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main ref="root" class="relative grid min-h-screen overflow-hidden bg-background lg:grid-cols-[1.25fr_0.75fr]">
    <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_18%_18%,oklch(0.7_0.17_35/.18),transparent_34%),radial-gradient(circle_at_85%_78%,oklch(0.5_0.12_260/.16),transparent_32%)]" />
    <section class="relative hidden min-h-screen flex-col justify-between border-r border-border p-10 lg:flex xl:p-16">
      <div data-reveal class="flex items-center gap-3">
        <span class="grid size-11 place-items-center rounded-xl bg-primary text-primary-foreground">
          <Sparkles class="size-5" />
        </span>
        <div>
          <p class="font-semibold">FrameFlow</p>
          <p class="text-xs text-muted-foreground">Backend interview showcase</p>
        </div>
      </div>

      <div class="max-w-2xl">
        <Badge data-reveal variant="outline" class="mb-6 border-primary/30 text-primary">PRIVATE DEMO</Badge>
        <h1 data-reveal class="text-balance text-5xl font-semibold leading-[1.08] tracking-tight xl:text-7xl">
          一个短视频 Feed 系统。
        </h1>
        <p data-reveal class="mt-6 max-w-xl text-lg leading-8 text-muted-foreground">
          聚焦对象存储、异步消息断点续传与 Feed 分页。登录后可体验完整产品流程，并查看系统架构说明。
        </p>
        <div class="mt-10 grid gap-3">
          <div
            v-for="item in capabilities"
            :key="item.label"
            data-reveal
            class="flex items-center gap-3 rounded-xl border border-border bg-card/60 px-4 py-3 text-sm"
          >
            <component :is="item.icon" class="size-4 text-primary" />
            {{ item.label }}
          </div>
        </div>
      </div>

      <p data-reveal class="text-xs text-muted-foreground">
        Vue 3 · Go · MySQL · Redis · RabbitMQ · MinIO
      </p>
    </section>

    <section class="relative grid min-h-screen place-items-center p-5 sm:p-10">
      <Card data-reveal class="w-full max-w-md border-border/80 bg-card/90 shadow-2xl backdrop-blur">
        <CardHeader class="space-y-3">
          <span class="grid size-11 place-items-center rounded-xl bg-primary/15 text-primary">
            <LockKeyhole class="size-5" />
          </span>
          <CardTitle class="text-2xl">进入项目演示</CardTitle>
          <CardDescription></CardDescription>
        </CardHeader>
        <CardContent>
          <form class="grid gap-5" @submit.prevent="login">
            <label class="grid gap-2 text-sm">
              <span class="text-muted-foreground">账号</span>
              <Input v-model="form.username" autocomplete="username" placeholder="interviewer" />
            </label>
            <label class="grid gap-2 text-sm">
              <span class="text-muted-foreground">密码</span>
              <Input
                v-model="form.password"
                type="password"
                autocomplete="current-password"
                placeholder="••••••••"
              />
            </label>
            <Button type="submit" size="lg" :disabled="busy">
              {{ busy ? '正在验证…' : '进入演示' }}
              <ArrowRight class="size-4" />
            </Button>
          </form>
          <p class="mt-5 text-xs leading-5 text-muted-foreground">
          </p>
        </CardContent>
      </Card>
    </section>
  </main>
</template>
