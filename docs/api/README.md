# API 文档

## RESTful API 接口

工作流编排系统提供完整的 RESTful API 接口，支持所有核心功能的程序化访问。

### 基础信息

- **基础URL**: `http://localhost:8080/api/v1`
- **数据格式**: JSON
- **字符编码**: UTF-8

### 健康检查

```http
GET /health
```

响应：
```json
{
  "status": "ok",
  "timestamp": "2025-09-20T10:30:00Z"
}
```

### 步骤管理 API

#### 获取步骤列表
```http
GET /api/v1/steps?page=1&size=10&search=keyword
```

#### 创建步骤
```http
POST /api/v1/steps
Content-Type: application/json

{
  "name": "example_step",
  "description": "示例步骤",
  "executor_type": "shell",
  "executor_config": {
    "command": "echo 'Hello World'",
    "timeout": 30
  }
}
```

#### 获取步骤详情
```http
GET /api/v1/steps/{id}
```

#### 更新步骤
```http
PUT /api/v1/steps/{id}
Content-Type: application/json
```

#### 删除步骤
```http
DELETE /api/v1/steps/{id}
```

#### 验证步骤
```http
POST /api/v1/steps/{id}/validate
```

### 模板管理 API

#### 获取模板列表
```http
GET /api/v1/templates?page=1&size=10&status=active
```

#### 创建模板
```http
POST /api/v1/templates
Content-Type: application/json

{
  "name": "example_template",
  "description": "示例模板",
  "status": "active",
  "steps": [
    {
      "step_id": "step_1",
      "dependencies": [],
      "run_mode": "serial",
      "on_failure": "abort",
      "order": 1
    }
  ]
}
```

#### 获取模板详情
```http
GET /api/v1/templates/{id}
```

#### 获取模板DAG结构
```http
GET /api/v1/templates/{id}/dag
```

#### 向模板添加步骤
```http
POST /api/v1/templates/{id}/steps
Content-Type: application/json

{
  "step_id": "step_2",
  "dependencies": ["step_1"],
  "run_mode": "parallel",
  "on_failure": "continue",
  "order": 2
}
```

#### 从模板移除步骤
```http
DELETE /api/v1/templates/{id}/steps/{stepId}
```

### 执行管理 API

#### 获取执行列表
```http
GET /api/v1/executions?page=1&size=10&template_id=1&status=running
```

#### 创建执行
```http
POST /api/v1/executions
Content-Type: application/json

{
  "template_id": 1,
  "executed_by": "admin",
  "remarks": "测试执行"
}
```

#### 启动执行
```http
POST /api/v1/executions/{id}/start
```

#### 取消执行
```http
POST /api/v1/executions/{id}/cancel
```

#### 获取执行日志
```http
GET /api/v1/executions/{id}/logs
```

#### 获取执行状态
```http
GET /api/v1/executions/{id}/status
```

### 错误响应格式

```json
{
  "error": "错误消息",
  "code": "ERROR_CODE",
  "details": "详细错误信息"
}
```

### 常见错误码

- `400` - 请求参数错误
- `401` - 未认证
- `403` - 权限不足
- `404` - 资源不存在
- `500` - 服务器内部错误

### 分页响应格式

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "size": 10,
    "total": 100,
    "pages": 10
  }
}
```

---

> 详细的接口参数说明和示例，请参考各个具体功能模块的API文档。