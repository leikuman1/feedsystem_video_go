<script setup lang="ts">
import { nextTick, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  CheckCircle2,
  FileVideo,
  Image,
  RotateCcw,
  UploadCloud,
  X,
} from '@lucide/vue'
import gsap from 'gsap'
import SparkMD5 from 'spark-md5'

import { ApiError } from '@/api/client'
import type { Video } from '@/api/types'
import * as videoApi from '@/api/video'
import AppShell from '@/components/AppShell.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import { Textarea } from '@/components/ui/textarea'
import { useToastStore } from '@/stores/toast'

const router = useRouter()
const toast = useToastStore()
const videoInput = ref<HTMLInputElement | null>(null)
const coverInput = ref<HTMLInputElement | null>(null)
const successCard = ref<HTMLElement | null>(null)
const busy = ref(false)
const published = ref<Video | null>(null)
const activeUploadId = ref('')

const form = reactive({
  title: '',
  description: '',
  video: null as File | null,
  cover: null as File | null,
})

const preview = reactive({ videoUrl: '', coverUrl: '' })
const upload = reactive({
  stage: '等待选择文件',
  uploadedBytes: 0,
  totalBytes: 0,
  percent: 0,
  hashPercent: 0,
  resumedChunks: 0,
  totalChunks: 0,
  retryCount: 0,
})

const CHUNK_SIZE = 5 << 20
const MAX_CONCURRENT = 3
const MAX_RETRIES = 3

function setPreview(kind: 'video' | 'cover', file: File | null) {
  const key = kind === 'video' ? 'videoUrl' : 'coverUrl'
  if (preview[key]) URL.revokeObjectURL(preview[key])
  preview[key] = file ? URL.createObjectURL(file) : ''
}

watch(() => form.video, (file) => setPreview('video', file))
watch(() => form.cover, (file) => setPreview('cover', file))
onUnmounted(() => {
  setPreview('video', null)
  setPreview('cover', null)
})

function resetUploadState() {
  upload.stage = '等待上传'
  upload.uploadedBytes = 0
  upload.totalBytes = 0
  upload.percent = 0
  upload.hashPercent = 0
  upload.resumedChunks = 0
  upload.totalChunks = 0
  upload.retryCount = 0
  activeUploadId.value = ''
}

function selectVideo(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] ?? null
  if (file && (file.type !== 'video/mp4' || file.size > 200 * 1024 * 1024)) {
    toast.error('仅支持不超过 200MB 的 MP4 视频')
    input.value = ''
    return
  }
  form.video = file
}

function selectCover(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] ?? null
  if (file && (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type) || file.size > 10 * 1024 * 1024)) {
    toast.error('封面仅支持不超过 10MB 的 JPG、PNG 或 WebP')
    input.value = ''
    return
  }
  form.cover = file
}

function clearFile(kind: 'video' | 'cover') {
  if (kind === 'video') {
    form.video = null
    if (videoInput.value) videoInput.value.value = ''
  } else {
    form.cover = null
    if (coverInput.value) coverInput.value.value = ''
  }
}

async function computeFileMD5(file: File) {
  upload.stage = '计算文件指纹'
  const readSize = 2 << 20
  const spark = new SparkMD5.ArrayBuffer()
  for (let offset = 0; offset < file.size; offset += readSize) {
    const end = Math.min(offset + readSize, file.size)
    spark.append(await file.slice(offset, end).arrayBuffer())
    upload.hashPercent = Math.round((end / file.size) * 100)
  }
  return spark.end()
}

async function computeChunkMD5(blob: Blob) {
  const spark = new SparkMD5.ArrayBuffer()
  spark.append(await blob.arrayBuffer())
  return spark.end()
}

