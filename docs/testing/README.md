# 测试文档

## 测试策略

工作流编排系统采用分层测试策略，确保系统质量和稳定性。

### 测试金字塔

```
        ┌─────────────────┐
        │   E2E Tests     │  ← 端到端测试
        │   (Cypress)     │
        └─────────────────┘
      ┌───────────────────────┐
      │  Integration Tests    │  ← 集成测试
      │   (API Tests)         │
      └───────────────────────┘
    ┌─────────────────────────────┐
    │      Unit Tests             │  ← 单元测试
    │  (Go + Jest/Vitest)         │
    └─────────────────────────────┘
```

## 单元测试

### 后端单元测试 (Go)

使用 Go 内置的 testing 包和 testify 断言库。

#### 测试结构
```
internal/
├── service/
│   ├── step_service.go
│   └── step_service_test.go
├── api/
│   ├── handler/
│   │   ├── step_handler.go
│   │   └── step_handler_test.go
└── dal/
    ├── step_dal.go
    └── step_dal_test.go
```

#### 示例测试

```go
// internal/service/step_service_test.go
package service

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestStepService_CreateStep(t *testing.T) {
    // Arrange
    mockDAL := new(MockStepDAL)
    service := NewStepService(mockDAL)
    
    stepReq := &StepCreateRequest{
        Name: "test_step",
        ExecutorType: "shell",
        ExecutorConfig: map[string]interface{}{
            "command": "echo 'test'",
        },
    }
    
    mockDAL.On("Create", mock.AnythingOfType("*model.Step")).Return(nil)
    
    // Act
    step, err := service.CreateStep(stepReq)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "test_step", step.Name)
    assert.Equal(t, "shell", step.ExecutorType)
    mockDAL.AssertExpectations(t)
}

func TestStepService_ValidateStepName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid name", "test_step-123", false},
        {"invalid chinese", "测试步骤", true},
        {"invalid space", "test step", true},
        {"invalid special", "test@step", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateStepName(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### 运行测试
```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/service

# 生成覆盖率报告
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 前端单元测试 (Vue + Vitest)

使用 Vitest 作为测试运行器，Vue Test Utils 进行组件测试。

#### 测试结构
```
web/frontend/
├── src/
│   ├── components/
│   │   ├── StepFormDialog.vue
│   │   └── __tests__/
│   │       └── StepFormDialog.test.ts
│   └── utils/
│       ├── validator.ts
│       └── __tests__/
│           └── validator.test.ts
└── vitest.config.ts
```

#### 示例测试

```typescript
// src/components/__tests__/StepFormDialog.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import StepFormDialog from '../StepFormDialog.vue'

describe('StepFormDialog', () => {
  let wrapper: any
  
  beforeEach(() => {
    const pinia = createPinia()
    wrapper = mount(StepFormDialog, {
      global: {
        plugins: [pinia]
      },
      props: {
        visible: true,
        mode: 'create'
      }
    })
  })
  
  it('should render form correctly', () => {
    expect(wrapper.find('[data-testid="step-name"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="executor-type"]').exists()).toBe(true)
  })
  
  it('should validate step name', async () => {
    const nameInput = wrapper.find('[data-testid="step-name"]')
    
    // 测试无效名称
    await nameInput.setValue('测试中文')
    await nameInput.trigger('blur')
    
    expect(wrapper.find('.ant-form-item-explain-error').text())
      .toContain('步骤名称只能包含字母、数字、下划线、横线')
  })
  
  it('should emit create event on form submit', async () => {
    // 填写表单
    await wrapper.find('[data-testid="step-name"]').setValue('test_step')
    await wrapper.find('[data-testid="executor-type"]').setValue('shell')
    
    // 提交表单
    await wrapper.find('form').trigger('submit')
    
    expect(wrapper.emitted('create')).toBeTruthy()
    expect(wrapper.emitted('create')[0][0]).toMatchObject({
      name: 'test_step',
      executor_type: 'shell'
    })
  })
})
```

#### 运行测试
```bash
cd web/frontend

# 运行所有测试
npm run test

# 运行测试并监听变化
npm run test:watch

# 生成覆盖率报告
npm run test:coverage
```

## 集成测试

### API 集成测试

测试各个 API 端点的集成功能。

