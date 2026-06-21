# 服务器部署手册

## 1. 前置条件

- 一台 Linux 服务器，安装 Docker Engine 与 Docker Compose Plugin。
- 两个解析到服务器公网 IP 的域名：
  - `APP_DOMAIN`：应用页面和 `/api/`
  - `MEDIA_DOMAIN`：MinIO 预签名媒体地址
- 防火墙仅开放 TCP 22、80、443。

## 2. 配置生产环境变量

复制 `.env.example` 为 `.env`，至少修改：

```dotenv
APP_DOMAIN=video.example.com
MEDIA_DOMAIN=media.example.com

MYSQL_ROOT_PASSWORD=<strong-random-password>
REDIS_PASSWORD=<strong-random-password>
RABBITMQ_USER=<random-user>
RABBITMQ_PASS=<strong-random-password>
JWT_SECRET=<at-least-32-random-bytes>

MINIO_ACCESS_KEY=<random-access-key>
MINIO_SECRET_KEY=<strong-random-password>
MINIO_BUCKET=feedsystem-media

ALLOW_PUBLIC_REGISTRATION=false
BOOTSTRAP_USERNAME=interviewer
BOOTSTRAP_PASSWORD=<private-demo-password>
```

不要提交 `.env`。演示密码只通过安全渠道提供给面试官。

## 3. 首次签发 HTTPS 证书

先启动不依赖证书的 HTTP bootstrap 网关：

```bash
set -a
source .env
set +a
docker compose -f docker-compose.prod.yml --profile bootstrap up -d gateway-bootstrap
```

使用 Certbot webroot 一次性签发两个域名：

```bash
docker compose -f docker-compose.prod.yml run --rm --entrypoint certbot certbot \
  certonly --webroot -w /var/www/certbot \
  --email you@example.com --agree-tos --no-eff-email \
  -d "$APP_DOMAIN" -d "$MEDIA_DOMAIN"
```

停止 bootstrap 网关：

```bash
docker compose -f docker-compose.prod.yml --profile bootstrap down
```

## 4. 启动生产服务

```bash
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml ps
```

验证：

```bash
curl -fsS "https://$APP_DOMAIN/api/healthz"
curl -I "https://$APP_DOMAIN/login"
```

MinIO Console、MySQL、Redis、RabbitMQ 和 Go API 均不应拥有公网端口。

## 5. 演示数据

1. 使用 `BOOTSTRAP_USERNAME` 和 `BOOTSTRAP_PASSWORD` 登录。
2. 通过发布页上传至少 3 个有使用权的 MP4 和封面。
3. 完成点赞、评论、关注操作，确保热榜、通知和个人主页有可展示数据。
4. 打开架构页检查说明与实际部署一致。

## 6. 证书续期

`certbot` 服务每 12 小时检查续期。证书更新后重载网关：

```bash
docker compose -f docker-compose.prod.yml exec gateway nginx -s reload
```

建议在宿主机增加每日执行上述 reload 的定时任务。

## 7. 备份与恢复

需要备份的 Docker volumes：

- `mysql_data`
- `redis_data`
- `rabbitmq_data`
- `minio_data`
- `letsencrypt`

MySQL 应额外定期执行逻辑备份：

```bash
docker compose -f docker-compose.prod.yml exec -T mysql \
  mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" > feedsystem.sql
```

MinIO 对象与 MySQL 元数据必须在相近时间点备份，避免 object key 与实际对象不一致。

## 8. 常见故障

- 视频 URL 指向内网：检查 `MINIO_PUBLIC_ENDPOINT=$MEDIA_DOMAIN` 和 `MINIO_PUBLIC_USE_SSL=true`。
- 预签名 URL 返回签名错误：Nginx 必须保留原始 `Host`，且应用与媒体域名不能混用。
- 上传无法恢复：检查 Redis 持久化和 `chunk_upload:*` 会话是否存在。
- 后端无法启动：依次检查 MySQL、Redis、RabbitMQ、MinIO healthcheck 和 bucket 初始化日志。
- SSE 无通知：检查浏览器 EventSource、RabbitMQ notification queue 和 Worker 日志。
