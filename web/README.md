# 工作流编排系统 Web 应用

## 项目概述

基于文档中的设计要求，完成了前后端交互的工作流编排系统，主要技术栈：

- **后端**: Gin + Gorm + MySQL
- **前端**: Vue 3 + Element Plus + Vite

## 功能特性

### 后端 API 功能

1. **步骤管理** (`/api/steps`)
   - 创建、查询、更新、删除步骤
   - 支持 HTTP、Shell、Function 三种执行器类型
   - 完整的参数验证和错误处理

2. **模板管理** (`/api/templates`)
   - 创建、查询、更新、删除工作流模板
   - 支持 DAG 依赖关系配置
   - 支持并行/串行执行模式

3. **执行管理** (`/api/executions`)
   - 创建、启动、取消工作流执行
   - 实时执行状态跟踪
   - 执行日志和进度监控

4. **workflow 引擎集成**
   - 完整集成现有的 DAG 工作流引擎
   - 支持状态回调和实时更新
   - 任务重试和错误处理机制

### 前端界面功能

1. **步骤管理界面**
   - 步骤列表查看和搜索
   - 步骤创建/编辑对话框
   - 支持不同执行器类型的参数配置
   - 表单验证和错误提示

2. **模板管理界面**
   - 模板列表查看和筛选
   - 模板执行功能
   - DAG 可视化支持（预留接口）

3. **执行记录界面**
   - 执行列表和状态监控
   - 执行进度可视化
   - 执行启动/取消操作
   - 日志查看功能（预留接口）

## 项目结构

```
web/
├── frontend/                 # Vue 前端项目
│   ├── src/
│   │   ├── api/             # API 接口封装
│   │   ├── components/      # 通用组件
│   │   ├── views/           # 页面组件
│   │   ├── router/          # 路由配置
│   │   └── main.js          # 应用入口
│   ├── package.json
│   ├── vite.config.js
│   └── index.html
└── README.md
```

## 快速开始

### 后端启动

1. 配置环境变量 (`.env` 文件):
```bash
SERVER_PORT=8080
SERVER_HOST=localhost

DB_HOST=localhost
DB_PORT=3306
DB_USERNAME=root
DB_PASSWORD=
DB_DATABASE=orchestrator
DB_CHARSET=utf8mb4
```

2. 启动后端服务:
```bash
# 从项目根目录执行
go run cmd/web/main.go
```

### 前端启动

1. 安装依赖:
```bash
cd web/frontend
npm install
```

2. 启动开发服务器:
```bash
npm run dev
```

前端将在 http://localhost:3000 启动，并自动代理 API 请求到后端 http://localhost:8080

## API 接口

### 步骤管理 API
- `GET /api/steps` - 获取步骤列表
- `POST /api/steps` - 创建步骤
- `GET /api/steps/{id}` - 获取步骤详情
- `PUT /api/steps/{id}` - 更新步骤
- `DELETE /api/steps/{id}` - 删除步骤

### 模板管理 API
- `GET /api/templates` - 获取模板列表
- `POST /api/templates` - 创建模板
- `GET /api/templates/{id}` - 获取模板详情
- `PUT /api/templates/{id}` - 更新模板
- `DELETE /api/templates/{id}` - 删除模板
- `GET /api/templates/{id}/dag` - 获取模板 DAG 图

### 执行管理 API
- `GET /api/executions` - 获取执行列表
- `POST /api/executions` - 创建执行
- `GET /api/executions/{id}` - 获取执行详情
- `POST /api/executions/{id}/start` - 启动执行
- `POST /api/executions/{id}/cancel` - 取消执行
- `GET /api/executions/{id}/status` - 获取执行状态
- `GET /api/executions/{id}/logs` - 获取执行日志

## 技术亮点

1. **完整的 CRUD 操作**: 后端提供完整的增删改查 API
2. **实时状态更新**: 执行过程中的状态实时回调更新
3. **参数验证**: 前后端双重参数验证
4. **错误处理**: 完善的错误处理和用户提示
5. **响应式设计**: 基于 Element Plus 的现代化 UI
6. **代码规范**: 清晰的代码结构和注释

## 下一步计划

1. 完善模板的 DAG 可视化编辑器
2. 增强执行日志的实时查看功能
3. 添加用户权限管理
4. 支持更多执行器类型
5. 性能优化和监控

## 数据库设计

系统使用以下主要数据表：

- `steps` - 步骤定义表
- `workflow_templates` - 工作流模板表
- `workflow_template_steps` - 模板步骤关联表
- `workflow_executions` - 工作流执行表
- `workflow_step_executions` - 步骤执行记录表

数据库会在首次启动时自动创建表结构。