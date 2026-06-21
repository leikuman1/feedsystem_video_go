<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Camera, LogOut, Save, ShieldCheck, UserRound } from '@lucide/vue'

import { ApiError } from '@/api/client'
import * as accountApi from '@/api/account'
import type { Account } from '@/api/types'
import AppShell from '@/components/AppShell.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'

const auth = useAuthStore()
const toast = useToastStore()
const router = useRouter()
const busy = ref(false)
const avatarInput = ref<HTMLInputElement | null>(null)
const profile = ref<Account | null>(null)
const accountId = computed(() => auth.claims?.account_id ?? 0)
const form = reactive({ username: '', bio: '' })

async function load() {
  if (!accountId.value) return
  try {
    profile.value = await accountApi.findById(accountId.value)
    form.username = profile.value.username
    form.bio = profile.value.bio ?? ''
  } catch (error) {
    toast.error(error instanceof ApiError ? error.message : String(error))
  }
}

async function saveProfile() {
  if (busy.value) return
  busy.value = true
  try {
    const username = form.username.trim()
    if (username && username !== profile.value?.username) {
      const response = await accountApi.rename(username)
      auth.setToken(response.token)
    }
    if (form.bio.trim()) await accountApi.updateProfile({ bio: form.bio.trim() })
    toast.success('资料已更新')
    await load()
  } catch (error) {
    toast.error(error instanceof ApiError ? error.message : String(error))
  } finally {
    busy.value = false
  }
}

async function uploadAvatar(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  busy.value = true
  try {
    const response = await accountApi.uploadAvatar(file)
    if (profile.value) profile.value.avatar_url = response.avatar_url
    toast.success('头像已更新')
  } catch (error) {
    toast.error(error instanceof ApiError ? error.message : String(error))
  } finally {
    busy.value = false
    input.value = ''
  }
}

async function logout() {
  try {
    await accountApi.logout()
  } catch {
    // Local session still needs to be cleared.
  }
  auth.clearTokens()
  await router.replace('/login')
}

onMounted(load)
</script>

<template>
  <AppShell>
    <div class="mx-auto grid max-w-5xl gap-6 lg:grid-cols-[1fr_20rem]">
      <Card>
        <CardHeader>
          <span class="mb-2 grid size-11 place-items-center rounded-xl bg-primary/15 text-primary"><UserRound class="size-5" /></span>
          <CardTitle class="text-2xl">个人资料</CardTitle>
          <CardDescription>头像文件存储在 MinIO，数据库只保存稳定 object key。</CardDescription>
        </CardHeader>
        <CardContent class="grid gap-6">
          <div class="flex items-center gap-4 rounded-xl border border-border bg-background/40 p-4">
            <UserAvatar :username="profile?.username ?? form.username" :id="accountId" :src="profile?.avatar_url" :size="72" />
            <div>
              <input ref="avatarInput" class="hidden" type="file" accept="image/jpeg,image/png,image/webp" @change="uploadAvatar" />
              <Button variant="outline" :disabled="busy" @click="avatarInput?.click()">
                <Camera class="size-4" />
                更换头像
              </Button>
              <p class="mt-2 text-xs text-muted-foreground">JPG、PNG 或 WebP，最大 10MB。</p>
            </div>
          </div>

          <label class="grid gap-2 text-sm">
            <span class="text-muted-foreground">用户名</span>
            <Input v-model="form.username" :disabled="busy" />
          </label>
          <label class="grid gap-2 text-sm">
            <span class="text-muted-foreground">个人简介</span>
            <Textarea v-model="form.bio" :disabled="busy" placeholder="介绍一下这个演示账号…" />
          </label>
          <Button :disabled="busy" @click="saveProfile">
            <Save class="size-4" />
            {{ busy ? '保存中…' : '保存修改' }}
          </Button>
        </CardContent>
      </Card>

      <div class="grid content-start gap-6">
        <Card>
          <CardHeader>
            <ShieldCheck class="mb-2 size-5 text-primary" />
            <CardTitle>账号安全</CardTitle>
            <CardDescription>修改密码会撤销当前 Token。</CardDescription>
          </CardHeader>
          <CardContent class="grid gap-2">
            <Button variant="outline" @click="router.push('/account/change-password')">修改密码</Button>
            <Button variant="destructive" @click="logout">
              <LogOut class="size-4" />
              退出登录
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  </AppShell>
</template>