```go
// test/integration/api_test.go
package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

func TestStepAPI_Integration(t *testing.T) {
    // 初始化测试数据库
    db := setupTestDB()
    defer cleanupTestDB(db)
    
    // 初始化 Gin 引擎
    gin.SetMode(gin.TestMode)
    router := setupRouter(db)
    
    t.Run("Create Step", func(t *testing.T) {
        stepData := map[string]interface{}{
            "name": "test_step",
            "description": "测试步骤",
            "executor_type": "shell",
            "executor_config": map[string]interface{}{
                "command": "echo 'test'",
                "timeout": 30,
            },
        }
        
        jsonData, _ := json.Marshal(stepData)
        req := httptest.NewRequest("POST", "/api/v1/steps", bytes.NewBuffer(jsonData))
        req.Header.Set("Content-Type", "application/json")
        
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        
        assert.Equal(t, http.StatusCreated, w.Code)
        
        var response map[string]interface{}
        err := json.Unmarshal(w.Body.Bytes(), &response)
        assert.NoError(t, err)
        assert.Equal(t, "test_step", response["name"])
    })
    
    t.Run("Get Steps List", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/steps", nil)
        w := httptest.NewRecorder()
        
        router.ServeHTTP(w, req)
        
        assert.Equal(t, http.StatusOK, w.Code)
        
        var response map[string]interface{}
        err := json.Unmarshal(w.Body.Bytes(), &response)
        assert.NoError(t, err)
        assert.Contains(t, response, "data")
        assert.Contains(t, response, "pagination")
    })
}
```

### 数据库集成测试

```go
// test/integration/database_test.go
package integration

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestDatabase_Migration(t *testing.T) {
    db := setupTestDB()
    defer cleanupTestDB(db)
    
    // 测试表创建
    assert.True(t, db.Migrator().HasTable(&model.Step{}))
    assert.True(t, db.Migrator().HasTable(&model.WorkflowTemplate{}))
    assert.True(t, db.Migrator().HasTable(&model.WorkflowExecution{}))
}

func TestStepDAL_CRUD(t *testing.T) {
    db := setupTestDB()
    defer cleanupTestDB(db)
    
    dal := NewStepDAL(db)
    
    // Create
    step := &model.Step{
        Name: "test_step",
        ExecutorType: "shell",
        ExecutorConfig: `{"command": "echo 'test'"}`,
    }
    
    err := dal.Create(step)
    assert.NoError(t, err)
    assert.NotZero(t, step.ID)
    
    // Read
    foundStep, err := dal.GetByID(step.ID)
    assert.NoError(t, err)
    assert.Equal(t, "test_step", foundStep.Name)
    
    // Update
    foundStep.Description = "更新后的描述"
    err = dal.Update(foundStep)
    assert.NoError(t, err)
    
    // Delete
    err = dal.Delete(step.ID)
    assert.NoError(t, err)
    
    // Verify deletion
    _, err = dal.GetByID(step.ID)
    assert.Error(t, err)
}
```

## 端到端测试

### Cypress E2E 测试

使用 Cypress 进行完整的用户流程测试。

```typescript
// cypress/e2e/step-management.cy.ts
describe('步骤管理', () => {
  beforeEach(() => {
    // 访问应用
    cy.visit('/steps')
    
    // 等待页面加载
    cy.get('[data-testid="steps-list"]').should('be.visible')
  })
  
  it('应该能够创建新步骤', () => {
    // 点击创建按钮
    cy.get('[data-testid="create-step-btn"]').click()
    
    // 填写表单
    cy.get('[data-testid="step-name"]').type('e2e_test_step')
    cy.get('[data-testid="step-description"]').type('E2E测试步骤')
    cy.get('[data-testid="executor-type"]').select('shell')
    cy.get('[data-testid="command"]').type('echo "E2E test"')
    
    // 提交表单
    cy.get('[data-testid="submit-btn"]').click()
    
    // 验证成功提示
    cy.get('.ant-message-success').should('contain', '步骤创建成功')
    
    // 验证列表中出现新步骤
    cy.get('[data-testid="steps-list"]').should('contain', 'e2e_test_step')
  })
  
  it('应该能够编辑步骤', () => {
    // 点击编辑按钮
    cy.get('[data-testid="edit-btn"]').first().click()
    
    // 修改描述
    cy.get('[data-testid="step-description"]').clear().type('修改后的描述')
    
    // 保存
    cy.get('[data-testid="submit-btn"]').click()
    
    // 验证更新成功
    cy.get('.ant-message-success').should('contain', '步骤更新成功')
  })
  
  it('应该能够删除步骤', () => {
    // 点击删除按钮
    cy.get('[data-testid="delete-btn"]').first().click()
    
    // 确认删除
    cy.get('.ant-modal-confirm-btns .ant-btn-primary').click()
    
    // 验证删除成功
    cy.get('.ant-message-success').should('contain', '步骤删除成功')
  })
})

// cypress/e2e/workflow-execution.cy.ts
describe('工作流执行', () => {
  it('应该能够执行完整的工作流', () => {
    // 创建步骤
    cy.visit('/steps')
    cy.get('[data-testid="create-step-btn"]').click()
    cy.get('[data-testid="step-name"]').type('step_a')
    cy.get('[data-testid="executor-type"]').select('shell')
    cy.get('[data-testid="command"]').type('echo "Step A"')
    cy.get('[data-testid="submit-btn"]').click()
    
    // 创建模板
    cy.visit('/templates')
    cy.get('[data-testid="create-template-btn"]').click()
    cy.get('[data-testid="template-name"]').type('e2e_workflow')
    cy.get('[data-testid="add-step-btn"]').click()
    cy.get('[data-testid="step-select"]').select('step_a')
    cy.get('[data-testid="submit-btn"]').click()
    
    // 执行工作流
    cy.get('[data-testid="execute-btn"]').first().click()
    cy.get('[data-testid="execute-confirm-btn"]').click()
    
    // 验证执行状态
    cy.visit('/executions')
    cy.get('[data-testid="execution-status"]').should('contain', 'running')
    
    // 等待执行完成
    cy.get('[data-testid="execution-status"]', { timeout: 30000 })
      .should('contain', 'success')
  })
})
```

