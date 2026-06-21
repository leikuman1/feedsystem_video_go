<script setup lang="ts">
import { reactive, watch } from 'vue'
import { RefreshCw, Send, Trash2, X } from '@lucide/vue'

import { ApiError } from '@/api/client'
import * as commentApi from '@/api/comment'
import type { Comment } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'

const props = defineProps<{ video: { id: number; title: string } | null }>()
const emit = defineEmits<{ close: [] }>()
const auth = useAuthStore()
const toast = useToastStore()

const drawer = reactive({
  loading: false,
  error: '',
  comments: [] as Comment[],
  content: '',
})

function close() {
  drawer.comments = []
  drawer.content = ''
  drawer.error = ''
  emit('close')
}

async function loadComments() {
  if (!props.video) return
  drawer.loading = true
  drawer.error = ''
  try {
    drawer.comments = await commentApi.listAll(props.video.id)
  } catch (error) {
    drawer.error = error instanceof ApiError ? error.message : String(error)
  } finally {
    drawer.loading = false
  }
}

async function publishComment() {
  if (!props.video) return
  const content = drawer.content.trim()
  if (!content) return
  drawer.loading = true
  drawer.error = ''
  try {
    await commentApi.publish(props.video.id, content)
    drawer.content = ''
    await loadComments()
    toast.success('评论已发布')
  } catch (error) {
    drawer.error = error instanceof ApiError ? error.message : String(error)
    toast.error(drawer.error)
  } finally {
    drawer.loading = false
  }
}

function canDeleteComment(comment: Comment) {
  return !!auth.claims?.account_id && auth.claims.account_id === comment.author_id
}

async function deleteComment(commentId: number) {
  if (!window.confirm('确认删除这条评论？')) return
  drawer.loading = true
  try {
    await commentApi.remove(commentId)
    await loadComments()
    toast.info('评论已删除')
  } catch (error) {
    drawer.error = error instanceof ApiError ? error.message : String(error)
    toast.error(drawer.error)
  } finally {
    drawer.loading = false
  }
}

watch(() => props.video?.id, (videoId) => {
  if (videoId) void loadComments()
}, { immediate: true })
</script>

<template>
  <div class="fixed inset-0 z-50 flex justify-end bg-black/70 backdrop-blur-sm" @click.self="close">
    <aside class="grid h-full w-full max-w-md grid-rows-[auto_1fr_auto] border-l border-border bg-card shadow-2xl">
      <header class="flex items-center justify-between border-b border-border p-4">
        <div class="min-w-0">
          <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">Comments</p>
          <h2 class="truncate font-semibold">{{ video?.title ?? '评论' }}</h2>
        </div>
        <Button variant="ghost" size="icon" @click="close">
          <X class="size-4" />
        </Button>
      </header>

      <div class="overflow-y-auto p-4">
        <div v-if="drawer.loading && drawer.comments.length === 0" class="py-12 text-center text-sm text-muted-foreground">
          正在加载评论…
        </div>
        <div v-else-if="drawer.error && drawer.comments.length === 0" class="py-12 text-center text-sm text-destructive">
          {{ drawer.error }}
        </div>
        <div v-else-if="drawer.comments.length === 0" class="py-12 text-center text-sm text-muted-foreground">
          暂无评论，来留下第一条。
        </div>

        <div v-else class="grid gap-3">
          <article v-for="comment in drawer.comments" :key="comment.id" class="rounded-xl border border-border bg-background/45 p-4">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-medium">@{{ comment.username }}</p>
                <p class="mt-1 text-xs text-muted-foreground">{{ new Date(comment.created_at).toLocaleString() }}</p>
              </div>
              <Button
                v-if="canDeleteComment(comment)"
                variant="ghost"
                size="icon"
                class="size-8 text-muted-foreground hover:text-destructive"
                :disabled="drawer.loading"
                @click="deleteComment(comment.id)"
              >
                <Trash2 class="size-4" />
              </Button>
            </div>
            <p class="mt-3 whitespace-pre-wrap break-words text-sm leading-6 text-foreground/90">{{ comment.content }}</p>
          </article>
        </div>
      </div>

      <footer class="border-t border-border bg-background/35 p-4">
        <Textarea v-model="drawer.content" placeholder="分享你的看法…" :disabled="drawer.loading" />
        <div class="mt-3 flex items-center justify-between">
          <Button variant="ghost" size="sm" :disabled="drawer.loading" @click="loadComments">
            <RefreshCw class="size-4" />
            刷新
          </Button>
          <Button size="sm" :disabled="drawer.loading || !drawer.content.trim()" @click="publishComment">
            <Send class="size-4" />
            发送
          </Button>
        </div>
      </footer>
    </aside>
  </div>
</template>