async function uploadVideoChunked(file: File) {
  const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
  upload.totalChunks = totalChunks
  upload.totalBytes = file.size
  const fileHash = await computeFileMD5(file)

  upload.stage = '初始化 MinIO Multipart'
  const initialized = await videoApi.initChunkUpload({
    filename: file.name,
    file_size: file.size,
    chunk_size: CHUNK_SIZE,
    total_chunks: totalChunks,
    file_hash: fileHash,
  })
  activeUploadId.value = initialized.upload_id
  const uploadedChunks = new Set(initialized.uploaded_chunks)
  upload.resumedChunks = uploadedChunks.size
  upload.uploadedBytes = [...uploadedChunks].reduce((sum, chunkIndex) => {
    const size = chunkIndex === totalChunks - 1 ? file.size - chunkIndex * CHUNK_SIZE : CHUNK_SIZE
    return sum + size
  }, 0)
  upload.percent = Math.round((upload.uploadedBytes / upload.totalBytes) * 100)

  const pending = Array.from({ length: totalChunks }, (_, index) => index)
    .filter((index) => !uploadedChunks.has(index))

  upload.stage = pending.length ? '上传视频分片' : '恢复完成，提交对象'
  let cursor = 0

  async function uploadOne(chunkIndex: number) {
    const start = chunkIndex * CHUNK_SIZE
    const end = Math.min(start + CHUNK_SIZE, file.size)
    const blob = file.slice(start, end)
    const hash = await computeChunkMD5(blob)
    let lastError: unknown

    for (let attempt = 0; attempt < MAX_RETRIES; attempt += 1) {
      try {
        await videoApi.uploadChunk(initialized.upload_id, chunkIndex, hash, blob)
        upload.uploadedBytes += blob.size
        upload.percent = Math.round((upload.uploadedBytes / upload.totalBytes) * 100)
        return
      } catch (error) {
        lastError = error
        upload.retryCount += 1
      }
    }
    throw lastError
  }

  await new Promise<void>((resolve, reject) => {
    let active = 0
    let stopped = false
    const schedule = () => {
      if (stopped) return
      if (cursor >= pending.length && active === 0) {
        resolve()
        return
      }
      while (active < MAX_CONCURRENT && cursor < pending.length) {
        const chunkIndex = pending[cursor++] as number
        active += 1
        uploadOne(chunkIndex)
          .then(() => {
            active -= 1
            schedule()
          })
          .catch((error) => {
            stopped = true
            reject(error)
          })
      }
    }
    schedule()
  })

  upload.stage = '提交 MinIO 对象'
  const response = await videoApi.completeChunkUpload(initialized.upload_id)
  activeUploadId.value = ''
  return response
}

async function publish() {
  if (busy.value) return
  const title = form.title.trim()
  if (!title || !form.video || !form.cover) {
    toast.error('请填写标题并选择视频与封面')
    return
  }

  busy.value = true
  published.value = null
  resetUploadState()
  try {
    const videoUpload = await uploadVideoChunked(form.video)
    upload.stage = '上传封面'
    const coverUpload = await videoApi.uploadCover(form.cover)
    upload.stage = '写入视频与 Outbox'
    published.value = await videoApi.publishVideo({
      title,
      description: form.description.trim(),
      play_object_key: videoUpload.object_key,
      cover_object_key: coverUpload.object_key,
    })
    upload.stage = '发布完成'
    upload.percent = 100
    toast.success('视频已发布')
    await nextTick()
    if (successCard.value && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      gsap.from(successCard.value, { opacity: 0, y: 20, duration: 0.5, ease: 'power2.out' })
    }
  } catch (error) {
    if (activeUploadId.value) {
      await videoApi.abortChunkUpload(activeUploadId.value).catch(() => undefined)
      activeUploadId.value = ''
    }
    upload.stage = '上传失败'
    toast.error(error instanceof ApiError ? error.message : String(error))
  } finally {
    busy.value = false
  }
}

function resetForm() {
  form.title = ''
  form.description = ''
  clearFile('video')
  clearFile('cover')
  published.value = null
  resetUploadState()
}
</script>

