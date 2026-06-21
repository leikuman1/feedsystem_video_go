<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, MessageCircle, RefreshCw, Send } from '@lucide/vue'

import { ApiError } from '@/api/client'
import * as accountApi from '@/api/account'
import * as messageApi from '@/api/message'
import type { Account, DirectMessage } from '@/api/types'
import AppShell from '@/components/AppShell.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Textarea } from '@/components/ui/textarea'
import { useAuthStore } from '@/stores/auth'
import { useSocialStore } from '@/stores/social'
import { useToastStore } from '@/stores/toast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()
const listElement = ref<HTMLDivElement | null>(null)
const content = ref('')

const peerId = computed(() => typeof route.params.peerId === 'string' ? Number(route.params.peerId) : 0)
const myId = computed(() => auth.claims?.account_id ?? 0)
const hasPeer = computed(() => Number.isFinite(peerId.value) && peerId.value > 0)
const state = reactive({
  loading: false,
  sending: false,
  error: '',
  peer: null as Account | null,
  messages: [] as DirectMessage[],
})

const messages = computed(() => [...state.messages].sort((a, b) => (
  new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
)))
const contacts = computed(() => {
  const map = new Map<number, Account>()
  for (const account of [...social.vloggers, ...social.followers]) map.set(account.id, account)
  return [...map.values()].filter((account) => account.id !== myId.value)
})

async function scrollToBottom() {
  await nextTick()
  if (listElement.value) listElement.value.scrollTop = listElement.value.scrollHeight
}

async function load() {
  state.error = ''
  if (!hasPeer.value) {
    await social.refreshMine()
    state.peer = null
    state.messages = []
    return
  }
  if (peerId.value === myId.value) {
    state.error = '不能给自己发送私信'
    return
  }
  state.loading = true
  try {
    const [peer, response] = await Promise.all([
      accountApi.findById(peerId.value),
      messageApi.listMessages(peerId.value),
    ])
    state.peer = peer
    state.messages = response.messages ?? []
    await scrollToBottom()
  } catch (error) {
    state.error = error instanceof ApiError ? error.message : String(error)
  } finally {
    state.loading = false
  }
}

async function send() {
  const text = content.value.trim()
  if (!text || state.sending || !state.peer) return
  state.sending = true
  try {
    const message = await messageApi.sendMessage(peerId.value, text)
    state.messages.push(message)
    content.value = ''
    await scrollToBottom()
  } catch (error) {
    toast.error(error instanceof ApiError ? error.message : String(error))
  } finally {
    state.sending = false
  }
}

function formatTime(value: string) {
  return new Date(value).toLocaleString([], { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

watch(() => route.params.peerId, load)
onMounted(load)
</script>

<template>
  <AppShell>
    <Card class="mx-auto h-[calc(100vh-7rem)] min-h-[34rem] max-w-6xl overflow-hidden">
      <div v-if="!hasPeer" class="grid h-full grid-rows-[auto_1fr]">
        <header class="flex items-center justify-between border-b border-border p-5">
          <div>
            <h1 class="text-xl font-semibold">私信</h1>
            <p class="mt-1 text-sm text-muted-foreground">选择关注或粉丝开始会话。</p>
          </div>
          <Button variant="outline" @click="social.refreshMine"><RefreshCw class="size-4" />刷新</Button>
        </header>
        <div class="overflow-y-auto p-4">
          <div v-if="contacts.length === 0" class="grid h-full place-items-center text-center text-sm text-muted-foreground">
            <div><MessageCircle class="mx-auto mb-3 size-8" />暂无可聊天联系人</div>
          </div>
          <div v-else class="grid gap-2">
            <button
              v-for="contact in contacts"
              :key="contact.id"
              class="flex items-center gap-3 rounded-xl border border-border bg-background/40 p-3 text-left transition hover:bg-accent"
              @click="router.push(`/messages/${contact.id}`)"
            >
              <UserAvatar :username="contact.username" :id="contact.id" :src="contact.avatar_url" :size="44" />
              <span class="min-w-0 flex-1">
                <span class="block truncate font-medium">@{{ contact.username }}</span>
                <span class="block text-xs text-muted-foreground">点击开始聊天</span>
              </span>
            </button>
          </div>
        </div>
      </div>

      <div v-else class="grid h-full grid-rows-[auto_1fr_auto]">
        <header class="flex items-center gap-3 border-b border-border p-4">
          <Button variant="ghost" size="icon" @click="router.push('/messages')"><ArrowLeft class="size-4" /></Button>
          <button class="flex min-w-0 flex-1 items-center gap-3 text-left" @click="state.peer && router.push(`/u/${state.peer.id}`)">
            <UserAvatar :username="state.peer?.username ?? 'User'" :id="peerId" :src="state.peer?.avatar_url" :size="42" />
            <span class="min-w-0">
              <span class="block truncate font-medium">@{{ state.peer?.username ?? '加载中' }}</span>
              <span class="block text-xs text-muted-foreground">ACCOUNT #{{ peerId }}</span>
            </span>
          </button>
          <Button variant="ghost" size="icon" :disabled="state.loading" @click="load"><RefreshCw class="size-4" /></Button>
        </header>

        <div ref="listElement" class="flex min-h-0 flex-col gap-3 overflow-y-auto p-5">
          <div v-if="state.loading" class="m-auto text-sm text-muted-foreground">加载消息…</div>
          <div v-else-if="state.error" class="m-auto text-sm text-destructive">{{ state.error }}</div>
          <div v-else-if="messages.length === 0" class="m-auto text-center text-sm text-muted-foreground">发送第一条消息开始会话。</div>

          <div
            v-for="message in messages"
            :key="message.id"
            class="flex"
            :class="message.from_id === myId ? 'justify-end' : 'justify-start'"
          >
            <div class="max-w-[78%]">
              <div
                class="rounded-2xl border px-4 py-2.5 text-sm leading-6"
                :class="message.from_id === myId ? 'border-primary/30 bg-primary text-primary-foreground' : 'border-border bg-background'"
              >
                {{ message.content }}
              </div>
              <p class="mt-1 px-1 text-xs text-muted-foreground" :class="message.from_id === myId ? 'text-right' : ''">
                {{ formatTime(message.created_at) }}
              </p>
            </div>
          </div>
        </div>

        <footer class="grid gap-3 border-t border-border p-4 sm:grid-cols-[1fr_auto]">
          <Textarea v-model="content" class="min-h-11 resize-none" placeholder="输入私信内容" @keydown.enter.exact.prevent="send" />
          <Button :disabled="!content.trim() || state.sending || !!state.error" @click="send">
            <Send class="size-4" />
            {{ state.sending ? '发送中' : '发送' }}
          </Button>
        </footer>
      </div>
    </Card>
  </AppShell>
</template>
