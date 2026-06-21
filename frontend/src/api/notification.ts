import { postJson } from './client'
import type { Notification } from './types'

export function listNotifications() {
  return postJson<{ notifications: Notification[] }>('/notification/list', {}, { authRequired: true })
}

export function unreadCount() {
  return postJson<{ count: number }>('/notification/unreadCount', {}, { authRequired: true })
}

export function markRead(id?: number) {
  return postJson<{ message: string }>('/notification/markRead', id ? { id } : {}, { authRequired: true })
}
