# 脚本工具集

工作流编排系统提供了完整的脚本工具集，用于开发、构建、测试、部署和运维。

## 📁 脚本目录结构

```
scripts/
├── dev/           # 开发环境脚本
├── build/         # 构建脚本
├── deploy/        # 部署脚本
├── test/          # 测试脚本
├── utils/         # 工具脚本
└── README.md      # 本文档
```

## 🚀 开发环境脚本 (dev/)

### start.sh - 启动开发环境
**用途**: 一键启动完整的开发环境
**特性**:
- 自动检查运行环境 (Go, Node.js, npm)
- 编译后端服务
- 安装前端依赖
- 启动后端API服务
- 启动前端开发服务器
- 健康检查和状态监控

```bash
# 启动开发环境
./scripts/dev/start.sh

# 访问服务
# 前端: http://localhost:3000
# 后端: http://localhost:8080
```

### stop.sh - 停止开发环境
**用途**: 停止所有开发环境服务
**特性**:
- 优雅停止后端服务
- 停止前端开发服务器
- 清理进程和端口占用
- 可选清理日志文件

```bash
# 停止所有服务
./scripts/dev/stop.sh
```

## 🔨 构建脚本 (build/)

### build.sh - 生产构建
**用途**: 构建生产环境的完整发布包
**特性**:
- 跨平台编译支持
- 前端生产构建
- 版本信息注入
- 自动打包和压缩
- 生成部署文件

```bash
# 基础构建
./scripts/build/build.sh

# 构建所有平台版本
BUILD_ALL_PLATFORMS=true ./scripts/build/build.sh

# 指定版本号
BUILD_VERSION=v1.2.0 ./scripts/build/build.sh
```

**输出产物**:
- `bin/` - 可执行文件
- `build/orchestrator-{version}/` - 发布目录
- `build/orchestrator-{version}.tar.gz` - 发布包

## 🚢 部署脚本 (deploy/)

### docker-build.sh - Docker镜像构建
**用途**: 构建和推送Docker镜像
**特性**:
- 多阶段构建优化
- 安全扫描
- 自动标签管理
- 支持私有仓库推送
- 生成Kubernetes部署文件

```bash
# 构建Docker镜像
./scripts/deploy/docker-build.sh

# 构建并推送到仓库
PUSH_IMAGE=true DOCKER_REGISTRY=registry.example.com ./scripts/deploy/docker-build.sh

# 指定镜像名称和版本
IMAGE_NAME=my-orchestrator BUILD_VERSION=v1.0.0 ./scripts/deploy/docker-build.sh
```

**生成文件**:
- `docker-compose.example.yml` - Docker Compose示例
- `k8s/deployment.yaml` - Kubernetes部署文件

## 🧪 测试脚本 (test/)

### run-tests.sh - 自动化测试
**用途**: 运行完整的测试套件
**特性**:
- 单元测试 (Go + Vue)
- 集成测试
- E2E测试
- 性能测试
- 覆盖率报告
- HTML测试报告

```bash
# 运行所有测试
./scripts/test/run-tests.sh

# 运行特定类型测试
./scripts/test/run-tests.sh unit        # 单元测试
./scripts/test/run-tests.sh integration # 集成测试
./scripts/test/run-tests.sh e2e         # E2E测试
./scripts/test/run-tests.sh performance # 性能测试

# CI模式运行
CI=true ./scripts/test/run-tests.sh
```

**测试报告**:
- `test-reports/test-summary.html` - 测试总结
- `test-reports/backend-coverage.html` - 后端覆盖率
- `test-reports/frontend-coverage/` - 前端覆盖率

## 🛠️ 工具脚本 (utils/)

### cleanup.sh - 项目清理
**用途**: 清理项目中的临时文件和构建产物
**特性**:
- 智能文件分类清理
- 大小统计和节省空间计算
- 安全确认机制
- 分类清理支持

```bash
# 清理所有临时文件
./scripts/utils/cleanup.sh

# 分类清理
./scripts/utils/cleanup.sh build   # 构建产物
./scripts/utils/cleanup.sh cache   # 缓存文件
./scripts/utils/cleanup.sh logs    # 日志文件
./scripts/utils/cleanup.sh deps    # 依赖文件
./scripts/utils/cleanup.sh temp    # 临时文件
./scripts/utils/cleanup.sh docker  # Docker相关

# 强制清理 (CI环境)
CI=true ./scripts/utils/cleanup.sh
```

