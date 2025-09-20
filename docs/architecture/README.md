# 系统架构文档

## 整体架构

工作流编排系统采用前后端分离的微服务架构，提供可扩展、高可用的工作流管理解决方案。

### 架构图

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   前端界面      │    │   后端API       │    │   数据存储      │
│  Vue.js 3.0     │◄──►│   Go + Gin      │◄──►│ MySQL/SQLite    │
│  Ant Design     │    │   RESTful API   │    │   GORM ORM      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   执行引擎      │
                       │ DAG Scheduler   │
                       │ Task Executor   │
                       └─────────────────┘
```

### 核心模块

#### 1. 前端层（Presentation Layer）
- **技术栈**: Vue.js 3.0 + TypeScript + Ant Design Vue
- **职责**: 用户界面、交互逻辑、状态管理
- **组件**:
  - 步骤管理界面
  - 模板管理界面
  - 执行监控界面
  - DAG可视化组件

#### 2. API层（API Layer）
- **技术栈**: Go + Gin Framework
- **职责**: HTTP接口、请求路由、参数验证
- **模块**:
  - `/api/v1/steps` - 步骤管理接口
  - `/api/v1/templates` - 模板管理接口
  - `/api/v1/executions` - 执行管理接口

#### 3. 业务逻辑层（Business Logic Layer）
- **包路径**: `internal/service/`
- **职责**: 核心业务逻辑、数据处理、业务规则
- **服务**:
  - `StepService` - 步骤管理服务
  - `TemplateService` - 模板管理服务
  - `ExecutionService` - 执行管理服务
  - `OrchestratorService` - 编排调度服务

#### 4. 数据访问层（Data Access Layer）
- **技术栈**: GORM + MySQL/SQLite
- **职责**: 数据持久化、事务管理、数据查询
- **模块**:
  - `internal/dal/` - 数据访问层
  - `internal/model/` - 数据模型

#### 5. 执行引擎（Execution Engine）
- **包路径**: `internal/engine/`
- **职责**: 工作流执行、任务调度、状态管理
- **组件**:
  - DAG解析器
  - 任务调度器
  - 执行器（Shell/HTTP/Function）

## 数据模型

### 核心实体关系

```
┌─────────────┐    ┌──────────────────┐    ┌─────────────┐
│    Steps    │    │ TemplateSteps    │    │  Templates  │
│   步骤定义   │◄──►│   模板步骤关联   │◄──►│   工作流模板 │
└─────────────┘    └──────────────────┘    └─────────────┘
                            │
                            ▼
                   ┌─────────────────┐
                   │   Executions    │
                   │    执行记录     │
                   └─────────────────┘
                            │
                            ▼
                   ┌─────────────────┐
                   │ StepExecutions  │
                   │  步骤执行记录   │
                   └─────────────────┘
```

### 主要数据表

#### steps (步骤表)
```sql
CREATE TABLE steps (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL UNIQUE,
    description TEXT,
    executor_type ENUM('shell','http','function'),
    executor_config JSON,
    timeout INT DEFAULT 300,
    retry_count INT DEFAULT 0,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

#### workflow_templates (模板表)
```sql
CREATE TABLE workflow_templates (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL UNIQUE,
    description TEXT,
    status ENUM('active','inactive') DEFAULT 'active',
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

#### workflow_template_steps (模板步骤关联表)
```sql
CREATE TABLE workflow_template_steps (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    template_id BIGINT,
    step_id BIGINT,
    dependencies JSON,
    run_mode ENUM('serial','parallel'),
    on_failure ENUM('abort','continue','retry'),
    order_num INT,
    FOREIGN KEY (template_id) REFERENCES workflow_templates(id),
    FOREIGN KEY (step_id) REFERENCES steps(id)
);
```

#### workflow_executions (执行记录表)
```sql
CREATE TABLE workflow_executions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    template_id BIGINT,
    status ENUM('pending','running','success','failed','cancelled'),
    executed_by VARCHAR(64),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    remarks TEXT,
    created_at TIMESTAMP,
    FOREIGN KEY (template_id) REFERENCES workflow_templates(id)
);
```

## 技术选型

### 后端技术栈
- **语言**: Go 1.21+ (高性能、并发友好)
- **Web框架**: Gin (轻量级、高性能)
- **ORM**: GORM (功能完善、社区活跃)
- **数据库**: MySQL 8.0+ / SQLite (主备切换)
- **日志**: Logrus (结构化日志)
- **配置**: Viper (多格式配置)

### 前端技术栈
- **框架**: Vue.js 3.0 (Composition API)
- **语言**: TypeScript (类型安全)
- **UI库**: Ant Design Vue 3.0 (企业级组件)
- **状态管理**: Pinia (新一代状态管理)
- **路由**: Vue Router 4.0 (官方路由)
- **HTTP**: Axios (请求拦截、响应处理)
- **构建**: Vite (快速构建)

### 基础设施
- **容器化**: Docker (开发、测试、部署)
- **数据库**: MySQL 8.0 / SQLite (高可用、易部署)
- **缓存**: Redis (可选，性能优化)
- **监控**: Prometheus + Grafana (可选)

## 设计原则

### 1. 分层架构
- 清晰的层次划分
- 单向依赖
- 接口抽象

### 2. 微服务思想
- 业务模块独立
- 接口标准化
- 可扩展性强

### 3. 数据一致性
- 事务边界清晰
- 数据完整性约束
- 并发控制机制

### 4. 高可用设计
- 优雅降级
- 错误隔离
- 监控告警

### 5. 安全设计
- 输入验证
- 权限控制
- 审计日志

## 扩展性考虑

### 水平扩展
- 无状态API设计
- 数据库分片支持
- 负载均衡友好

### 功能扩展
- 插件式执行器
- 自定义工作流引擎
- 第三方系统集成

### 性能优化
- 数据库索引优化
- 缓存策略
- 异步处理机制

---

> 架构设计会随着业务发展持续优化，请关注版本更新。