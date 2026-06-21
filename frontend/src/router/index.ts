import { createRouter, createWebHistory } from 'vue-router'

import * as accountApi from '@/api/account'
import { useAuthStore } from '@/stores/auth'
import AccountView from '@/views/AccountView.vue'
import ChangePasswordView from '@/views/ChangePasswordView.vue'
import HomeView from '@/views/HomeView.vue'
import HotView from '@/views/HotView.vue'
import LoginView from '@/views/LoginView.vue'
import MessageView from '@/views/MessageView.vue'
import SettingsView from '@/views/SettingsView.vue'
import UserProfileView from '@/views/UserProfileView.vue'
import VideoDetailView from '@/views/VideoDetailView.vue'
import VideoView from '@/views/VideoView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
    { path: '/', name: 'home', component: HomeView },
    { path: '/feed', redirect: '/' },
    { path: '/hot', name: 'hot', component: HotView },
    { path: '/video', name: 'video', component: VideoView },
    { path: '/video/:id', name: 'video-detail', component: VideoDetailView, props: true },
    { path: '/account', name: 'account', component: AccountView },
    { path: '/account/change-password', name: 'account-change-password', component: ChangePasswordView },
    { path: '/settings', name: 'settings', component: SettingsView },
    { path: '/u/:id', name: 'user-profile', component: UserProfileView, props: true },
    { path: '/messages', name: 'message-list', component: MessageView },
    { path: '/messages/:peerId', name: 'messages', component: MessageView },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

let refreshPromise: Promise<boolean> | null = null

function tokenIsUsable(auth: ReturnType<typeof useAuthStore>) {
  const expiresAt = auth.claims?.exp
  return !!auth.token && !!expiresAt && expiresAt * 1000 > Date.now() + 5_000
}

async function restoreSession(auth: ReturnType<typeof useAuthStore>) {
  if (tokenIsUsable(auth)) return true
  if (!auth.refreshToken) {
    auth.clearTokens()
    return false
  }
  if (!refreshPromise) {
    refreshPromise = accountApi.refresh(auth.refreshToken)
      .then((response) => {
        auth.setToken(response.token)
        return true
      })
      .catch(() => {
        auth.clearTokens()
        return false
      })
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
}

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.meta.public) {
    if (to.name === 'login' && await restoreSession(auth)) return { path: '/' }
    return true
  }
  if (await restoreSession(auth)) return true
  return { path: '/login', query: { redirect: to.fullPath } }
})

export default router