## 性能测试

### 负载测试

使用 Apache Bench 或 wrk 进行 API 负载测试。

```bash
# 使用 ab 测试步骤列表 API
ab -n 1000 -c 10 http://localhost:8080/api/v1/steps

# 使用 wrk 测试
wrk -t10 -c100 -d30s http://localhost:8080/api/v1/steps
```

### 数据库性能测试

```sql
-- 插入测试数据
INSERT INTO steps (name, description, executor_type, executor_config) 
SELECT 
    CONCAT('test_step_', n),
    CONCAT('测试步骤 ', n),
    'shell',
    '{"command": "echo test"}'
FROM (
    SELECT ROW_NUMBER() OVER () as n
    FROM information_schema.columns c1, information_schema.columns c2
    LIMIT 10000
) t;

-- 测试查询性能
EXPLAIN ANALYZE SELECT * FROM steps WHERE name LIKE 'test%' LIMIT 20;
EXPLAIN ANALYZE SELECT * FROM workflow_executions WHERE status = 'running';
```

## 测试数据管理

### 测试夹具 (Fixtures)

```go
// test/fixtures/step_fixtures.go
package fixtures

func CreateTestStep(db *gorm.DB, name string) *model.Step {
    step := &model.Step{
        Name: name,
        Description: "测试步骤",
        ExecutorType: "shell",
        ExecutorConfig: `{"command": "echo 'test'"}`,
        Timeout: 300,
        RetryCount: 0,
    }
    db.Create(step)
    return step
}

func CreateTestTemplate(db *gorm.DB, name string) *model.WorkflowTemplate {
    template := &model.WorkflowTemplate{
        Name: name,
        Description: "测试模板",
        Status: "active",
    }
    db.Create(template)
    return template
}
```

### 数据清理

```go
// test/helpers/cleanup.go
package helpers

func CleanupTestData(db *gorm.DB) {
    // 清理测试数据，注意外键约束
    db.Exec("DELETE FROM workflow_step_executions WHERE 1=1")
    db.Exec("DELETE FROM workflow_executions WHERE 1=1")
    db.Exec("DELETE FROM workflow_template_steps WHERE 1=1")
    db.Exec("DELETE FROM workflow_templates WHERE 1=1")
    db.Exec("DELETE FROM steps WHERE name LIKE 'test_%'")
}
```

## 测试运行

### 持续集成配置

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  backend-test:
    runs-on: ubuntu-latest
    
    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_ROOT_PASSWORD: root123456
          MYSQL_DATABASE: orchestrator_test
        options: >-
          --health-cmd="mysqladmin ping"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=3
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Run tests
      env:
        DB_HOST: localhost
        DB_PASSWORD: root123456
        DB_DATABASE: orchestrator_test
      run: |
        go test -v -cover ./...
        
  frontend-test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Node.js
      uses: actions/setup-node@v3
      with:
        node-version: 18
    
    - name: Install dependencies
      run: |
        cd web/frontend
        npm ci
    
    - name: Run tests
      run: |
        cd web/frontend
        npm run test:coverage
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
```

### 本地测试脚本

```bash
#!/bin/bash
# scripts/run-tests.sh

set -e

echo "Running backend tests..."
go test -v -cover ./...

echo "Running frontend tests..."
cd web/frontend
npm run test

echo "Running E2E tests..."
npm run cypress:run

echo "All tests passed!"
```

## 测试报告

### 覆盖率报告

```bash
# 生成 Go 覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 生成前端覆盖率报告
cd web/frontend
npm run test:coverage
```

### 测试指标

- **单元测试覆盖率**: > 80%
- **集成测试覆盖率**: > 70%
- **E2E 测试覆盖率**: > 60%
- **API 响应时间**: < 100ms (95th percentile)
- **数据库查询时间**: < 50ms (95th percentile)

---

> 测试是保证代码质量的重要手段，建议在每次提交前运行完整的测试套件。