# 🚀 New API 生产环境部署指南

本指南专注于使用 Docker Compose 在生产环境部署 New API。

## 📋 部署前准备

### 服务器要求

- **操作系统**: Linux (Ubuntu 20.04+ / Debian 11+ / CentOS 8+ 推荐)
- **内存**: 最低 4GB，推荐 8GB+
- **磁盘**: 最低 50GB SSD
- **Docker**: 20.10+
- **Docker Compose**: 2.0+

### 安装 Docker 和 Docker Compose

```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sh

# 启动 Docker 服务
sudo systemctl start docker
sudo systemctl enable docker

# 验证安装
docker --version
docker-compose --version
```

## 🔧 部署步骤

### 1. 克隆项目或上传代码

```bash
# 方式一：从 Git 克隆
git clone https://github.com/QuantumNous/new-api.git
cd new-api

# 方式二：上传到服务器
# 使用 scp 或 rsync 上传项目到服务器
rsync -avz --progress ./ user@server:/opt/new-api/
```

### 2. 修改 docker-compose.yml 配置

编辑 `docker-compose.yml` 文件，**必须修改**以下配置：

```bash
nano docker-compose.yml
```

#### 必须修改的配置项：

```yaml
# 1. 数据库密码（第 29 行）
- SQL_DSN=postgresql://root:YOUR_STRONG_PASSWORD@postgres:5432/new-api

# 2. PostgreSQL 密码（第 63 行）
POSTGRES_PASSWORD: YOUR_STRONG_PASSWORD

# 3. 多机部署时设置 SESSION_SECRET（第 36 行）
- SESSION_SECRET=YOUR_RANDOM_SESSION_SECRET  # 取消注释并修改

# 4. 使用 Redis 时设置 CRYPTO_SECRET（可选）
# - CRYPTO_SECRET=YOUR_RANDOM_CRYPTO_SECRET
```

#### 生成随机密钥：

```bash
# 生成 SESSION_SECRET
openssl rand -base64 32

# 生成 CRYPTO_SECRET
openssl rand -base64 32

# 生成数据库密码
openssl rand -hex 16
```

### 3. 可选配置调整

根据实际需求调整以下配置：

#### 切换到 MySQL

如需使用 MySQL 而非 PostgreSQL：

```yaml
# 1. 注释掉 PostgreSQL 配置（第 29、57-68 行）
# 2. 取消注释 MySQL 配置（第 30、70-80 行）
# 3. 修改 depends_on（第 45 行）
# 4. 修改 volumes（第 84 行）
```

#### 调整端口映射

```yaml
ports:
  - "3000:3000" # 修改为你需要的端口，如 "8080:3000"
```

#### 性能优化配置

```yaml
environment:
  - STREAMING_TIMEOUT=300 # 流式超时（秒）
  - STREAM_SCANNER_MAX_BUFFER_MB=128 # 增加缓冲区
  - BATCH_UPDATE_ENABLED=true # 启用批量更新
  - BATCH_UPDATE_INTERVAL=5 # 批量更新间隔（秒）
```

### 4. 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 查看启动日志
docker-compose logs -f new-api

# 查看所有容器状态
docker-compose ps
```

### 5. 验证部署

```bash
# 检查服务健康状态
curl http://localhost:3000/api/status

# 预期返回
{"success":true,"message":"success"}
```

访问 `http://服务器IP:3000` 即可看到登录界面。

**默认管理员账号**：

- 用户名: `root`
- 密码: `123456`

⚠️ **首次登录后请立即修改密码！**

## 🌐 配置 Nginx 反向代理（推荐）

### 1. 安装 Nginx

```bash
sudo apt update
sudo apt install nginx
```

### 2. 创建 Nginx 配置

创建文件 `/etc/nginx/sites-available/new-api.conf`:

```nginx
server {
    listen 80;
    server_name your-domain.com;  # 修改为你的域名

    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;  # 修改为你的域名

    # SSL 证书配置
    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    # SSL 安全配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # 日志
    access_log /var/log/nginx/new-api.access.log;
    error_log /var/log/nginx/new-api.error.log;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket 支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
}
```

### 3. 配置 SSL 证书

使用 Let's Encrypt 免费证书：

```bash
# 安装 Certbot
sudo apt install certbot python3-certbot-nginx

# 获取证书并自动配置 Nginx
sudo certbot --nginx -d your-domain.com

# 证书自动续期
sudo certbot renew --dry-run
```

### 4. 启用配置

```bash
# 创建软链接
sudo ln -s /etc/nginx/sites-available/new-api.conf /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重载 Nginx
sudo systemctl reload nginx
```

## 🔧 常用管理命令

### 服务管理

```bash
# 启动服务
docker-compose up -d

# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 重启特定服务
docker-compose restart new-api

# 查看日志
docker-compose logs -f new-api

# 查看所有容器状态
docker-compose ps
```

### 更新应用

