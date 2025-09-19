package task

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBaseTask 测试基础任务功能
func TestBaseTask(t *testing.T) {
	task := &BaseTask{
		ID:      "test-task-1",
		Name:    "Test Task",
		Type:    TaskTypeFunc,
		Timeout: 10 * time.Second,
	}

	assert.Equal(t, "test-task-1", task.GetID())
	assert.Equal(t, "Test Task", task.GetName())
	assert.Equal(t, TaskTypeFunc, task.GetType())
	assert.Equal(t, 10*time.Second, task.GetTimeout())

	// 测试验证
	assert.NoError(t, task.Validate())

	// 测试空ID验证失败
	task.ID = ""
	assert.Error(t, task.Validate())

	// 测试空名称验证失败
	task.ID = "test"
	task.Name = ""
	assert.Error(t, task.Validate())
}

// TestHTTPTask 测试HTTP任务
func TestHTTPTask(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		assert.Equal(t, "POST", r.Method)

		// 验证请求头
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-value", r.Header.Get("X-Test-Header"))

		// 验证请求体
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		assert.Equal(t, `{"key":"value"}`, string(body))

		// 返回响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","message":"ok"}`))
	}))
	defer server.Close()

	// 创建HTTP任务
	task := NewHTTPTask("http-1", "HTTP Test", "POST", server.URL).
		WithHeaders(map[string]string{
			"Content-Type":  "application/json",
			"X-Test-Header": "test-value",
		}).
		WithBody(`{"key":"value"}`)

	// 验证任务
	assert.NoError(t, task.Validate())

	// 执行任务
	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "http-1", result.TaskID)
	assert.NotZero(t, result.Duration)
	assert.Equal(t, 200, result.Metadata["status_code"])

	// 验证输出
	output, ok := result.Output.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 200, output["status_code"])
	assert.Contains(t, output["body"], "success")
}

// TestHTTPTaskError 测试HTTP任务错误处理
func TestHTTPTaskError(t *testing.T) {
	// 创建返回错误的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	task := NewHTTPTask("http-error", "HTTP Error Test", "GET", server.URL)

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
	assert.Equal(t, "http-error", result.TaskID)
	assert.Equal(t, 500, result.Metadata["status_code"])
}

// TestHTTPTaskTimeout 测试HTTP任务超时
func TestHTTPTaskTimeout(t *testing.T) {
	// 创建慢响应的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	task := NewHTTPTask("http-timeout", "HTTP Timeout Test", "GET", server.URL)
	task.Timeout = 100 * time.Millisecond // 设置很短的超时

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "timeout")
	assert.Equal(t, "http-timeout", result.TaskID)
}

// TestHTTPTaskValidation 测试HTTP任务验证
func TestHTTPTaskValidation(t *testing.T) {
	// 测试空URL
	task := NewHTTPTask("http-invalid", "Invalid HTTP", "GET", "")
	assert.Error(t, task.Validate())

	// 测试无效方法
	task.URL = "http://example.com"
	task.Method = "INVALID"
	assert.Error(t, task.Validate())

	// 测试有效配置
	task.Method = "GET"
	assert.NoError(t, task.Validate())
}

// TestShellTask 测试Shell任务
func TestShellTask(t *testing.T) {
	// 测试简单命令
	task := NewShellTask("shell-1", "Shell Test", "echo 'Hello World'")

	assert.NoError(t, task.Validate())

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "shell-1", result.TaskID)
	assert.Contains(t, result.Output, "Hello World")
	assert.Equal(t, 0, result.Metadata["exit_code"])
}

// TestShellTaskWithEnv 测试带环境变量的Shell任务
func TestShellTaskWithEnv(t *testing.T) {
	task := NewShellTask("shell-env", "Shell Env Test", "echo $TEST_VAR").
		AddEnvironment("TEST_VAR", "test-value")

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.NoError(t, err)
	assert.Contains(t, result.Output, "test-value")
	assert.Equal(t, map[string]string{"TEST_VAR": "test-value"}, result.Metadata["environment"])
}

// TestShellTaskError 测试Shell任务错误处理
func TestShellTaskError(t *testing.T) {
	task := NewShellTask("shell-error", "Shell Error Test", "exit 1")

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.Error(t, err)
	assert.Equal(t, "shell-error", result.TaskID)
	assert.Equal(t, 1, result.Metadata["exit_code"])
}

// TestShellTaskValidation 测试Shell任务验证
func TestShellTaskValidation(t *testing.T) {
	// 测试空脚本
	task := NewShellTask("shell-invalid", "Invalid Shell", "")
	assert.Error(t, task.Validate())

	// 测试空白脚本
	task.Script = "   "
	assert.Error(t, task.Validate())

	// 测试有效脚本
	task.Script = "echo test"
	assert.NoError(t, task.Validate())
}

// TestFuncTask 测试函数任务
func TestFuncTask(t *testing.T) {
	// 测试成功的函数
	task := NewFuncTask("func-1", "Func Test", func(ctx context.Context) (interface{}, error) {
		return "function result", nil
	})

	assert.NoError(t, task.Validate())

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "func-1", result.TaskID)
	assert.Equal(t, "function result", result.Output)
}

// TestFuncTaskError 测试函数任务错误处理
func TestFuncTaskError(t *testing.T) {
	task := NewFuncTask("func-error", "Func Error Test", func(ctx context.Context) (interface{}, error) {
		return nil, assert.AnError
	})

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.Error(t, err)
	assert.Equal(t, "func-error", result.TaskID)
	assert.Equal(t, assert.AnError.Error(), result.Error)
}

// TestSimpleFuncTask 测试简单函数任务
func TestSimpleFuncTask(t *testing.T) {
	task := NewSimpleFuncTask("simple-func", "Simple Func Test", func() (interface{}, error) {
		return map[string]interface{}{"result": "success"}, nil
	})

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "simple-func", result.TaskID)

	output, ok := result.Output.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "success", output["result"])
}

// TestVoidFuncTask 测试无返回值函数任务
func TestVoidFuncTask(t *testing.T) {
	executed := false
	task := NewVoidFuncTask("void-func", "Void Func Test", func() error {
		executed = true
		return nil
	})

	ctx := context.Background()
	result, err := task.Execute(ctx)

	assert.NoError(t, err)
	assert.True(t, executed)
	assert.Equal(t, "void-func", result.TaskID)
	assert.Nil(t, result.Output)
}

// TestFuncTaskValidation 测试函数任务验证
func TestFuncTaskValidation(t *testing.T) {
	// 测试nil函数
	task := NewFuncTask("func-invalid", "Invalid Func", nil)
	assert.Error(t, task.Validate())

	// 测试有效函数
	task.Func = func(ctx context.Context) (interface{}, error) { return nil, nil }
	assert.NoError(t, task.Validate())
}

// TestTaskTypes 测试任务类型
func TestTaskTypes(t *testing.T) {
	httpTask := NewHTTPTask("http", "HTTP", "GET", "http://example.com")
	shellTask := NewShellTask("shell", "Shell", "echo test")
	funcTask := NewFuncTask("func", "Func", func(ctx context.Context) (interface{}, error) { return nil, nil })

	assert.Equal(t, TaskTypeHTTP, httpTask.GetType())
	assert.Equal(t, TaskTypeShell, shellTask.GetType())
	assert.Equal(t, TaskTypeFunc, funcTask.GetType())
}
