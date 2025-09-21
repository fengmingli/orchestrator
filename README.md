# 工作流编排系统 (Orchestrator)

基于 Go + Vue.js 的现代化工作流编排系统，支持可视化流程管理和自动化任务执行。

## 🚀 功能特性

- **步骤管理**: 支持 Shell、HTTP、Function 多种执行器类型
- **模板管理**: 可视化工作流模板设计，支持DAG图展示
- **执行管理**: 实时监控执行状态，支持日志查看和任务控制
- **现代化界面**: 基于 Ant Design Vue 3.0 的美观管理界面
- **数据库支持**: 优先使用 MySQL，自动降级到 SQLite
- **RESTful API**: 完整的 API 接口，支持第三方集成

## 🛠️ 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin + GORM
- **数据库**: MySQL (主) / SQLite (备)
- **日志**: Logrus

### 前端
- **框架**: Vue.js 3.0 + TypeScript
- **UI组件**: Ant Design Vue 3.0
- **状态管理**: Pinia
- **路由**: Vue Router 4.0
- **HTTP客户端**: Axios
- **构建工具**: Vite

## 📦 安装与启动

### 环境要求

- Go 1.21+
- Node.js 18+
- MySQL 8.0+ (可选，会自动降级到 SQLite)

### 快速启动

```bash
# 克隆项目
git clone <repository-url>
cd orchestrator

# 一键启动 (包含编译、构建、启动所有服务)
./scripts/start.sh

# 直接使用开发环境脚本
./scripts/dev/start.sh

# 访问管理界面
# 前端: http://localhost:3000
# 后端API: http://localhost:8080
```

### 手动启动

```bash
# 1. 编译后端
cd cmd/web
go build -o ../../bin/web main.go
cd ../..

# 2. 构建前端
cd web/frontend
npm install
npm run build
cd ../..

# 3. 启动后端
./bin/web

# 4. 启动前端开发服务器 (新终端)
cd web/frontend
npm run dev
```

### 停止服务

```bash
# 主要方式
./scripts/stop.sh

# 直接使用开发环境脚本
./scripts/dev/stop.sh
```

## 🔧 配置说明

### 数据库配置

系统使用环境变量或默认值进行配置：

```bash
# MySQL 配置 (优先使用)
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USERNAME=root
DB_PASSWORD=root123456
DB_DATABASE=orchestrator
DB_CHARSET=utf8mb4

# 服务器配置
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
```

如果 MySQL 连接失败，系统会自动使用 SQLite 数据库。

## 📚 API 文档

### 健康检查
```bash
GET /health
```

### 步骤管理
```bash
GET    /api/v1/steps              # 获取步骤列表
POST   /api/v1/steps              # 创建步骤
GET    /api/v1/steps/{id}         # 获取步骤详情
PUT    /api/v1/steps/{id}         # 更新步骤
DELETE /api/v1/steps/{id}         # 删除步骤
POST   /api/v1/steps/{id}/validate # 验证步骤
```

### 模板管理
```bash
GET    /api/v1/templates              # 获取模板列表
POST   /api/v1/templates              # 创建模板
GET    /api/v1/templates/{id}         # 获取模板详情
PUT    /api/v1/templates/{id}         # 更新模板
DELETE /api/v1/templates/{id}         # 删除模板
GET    /api/v1/templates/{id}/dag     # 获取模板DAG结构
POST   /api/v1/templates/{id}/steps   # 向模板添加步骤
DELETE /api/v1/templates/{id}/steps/{stepId} # 从模板移除步骤
```

### 执行管理
```bash
GET    /api/v1/executions              # 获取执行列表
POST   /api/v1/executions              # 创建执行
GET    /api/v1/executions/{id}         # 获取执行详情
POST   /api/v1/executions/{id}/start   # 启动执行
POST   /api/v1/executions/{id}/cancel  # 取消执行
GET    /api/v1/executions/{id}/logs    # 获取执行日志
GET    /api/v1/executions/{id}/status  # 获取执行状态
```

## 🗂️ 项目结构

```
orchestrator/
├── cmd/
│   ├── executor/           # 执行器命令行工具
│   └── web/               # Web服务主程序
├── internal/
│   ├── api/               # API路由和处理器
│   ├── config/            # 配置管理
│   ├── dal/               # 数据访问层
│   ├── engine/            # 执行引擎
│   ├── model/             # 数据模型
│   ├── service/           # 业务逻辑层
│   └── store/             # 存储层
├── web/
│   └── frontend/          # Vue.js前端项目
├── pkg/                   # 公共包
├── bin/                   # 编译输出目录
├── scripts/               # 脚本目录
│   ├── start.sh           # 主启动脚本
│   ├── stop.sh            # 主停止脚本
│   ├── dev/               # 开发环境脚本
│   │   ├── start.sh       # 开发环境启动脚本
│   │   └── stop.sh        # 开发环境停止脚本
│   ├── build/             # 构建脚本
│   ├── deploy/            # 部署脚本
│   ├── test/              # 测试脚本
│   └── utils/             # 工具脚本
└── CLAUDE.md              # 项目配置文档
```

## 🔍 开发调试

### 查看日志
```bash
# 后端日志
tail -f web.log

# 前端开发服务器日志
# 在前端启动终端查看
```

### 数据库管理
```bash
# 查看 SQLite 数据库
sqlite3 orchestrator.db ".tables"
sqlite3 orchestrator.db ".schema"
```

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 📚 文档导航

### 📖 完整文档
- [📁 文档中心](./docs/) - 完整的项目文档和指南

### 🏗️ 架构设计
- [系统架构概述](./docs/architecture/) - 系统整体架构设计
- [数据模型设计](./docs/architecture/) - 数据库表结构和关系

### 🚀 功能特性
- [功能实现总结](./docs/features/功能实现总结.md) - 系统核心功能概述
- [DAG图可视化优化](./docs/features/DAG图可视化优化总结.md) - 可视化工作流图
- [模板多依赖关系](./docs/features/模板多依赖关系实现总结.md) - 复杂依赖关系支持

### 🛠️ 开发指南
- [Web应用开发](./docs/development/) - 前端开发指南和API说明
- [开发环境搭建](./docs/development/) - 开发环境配置

### 📡 API文档
- [RESTful API](./docs/api/) - 完整的API接口文档

### 🚀 部署运维
- [部署指南](./docs/deployment/) - 生产环境部署配置
- [运维监控](./docs/deployment/) - 日志和性能监控

### 🧪 测试文档
- [测试策略](./docs/testing/) - 测试方案和覆盖范围

## 📄 许可证

本项目基于 MIT 许可证开源。详见 [LICENSE](LICENSE) 文件。