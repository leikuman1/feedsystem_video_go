<script setup lang="ts">
import {
  ArrowRight,
  Boxes,
  Cable,
  Cloud,
  Database,
  Gauge,
  HardDrive,
  Layers3,
  LockKeyhole,
  Network,
  Radio,
  Server,
} from '@lucide/vue'

import AppShell from '@/components/AppShell.vue'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const services = [
  { name: 'Nginx', role: 'HTTPS / SPA / API 入口', icon: Network },
  { name: 'Vue 3', role: '沉浸式产品界面', icon: Layers3 },
  { name: 'Go API', role: '鉴权与业务编排', icon: Server },
  { name: 'MySQL', role: '业务事实数据', icon: Database },
  { name: 'Redis', role: 'Feed、缓存、上传会话', icon: Gauge },
  { name: 'RabbitMQ', role: '异步事件与通知', icon: Radio },
  { name: 'Worker', role: '消费、重试与投影', icon: Boxes },
  { name: 'MinIO', role: '私有媒体对象存储', icon: HardDrive },
]

const flows = [
  {
    title: '视频上传',
    summary: '请求 → MD5 校验 → MinIO Multipart → object key → 发布事务',
    steps: ['浏览器并发分片', 'Go API 校验 JWT / MD5', 'MinIO Multipart Parts', 'Redis 保存 ETag 与断点', '完成对象并写入 Video + Outbox'],
  },
  {
    title: 'Feed 查询',
    summary: '请求 → Redis 时间线 → 多级实体缓存 → MySQL 降级 → 游标响应',
    steps: ['强 JWT 鉴权', 'Redis ZSET 热时间线', 'L1 本地缓存', 'L2 Redis 实体缓存', 'L3 MySQL 与 singleflight 回填'],
  },
  {
    title: '发布一致性',
    summary: 'Video 与 Outbox 同事务 → Poller → RabbitMQ → Redis Feed 投影',
    steps: ['MySQL 事务提交', 'Outbox pending', '轮询投递 RabbitMQ', 'Consumer ACK / 重连', 'Redis 时间线最终一致'],
  },
]

const improvements = [
  '本地磁盘迁移为 MinIO 私有对象存储',
  'Multipart 分片直写、断点续传、ETag 会话和主动终止',
  '数据库只保存 object key，响应阶段生成预签名 URL',
  '私有演示账号门禁与公网注册关闭',
  'Vue 3 + shadcn-vue + GSAP 全站视觉重构',
  'Docker Compose / Nginx / HTTPS 生产展示方案',
]
</script>

<template>
  <AppShell>
    <div class="mx-auto grid max-w-7xl gap-6">
      <Card class="overflow-hidden">
        <CardHeader class="relative border-b border-border bg-[radial-gradient(circle_at_15%_20%,oklch(0.7_0.17_35/.16),transparent_35%)] py-10">
          <Badge variant="outline" class="mb-4 w-fit border-primary/30 text-primary">INTERVIEW ARCHITECTURE</Badge>
          <CardTitle class="max-w-3xl text-3xl leading-tight md:text-5xl">短视频 Feed 系统架构与 dev 分支改造</CardTitle>
          <CardDescription class="mt-4 max-w-3xl text-base leading-7">
            面向面试演示，重点展示 MinIO 媒体存储、分片上传、Feed 缓存、Outbox 消息链路和生产部署方案。
          </CardDescription>
        </CardHeader>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>服务拓扑</CardTitle>
          <CardDescription>Nginx 是唯一公网入口，其余服务仅在 Docker 网络内互通。</CardDescription>
        </CardHeader>
        <CardContent>
          <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <div v-for="service in services" :key="service.name" class="rounded-xl border border-border bg-background/45 p-4">
              <component :is="service.icon" class="mb-4 size-5 text-primary" />
              <h3 class="font-semibold">{{ service.name }}</h3>
              <p class="mt-1 text-sm text-muted-foreground">{{ service.role }}</p>
            </div>
          </div>
          <div class="mt-5 flex flex-wrap items-center justify-center gap-2 text-xs text-muted-foreground">
            <Badge variant="outline"><LockKeyhole class="mr-1 size-3" />HTTPS</Badge>
            <ArrowRight class="size-4" />
            <Badge variant="outline"><Cable class="mr-1 size-3" />Go API</Badge>
            <ArrowRight class="size-4" />
            <Badge variant="outline"><Cloud class="mr-1 size-3" />内部依赖</Badge>
          </div>
        </CardContent>
      </Card>

      <div class="grid gap-6 lg:grid-cols-3">
        <Card v-for="flow in flows" :key="flow.title">
          <CardHeader>
            <CardTitle>{{ flow.title }}</CardTitle>
            <CardDescription class="leading-6">{{ flow.summary }}</CardDescription>
          </CardHeader>
          <CardContent>
            <ol class="grid gap-3">
              <li v-for="(step, index) in flow.steps" :key="step" class="flex gap-3">
                <span class="grid size-7 shrink-0 place-items-center rounded-full bg-primary/15 font-mono text-xs text-primary">{{ index + 1 }}</span>
                <span class="pt-1 text-sm text-foreground/80">{{ step }}</span>
              </li>
            </ol>
          </CardContent>
        </Card>
      </div>

      <div>
        <Card class="border-primary/25">
          <CardHeader>
            <CardTitle>dev 分支核心改造</CardTitle>
            <CardDescription>可通过提交记录、测试和部署结果验证。</CardDescription>
          </CardHeader>
          <CardContent class="grid gap-3 md:grid-cols-2">
            <div v-for="item in improvements" :key="item" class="rounded-lg border border-primary/20 bg-primary/5 p-3 text-sm">{{ item }}</div>
          </CardContent>
        </Card>
      </div>
    </div>
  </AppShell>
</template>
