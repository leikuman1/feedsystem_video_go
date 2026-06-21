# 面试演示与讲解指南

## 5 分钟演示顺序

1. 登录页：说明私有演示账号和公网注册关闭。
2. 推荐 Feed：展示沉浸播放、游标加载、点赞、关注和评论。
3. 发布页：上传一个视频，指出 MD5、并发分片、Redis 断点状态和 MinIO Multipart。
4. 热榜与通知：展示 Redis 时间窗口和 RabbitMQ/SSE 链路。
5. 架构页：说明服务拓扑、Outbox 和个人改造边界。

## 30 秒项目介绍

这是一个 Go + Vue 的短视频 Feed 系统。原项目具备账号、互动、Feed、RabbitMQ Worker 和 Redis 缓存；我在 dev 分支完成了 MinIO 对象存储迁移，把分片从本地磁盘合并改为后端直写 Multipart，并让数据库只保存 object key、响应阶段生成预签名 URL。同时增加了私有演示门禁、生产 HTTPS 编排和全站界面重构。

## 2 分钟核心链路

### 上传

浏览器先计算文件 MD5，调用初始化接口。后端创建 MinIO Multipart Upload，并把业务 upload ID、MinIO upload ID、object key、已完成分片和 ETag 存入 Redis。每个分片经 JWT、大小和 MD5 校验后直接写 MinIO；完成时按 part number 提交 ETag 列表，不再占用 API 容器磁盘做合并。发布接口只接受当前账号前缀下的 object key。

### Feed

最新 Feed 优先查 Redis ZSET 时间线，实体按 L1 进程缓存、L2 Redis、L3 MySQL 获取。缓存击穿使用 singleflight 合并同一实体的回源请求，Redis 不可用时降级数据库。分页使用时间或复合游标，不使用不稳定的 offset。

### 一致性

视频记录和 Outbox 消息在同一个 MySQL 事务提交。Poller 将 pending 事件投递 RabbitMQ，Consumer 更新 Redis Feed 投影，因此数据库发布与 Feed 可见性是最终一致。RabbitMQ 不可用时 Outbox 记录仍保留，可以恢复后继续投递。

## 深挖问题

### 为什么不让浏览器直接预签名上传 MinIO？

本次选择浏览器 → Go API → MinIO，是为了复用现有 JWT、分片 MD5、Redis 会话和限流，同时把一周改造范围控制在可测试、可部署的边界。代价是 API 仍承担上传带宽；更大规模下可以演进为后端签发预签名 part URL。

### 预签名 URL 为什么不存数据库？

预签名 URL 有过期时间且绑定域名。持久化它会造成缓存污染和域名迁移困难。数据库与 Redis 只保存稳定 object key，对外响应时生成短期 URL；旧数据没有 object key 时回退原 URL。

### Redis 丢失会怎样？

Feed 可以降级 MySQL，但正在进行的分片上传会失去断点状态。MinIO 未完成 Multipart 会由生命周期规则清理，用户重新初始化上传。Redis AOF 能降低会话丢失概率，但上传会话不是业务事实数据。

### Outbox 是否保证“恰好一次”？

不保证。它解决数据库记录和待投递事件的原子性，投递仍是至少一次语义。消费者需要幂等；当前时间线使用视频 ID 作为 ZSET member，重复投递会覆盖同一成员，而不是产生重复视频。

## 个人改造边界

原项目能力：

- 账号、互动、Feed、私信和通知基础业务
- Redis 缓存、RabbitMQ Worker、Outbox 和 SSE
- 原始分片上传与断点续传

本次 dev 分支完成：

- MinIO Storage 抽象和私有 bucket
- Multipart 直写、ETag 会话、主动终止和相关测试
- object key 数据模型、预签名 URL 和旧数据回退
- 私有演示账号和全局鉴权门禁
- Vue 3 / shadcn-vue / GSAP 界面重构
- Nginx 双域名 HTTPS 与生产 Compose