<template>
  <AppShell>
    <div class="mx-auto grid max-w-6xl gap-6 xl:grid-cols-[1fr_22rem]">
      <Card>
        <CardHeader>
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div>
              <Badge variant="outline" class="mb-3 border-primary/30 text-primary">MINIO MULTIPART</Badge>
              <CardTitle class="text-2xl">发布新视频</CardTitle>
              <CardDescription class="mt-2">
                浏览器计算 MD5，Go API 校验分片并直接写入 MinIO，Redis 保存断点状态。
              </CardDescription>
            </div>
            <Button variant="ghost" size="sm" :disabled="busy" @click="resetForm">
              <RotateCcw class="size-4" />
              重置
            </Button>
          </div>
        </CardHeader>
        <CardContent class="grid gap-6">
          <div class="grid gap-5 md:grid-cols-2">
            <label class="grid gap-2 text-sm md:col-span-2">
              <span class="text-muted-foreground">标题</span>
              <Input v-model="form.title" :disabled="busy" placeholder="一句清晰的标题，可包含 #话题" />
            </label>
            <label class="grid gap-2 text-sm md:col-span-2">
              <span class="text-muted-foreground">描述</span>
              <Textarea v-model="form.description" :disabled="busy" placeholder="补充视频内容与背景…" />
            </label>

            <div class="grid gap-3">
              <span class="text-sm text-muted-foreground">视频文件</span>
              <input ref="videoInput" class="hidden" type="file" accept="video/mp4" :disabled="busy" @change="selectVideo" />
              <button
                class="group grid min-h-44 place-items-center rounded-xl border border-dashed border-border bg-background/40 p-5 text-center transition hover:border-primary/50 hover:bg-primary/5"
                type="button"
                :disabled="busy"
                @click="videoInput?.click()"
              >
                <div>
                  <FileVideo class="mx-auto size-8 text-primary" />
                  <p class="mt-3 text-sm font-medium">{{ form.video?.name ?? '选择 MP4 视频' }}</p>
                  <p class="mt-1 text-xs text-muted-foreground">
                    {{ form.video ? `${(form.video.size / 1024 / 1024).toFixed(1)} MB` : '最大 200MB，5MB 分片' }}
                  </p>
                </div>
              </button>
              <Button v-if="form.video" variant="ghost" size="sm" :disabled="busy" @click="clearFile('video')">
                <X class="size-4" />
                移除视频
              </Button>
            </div>

            <div class="grid gap-3">
              <span class="text-sm text-muted-foreground">视频封面</span>
              <input ref="coverInput" class="hidden" type="file" accept="image/jpeg,image/png,image/webp" :disabled="busy" @change="selectCover" />
              <button
                class="group grid min-h-44 place-items-center overflow-hidden rounded-xl border border-dashed border-border bg-background/40 p-2 text-center transition hover:border-primary/50 hover:bg-primary/5"
                type="button"
                :disabled="busy"
                @click="coverInput?.click()"
              >
                <img v-if="preview.coverUrl" :src="preview.coverUrl" alt="封面预览" class="size-full max-h-56 rounded-lg object-cover" />
                <div v-else>
                  <Image class="mx-auto size-8 text-primary" />
                  <p class="mt-3 text-sm font-medium">选择封面图片</p>
                  <p class="mt-1 text-xs text-muted-foreground">JPG / PNG / WebP，最大 10MB</p>
                </div>
              </button>
              <Button v-if="form.cover" variant="ghost" size="sm" :disabled="busy" @click="clearFile('cover')">
                <X class="size-4" />
                移除封面
              </Button>
            </div>
          </div>

          <div v-if="busy || upload.percent > 0" class="rounded-xl border border-border bg-background/45 p-4">
            <div class="mb-3 flex items-center justify-between gap-3 text-sm">
              <span class="font-medium">{{ upload.stage }}</span>
              <span class="font-mono text-muted-foreground">{{ upload.percent }}%</span>
            </div>
            <Progress :model-value="upload.percent" />
            <div class="mt-3 flex flex-wrap gap-x-5 gap-y-1 text-xs text-muted-foreground">
              <span>哈希 {{ upload.hashPercent }}%</span>
              <span>分片 {{ upload.totalChunks }}</span>
              <span>恢复 {{ upload.resumedChunks }}</span>
              <span>重试 {{ upload.retryCount }}</span>
            </div>
          </div>

          <Button size="lg" :disabled="busy" class="w-full" @click="publish">
            <UploadCloud class="size-5" />
            {{ busy ? upload.stage : '开始上传并发布' }}
          </Button>
        </CardContent>
      </Card>

      <div class="grid content-start gap-6">
        <Card>
          <CardHeader>
            <CardTitle>本地预览</CardTitle>
            <CardDescription>上传前确认视频和封面是否匹配。</CardDescription>
          </CardHeader>
          <CardContent>
            <div class="aspect-[9/14] overflow-hidden rounded-xl border border-border bg-black">
              <video
                v-if="preview.videoUrl"
                class="size-full object-contain"
                :src="preview.videoUrl"
                :poster="preview.coverUrl"
                controls
                playsinline
                preload="metadata"
              />
              <div v-else class="grid size-full place-items-center text-center text-sm text-muted-foreground">
                <div>
                  <FileVideo class="mx-auto mb-3 size-8" />
                  尚未选择视频
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <div v-if="published" ref="successCard">
          <Card class="border-emerald-500/30">
            <CardHeader>
              <span class="mb-2 grid size-10 place-items-center rounded-full bg-emerald-500/15 text-emerald-400">
                <CheckCircle2 class="size-5" />
              </span>
              <CardTitle>发布成功</CardTitle>
              <CardDescription>{{ published.title }}</CardDescription>
            </CardHeader>
            <CardContent class="grid gap-2">
              <Button @click="router.push(`/video/${published.id}`)">查看视频</Button>
              <Button variant="outline" @click="resetForm">继续发布</Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  </AppShell>
</template>
