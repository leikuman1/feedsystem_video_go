<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Bell, CheckCheck, Heart, MessageCircle, UserPlus } from '@lucide/vue'

import * as notificationApi from '@/api/notification'
import type { Notification } from '@/api/types'
import AppShell from '@/components/AppShell.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'

const auth = useAuthStore()
const toast = useToastStore()
const router = useRouter()
const items = ref<Notification[]>([])
const loading = ref(false)
let stream: EventSource | null = null

const unread = computed(() => items.value.filter((item) => !item.is_read).length)

function iconFor(type: string) {
  if (type === 'like') return Heart
  if (type === 'comment') return MessageCircle
  return UserPlus
}

async function load() {
  loading.value = true
  try {
    const response = await notificationApi.listNotifications()
    items.value = response.notifications ?? []
  } catch (error) {
    toast.error(String(error))
  } finally {
    loading.value = false
  }
}

async function markAllRead() {
  await notificationApi.markRead()
  items.value = items.value.map((item) => ({ ...item, is_read: true }))
}

async function openNotification(item: Notification) {
  if (!item.is_read) {
    await notificationApi.markRead(item.id)
    item.is_read = true
  }
  if (item.type === 'follow') await router.push(`/u/${item.sender_id}`)
  else await router.push(`/video/${item.target_id}`)
}

onMounted(async () => {
  await load()
  if (!auth.token) return
  stream = new EventSource(`/api/notification/stream?token=${encodeURIComponent(auth.token)}`)
  stream.onmessage = (event) => {
    try {
      const item = JSON.parse(event.data) as Notification
      items.value.unshift(item)
      toast.info(item.content)
    } catch {
      // Ignore malformed events.
    }
  }
})

onBeforeUnmount(() => stream?.close())
</script>

<template>
  <AppShell>
    <Card class="mx-auto max-w-4xl">
      <CardHeader>
        <div class="flex items-start justify-between gap-4">
          <div>
            <span class="mb-3 grid size-11 place-items-center rounded-xl bg-primary/15 text-primary"><Bell class="size-5" /></span>
            <CardTitle class="text-2xl">实时通知</CardTitle>
            <CardDescription class="mt-2">点赞、评论和关注事件通过 RabbitMQ 落库，并由 SSE 实时推送。</CardDescription>
          </div>
          <Button variant="outline" :disabled="unread === 0" @click="markAllRead">
            <CheckCheck class="size-4" />
            全部已读
          </Button>
        </div>
      </CardHeader>
      <CardContent class="grid gap-3">
        <div v-if="loading" class="py-14 text-center text-sm text-muted-foreground">加载通知…</div>
        <div v-else-if="items.length === 0" class="py-14 text-center text-sm text-muted-foreground">暂无通知</div>
        <button
          v-for="item in items"
          v-else
          :key="item.id"
          class="flex items-center gap-4 rounded-xl border p-4 text-left transition hover:bg-accent"
          :class="item.is_read ? 'border-border bg-background/30' : 'border-primary/30 bg-primary/5'"
          @click="openNotification(item)"
        >
          <span class="grid size-10 place-items-center rounded-full bg-primary/10 text-primary">
            <component :is="iconFor(item.type)" class="size-4" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block text-sm font-medium">{{ item.content }}</span>
            <span class="mt-1 block text-xs text-muted-foreground">{{ new Date(item.created_at).toLocaleString() }}</span>
          </span>
          <span v-if="!item.is_read" class="size-2 rounded-full bg-primary" />
        </button>
      </CardContent>
    </Card>
  </AppShell>
</template>