```bash
# 1. 拉取最新镜像
docker-compose pull

# 2. 重启服务
docker-compose up -d

# 3. 清理旧镜像
docker image prune -f
```

### 数据备份

```bash
# 备份 PostgreSQL 数据库
docker exec postgres pg_dump -U root new-api > backup-$(date +%Y%m%d).sql

# 备份数据目录
tar -czf data-backup-$(date +%Y%m%d).tar.gz ./data

# 备份配置文件
cp docker-compose.yml docker-compose.yml.backup
```

### 数据恢复

```bash
# 恢复 PostgreSQL 数据库
cat backup.sql | docker exec -i postgres psql -U root -d new-api

# 恢复数据目录
tar -xzf data-backup.tar.gz
```

## 📊 监控和维护

### 查看资源使用

```bash
# 查看容器资源使用情况
docker stats

# 查看磁盘使用
df -h
du -sh ./data ./logs
```

### 日志管理

```bash
# 查看实时日志
docker-compose logs -f new-api

# 查看最近 100 行日志
docker-compose logs --tail=100 new-api

# 导出日志
docker-compose logs new-api > new-api.log
```

### 清理磁盘空间

```bash
# 清理未使用的容器
docker container prune -f

# 清理未使用的镜像
docker image prune -a -f

# 清理未使用的卷
docker volume prune -f

# 一键清理所有未使用资源
docker system prune -a --volumes -f
```

## 🔐 安全加固

### 1. 防火墙配置

```bash
# 启用 UFW 防火墙
sudo ufw enable

# 只开放必要端口
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS

# 禁止直接访问应用端口（如果使用 Nginx）
sudo ufw deny 3000/tcp

# 查看规则
sudo ufw status
```

### 2. 修改默认端口

编辑 `docker-compose.yml`:

```yaml
ports:
  - "127.0.0.1:3000:3000" # 只允许本地访问
```

### 3. 数据库安全

```yaml
# 不要暴露数据库端口到外网
postgres:
  # ports:  # 注释掉这行
  #   - "5432:5432"
```

### 4. 定期更新

```bash
# 定期更新系统
sudo apt update && sudo apt upgrade -y

# 定期更新 Docker 镜像
docker-compose pull
docker-compose up -d
```

## 🐛 故障排查

### 服务无法启动

```bash
# 查看详细日志
docker-compose logs new-api

# 查看容器状态
docker-compose ps

# 检查端口占用
sudo lsof -i :3000

# 重新构建并启动
docker-compose up -d --force-recreate
```

### 数据库连接失败

```bash
# 检查数据库容器状态
docker-compose ps postgres

# 查看数据库日志
docker-compose logs postgres

# 测试数据库连接
docker exec -it postgres psql -U root -d new-api
```

### 内存不足

```bash
# 查看系统内存
free -h

# 增加 swap
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

### 磁盘空间不足

```bash
# 查看磁盘使用
df -h

# 清理 Docker 资源
docker system prune -a --volumes -f

# 清理日志
sudo journalctl --vacuum-time=7d
```

## 📝 环境变量配置参考

### 必需配置

| 变量名           | 说明                     | 默认值         |
| ---------------- | ------------------------ | -------------- |
| `SESSION_SECRET` | 会话密钥（多机部署必须） | -              |
| `CRYPTO_SECRET`  | 加密密钥（Redis 必须）   | SESSION_SECRET |
| `SQL_DSN`        | 数据库连接字符串         | -              |

### 可选配置

| 变量名                 | 说明               | 默认值        |
| ---------------------- | ------------------ | ------------- |
| `REDIS_CONN_STRING`    | Redis 连接字符串   | -             |
| `PORT`                 | 监听端口           | 3000          |
| `TZ`                   | 时区               | Asia/Shanghai |
| `STREAMING_TIMEOUT`    | 流式超时（秒）     | 300           |
| `BATCH_UPDATE_ENABLED` | 批量更新           | false         |
| `ERROR_LOG_ENABLED`    | 错误日志           | false         |
| `SYNC_FREQUENCY`       | 缓存同步频率（秒） | 60            |

更多配置请参考：[官方文档 - 环境变量](https://docs.newapi.pro/installation/environment-variables)

## 📞 获取帮助

- 📖 官方文档: https://docs.newapi.pro/
- 🐛 问题反馈: https://github.com/Calcium-Ion/new-api/issues
- 💬 社区交流: 参考官方文档中的交流渠道

## ✅ 部署检查清单

- [ ] 已安装 Docker 和 Docker Compose
- [ ] 已修改数据库密码
- [ ] 已设置 SESSION_SECRET（多机部署）
- [ ] 已配置防火墙
- [ ] 已配置 HTTPS（生产环境必须）
- [ ] 已修改默认管理员密码
- [ ] 已配置数据库备份
- [ ] 已配置日志轮转
- [ ] 已设置监控告警

---

**祝部署顺利！🎉**
