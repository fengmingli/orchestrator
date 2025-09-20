# 部署指南

## 部署方式

工作流编排系统支持多种部署方式，可根据实际需求选择合适的部署方案。

### 快速部署（推荐）

使用项目提供的脚本进行一键部署：

```bash
# 克隆项目
git clone <repository-url>
cd orchestrator

# 一键启动 (推荐使用新路径)
./scripts/dev/start.sh

# 兼容性方式 (会自动重定向)
./start.sh

# 访问应用
# 前端: http://localhost:3000
# 后端API: http://localhost:8080
```

### 手动部署

#### 1. 环境准备

**系统要求**:
- Linux / macOS / Windows
- Go 1.21+
- Node.js 18+
- MySQL 8.0+ (可选)

**依赖安装**:
```bash
# 安装 Go (以 Linux 为例)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 安装 Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# 安装 MySQL (可选)
sudo apt-get install mysql-server
```

#### 2. 后端部署

```bash
# 编译后端
cd cmd/web
go build -o ../../bin/web main.go
cd ../..

# 配置环境变量
export DB_HOST=127.0.0.1
export DB_PORT=3306
export DB_USERNAME=root
export DB_PASSWORD=root123456
export DB_DATABASE=orchestrator
export SERVER_PORT=8080

# 启动后端服务
./bin/web
```

#### 3. 前端部署

```bash
# 安装依赖
cd web/frontend
npm install

# 构建生产版本
npm run build

# 启动前端服务
npm run preview
# 或使用 nginx 等 web 服务器托管 dist 目录
```

## 配置说明

### 环境变量

系统支持通过环境变量进行配置：

#### 数据库配置
```bash
# MySQL 配置 (优先使用)
DB_HOST=127.0.0.1              # 数据库主机
DB_PORT=3306                   # 数据库端口
DB_USERNAME=root               # 数据库用户名
DB_PASSWORD=root123456         # 数据库密码
DB_DATABASE=orchestrator       # 数据库名称
DB_CHARSET=utf8mb4             # 字符集

# 连接池配置
DB_MAX_IDLE_CONNS=10          # 最大空闲连接数
DB_MAX_OPEN_CONNS=100         # 最大打开连接数
DB_CONN_MAX_LIFETIME=3600     # 连接最大生存时间(秒)
```

#### 服务器配置
```bash
SERVER_HOST=0.0.0.0           # 服务监听地址
SERVER_PORT=8080              # 服务监听端口
LOG_LEVEL=info                # 日志级别 (debug,info,warn,error)
LOG_FORMAT=json               # 日志格式 (json,text)
```

#### 执行引擎配置
```bash
EXECUTOR_POOL_SIZE=10         # 执行器线程池大小
EXECUTOR_QUEUE_SIZE=100       # 执行队列大小
EXECUTOR_TIMEOUT=3600         # 默认执行超时时间(秒)
```

### 配置文件

也可以使用配置文件进行设置，在项目根目录创建 `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "127.0.0.1"
  port: 3306
  username: "root"
  password: "root123456"
  database: "orchestrator"
  charset: "utf8mb4"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600

logging:
  level: "info"
  format: "json"
  
executor:
  pool_size: 10
  queue_size: 100
  timeout: 3600
```

## Docker 部署

### 使用 Docker Compose

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root123456
      MYSQL_DATABASE: orchestrator
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    restart: unless-stopped

  orchestrator-backend:
    build: .
    environment:
      DB_HOST: mysql
      DB_PASSWORD: root123456
      DB_DATABASE: orchestrator
    ports:
      - "8080:8080"
    depends_on:
      - mysql
    restart: unless-stopped

  orchestrator-frontend:
    build: ./web/frontend
    ports:
      - "3000:3000"
    depends_on:
      - orchestrator-backend
    restart: unless-stopped

volumes:
  mysql_data:
```

启动服务：
```bash
docker-compose up -d
```

### 单独构建镜像

#### 后端镜像
创建 `Dockerfile`:
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bin/web cmd/web/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bin/web .
EXPOSE 8080
CMD ["./web"]
```

构建和运行：
```bash
docker build -t orchestrator-backend .
docker run -d -p 8080:8080 --name orchestrator-backend orchestrator-backend
```