### health-check.sh - 系统健康检查
**用途**: 检查系统各组件运行状态和健康度
**特性**:
- 服务状态检查
- 数据库连接检查
- 系统资源监控
- API接口测试
- 日志分析
- 健康评分

```bash
# 运行健康检查
./scripts/utils/health-check.sh

# 生成健康报告
GENERATE_REPORT=true ./scripts/utils/health-check.sh
```

**检查项目**:
- ✅ 后端服务 (端口8080)
- ✅ 前端服务 (端口3000)
- ✅ 数据库连接 (MySQL/SQLite)
- ✅ API接口可用性
- ✅ 磁盘空间使用
- ✅ 内存使用率
- ✅ 系统负载
- ✅ 日志错误分析

## 📋 使用场景

### 日常开发流程
```bash
# 1. 启动开发环境
./scripts/dev/start.sh

# 2. 开发完成后运行测试
./scripts/test/run-tests.sh unit

# 3. 停止开发环境
./scripts/dev/stop.sh
```

### 构建发布流程
```bash
# 1. 清理项目
./scripts/utils/cleanup.sh

# 2. 运行完整测试
./scripts/test/run-tests.sh

# 3. 构建发布包
BUILD_VERSION=v1.0.0 ./scripts/build/build.sh

# 4. 构建Docker镜像
BUILD_VERSION=v1.0.0 ./scripts/deploy/docker-build.sh
```

### 运维监控流程
```bash
# 1. 健康检查
./scripts/utils/health-check.sh

# 2. 清理日志文件
./scripts/utils/cleanup.sh logs

# 3. 重启服务 (如果需要)
./scripts/dev/stop.sh && ./scripts/dev/start.sh
```

## 🔧 环境变量配置

### 通用配置
```bash
# 项目版本
BUILD_VERSION=v1.0.0

# CI模式
CI=true

# 生成报告
GENERATE_REPORTS=true
```

### 构建配置
```bash
# 构建所有平台
BUILD_ALL_PLATFORMS=true

# Go构建参数
CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
```

### Docker配置
```bash
# Docker仓库
DOCKER_REGISTRY=registry.example.com
DOCKER_USERNAME=myuser
DOCKER_PASSWORD=mypass

# 推送镜像
PUSH_IMAGE=true

# 镜像名称
IMAGE_NAME=orchestrator
```

### 测试配置
```bash
# 覆盖率阈值
COVERAGE_THRESHOLD=80

# 数据库配置
DB_HOST=localhost
DB_PORT=3307
DB_USERNAME=test
DB_PASSWORD=test
DB_DATABASE=orchestrator_test
```

## 🚀 最佳实践

### 开发环境
1. 使用 `scripts/dev/start.sh` 启动开发环境
2. 定期运行 `scripts/utils/health-check.sh` 检查系统状态
3. 提交代码前运行 `scripts/test/run-tests.sh unit`

### 持续集成
1. 在CI流水线中使用 `CI=true` 环境变量
2. 按顺序运行: 清理 → 测试 → 构建 → 部署
3. 保存测试报告和构建产物

### 生产部署
1. 使用构建脚本生成标准化发布包
2. 通过Docker镜像进行容器化部署
3. 定期运行健康检查脚本监控系统状态

### 运维管理
1. 定期清理临时文件和日志
2. 监控系统资源使用情况
3. 建立自动化监控和告警机制

## ⚠️ 注意事项

1. **权限要求**: 所有脚本需要执行权限 (`chmod +x`)
2. **依赖检查**: 脚本会自动检查必要的依赖工具
3. **安全清理**: 清理依赖文件时会有确认提示
4. **端口占用**: 启动前会自动检查和清理端口占用
5. **环境隔离**: 测试和开发环境使用不同的端口和数据库

## 🔗 相关文档

- [开发指南](../docs/development/README.md)
- [部署指南](../docs/deployment/README.md)
- [测试文档](../docs/testing/README.md)
- [API文档](../docs/api/README.md)

---

> 如有问题或建议，请查看项目文档或提交Issue。