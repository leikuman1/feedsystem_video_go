<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { KeyRound } from '@lucide/vue'

import { ApiError } from '@/api/client'
import * as accountApi from '@/api/account'
import AppShell from '@/components/AppShell.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'

const auth = useAuthStore()
const toast = useToastStore()
const router = useRouter()
const busy = ref(false)
const form = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

async function submit() {
  if (busy.value) return
  if (!form.oldPassword || !form.newPassword) {
    toast.error('请填写完整密码信息')
    return
  }
  if (form.newPassword !== form.confirmPassword) {
    toast.error('两次输入的新密码不一致')
    return
  }

  busy.value = true
  try {
    await accountApi.changePassword(auth.claims?.username ?? '', form.oldPassword, form.newPassword)
    auth.clearTokens()
    toast.success('密码已修改，请重新登录')
    await router.replace('/login')
  } catch (error) {
    toast.error(error instanceof ApiError ? error.message : String(error))
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AppShell>
    <Card class="mx-auto max-w-lg">
      <CardHeader>
        <span class="mb-2 grid size-11 place-items-center rounded-xl bg-primary/15 text-primary"><KeyRound class="size-5" /></span>
        <CardTitle class="text-2xl">修改密码</CardTitle>
        <CardDescription>成功后后端会撤销 Access Token 与 Refresh Token。</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="grid gap-5" @submit.prevent="submit">
          <label class="grid gap-2 text-sm">
            <span class="text-muted-foreground">当前密码</span>
            <Input v-model="form.oldPassword" type="password" autocomplete="current-password" />
          </label>
          <label class="grid gap-2 text-sm">
            <span class="text-muted-foreground">新密码</span>
            <Input v-model="form.newPassword" type="password" autocomplete="new-password" />
          </label>
          <label class="grid gap-2 text-sm">
            <span class="text-muted-foreground">确认新密码</span>
            <Input v-model="form.confirmPassword" type="password" autocomplete="new-password" />
          </label>
          <Button type="submit" size="lg" :disabled="busy">{{ busy ? '提交中…' : '更新密码' }}</Button>
        </form>
      </CardContent>
    </Card>
  </AppShell>
</template>