#### 前端镜像
在 `web/frontend/` 目录创建 `Dockerfile`:
```dockerfile
FROM node:18-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 3000
CMD ["nginx", "-g", "daemon off;"]
```

## 生产环境部署

### 1. 系统优化

#### 操作系统配置
```bash
# 调整文件描述符限制
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# 调整内核参数
echo "net.core.rmem_max = 268435456" >> /etc/sysctl.conf
echo "net.core.wmem_max = 268435456" >> /etc/sysctl.conf
sysctl -p
```

#### MySQL 优化
```sql
-- 创建数据库
CREATE DATABASE orchestrator DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建用户
CREATE USER 'orchestrator'@'%' IDENTIFIED BY 'secure_password';
GRANT ALL PRIVILEGES ON orchestrator.* TO 'orchestrator'@'%';
FLUSH PRIVILEGES;
```

`/etc/mysql/mysql.conf.d/mysqld.cnf`:
```ini
[mysqld]
# 基础配置
max_connections = 1000
innodb_buffer_pool_size = 1G
innodb_log_file_size = 256M
innodb_flush_log_at_trx_commit = 2

# 字符集
character-set-server = utf8mb4
collation-server = utf8mb4_unicode_ci
```

### 2. 监控和日志

#### 日志配置
```yaml
logging:
  level: "info"
  format: "json"
  output: "/var/log/orchestrator/app.log"
  max_size: 100    # MB
  max_age: 30      # days
  max_backups: 10
```

#### 健康检查
```bash
# 检查后端服务
curl http://localhost:8080/health

# 检查数据库连接
curl http://localhost:8080/api/v1/health/db
```

### 3. 负载均衡

#### Nginx 配置
```nginx
upstream orchestrator_backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;  # 多实例
}

server {
    listen 80;
    server_name orchestrator.example.com;
    
    location /api/ {
        proxy_pass http://orchestrator_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
    
    location / {
        root /var/www/orchestrator;
        try_files $uri $uri/ /index.html;
    }
}
```

### 4. 安全配置

#### HTTPS 配置
```nginx
server {
    listen 443 ssl http2;
    server_name orchestrator.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    
    # 其他配置...
}
```

#### 防火墙配置
```bash
# 仅允许必要端口
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable
```

## 运维管理

### 启动和停止

```bash
# 启动服务 (推荐使用新路径)
./scripts/dev/start.sh

# 停止服务 (推荐使用新路径)  
./scripts/dev/stop.sh

# 重启服务
./scripts/dev/stop.sh && ./scripts/dev/start.sh

# 兼容性方式 (会自动重定向)
./start.sh   # 启动
./stop.sh    # 停止

# 查看服务状态
ps aux | grep orchestrator
```

### 备份和恢复

#### 数据备份
```bash
# MySQL 备份
mysqldump -u root -p orchestrator > backup_$(date +%Y%m%d_%H%M%S).sql

# 配置文件备份
tar -czf config_backup_$(date +%Y%m%d).tar.gz config.yaml
```

#### 数据恢复
```bash
# MySQL 恢复
mysql -u root -p orchestrator < backup_20250920_120000.sql
```

### 日志管理

```bash
# 查看应用日志
tail -f web.log

# 查看错误日志
grep "ERROR" web.log

# 日志轮转
logrotate /etc/logrotate.d/orchestrator
```

### 性能监控

```bash
# CPU 和内存使用
top -p $(pgrep orchestrator)

# 网络连接
netstat -tulpn | grep :8080

# 数据库连接
mysql -e "SHOW PROCESSLIST;"
```

## 故障排除

### 常见问题

#### 1. 端口占用
```bash
# 查找占用端口的进程
lsof -i :8080
kill -9 <PID>
```

#### 2. 数据库连接失败
```bash
# 检查数据库服务
systemctl status mysql

# 检查连接
mysql -h 127.0.0.1 -u root -p
```

#### 3. 内存不足
```bash
# 查看内存使用
free -h

# 增加交换空间
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

### 日志分析

```bash
# 查看错误统计
grep "ERROR" web.log | wc -l

# 分析响应时间
grep "duration" web.log | awk '{print $NF}' | sort -n

# 查看访问频率
grep "GET\|POST" web.log | awk '{print $1}' | sort | uniq -c | sort -nr
```

---

> 生产环境部署建议定期备份数据，监控系统状态，及时处理异常。