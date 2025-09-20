package task

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPTask HTTP 任务实现
type HTTPTask struct {
	BaseTask
	Method        string            `json:"method"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers"`
	Body          string            `json:"body"`
	SkipTLSVerify bool              `json:"skip_tls_verify"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enable     bool   `json:"enable"`
	Name       string `json:"name,omitempty"`
	SkipVerify bool   `json:"skip_verify"`
}

// NewHTTPTask 创建新的HTTP任
// NewHTTPTask 创建新的HTTP任务
func NewHTTPTask(id, name, method, url string) *HTTPTask {
	return &HTTPTask{
		BaseTask: BaseTask{
			ID:      id,
			Name:    name,
			Type:    TaskTypeHTTP,
			Timeout: 30 * time.Second,
		},
		Method:  method,
		URL:     url,
		Headers: make(map[string]string),
	}
}

// WithHeaders 设置请求头
func (h *HTTPTask) WithHeaders(headers map[string]string) *HTTPTask {
	h.Headers = headers
	return h
}

// WithBody 设置请求体
func (h *HTTPTask) WithBody(body string) *HTTPTask {
	h.Body = body
	return h
}

// WithSkipTLSVerify 设置跳过TLS验证
func (h *HTTPTask) WithSkipTLSVerify(skip bool) *HTTPTask {
	h.SkipTLSVerify = skip
	return h
}

// SetHeaders 设置请求头
func (h *HTTPTask) SetHeaders(headers map[string]string) {
	h.Headers = headers
}

// SetBody 设置请求体
func (h *HTTPTask) SetBody(body string) {
	h.Body = body
}

// Validate 验证HTTP任务参数
func (h *HTTPTask) Validate() error {
	if err := h.BaseTask.Validate(); err != nil {
		return err
	}

	if h.URL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	if h.Method == "" {
		h.Method = "GET" // 默认GET方法
	}

	h.Method = strings.ToUpper(h.Method)
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	isValid := false
	for _, method := range validMethods {
		if h.Method == method {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid HTTP method: %s", h.Method)
	}

	return nil
}

// Execute 执行HTTP任务
func (h *HTTPTask) Execute(ctx context.Context) (ExecResult, error) {
	start := time.Now()
	result := ExecResult{
		TaskID:    h.ID,
		StartTime: start,
		Metadata:  make(map[string]interface{}),
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: h.GetTimeout(),
	}
	// 配置TLS
	if h.SkipTLSVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// 创建请求
	var body io.Reader
	if h.Body != "" {
		body = bytes.NewBufferString(h.Body)
	}

	req, err := http.NewRequestWithContext(ctx, h.Method, h.URL, body)
	if err != nil {
		result.Error = err.Error()
		result.FinishTime = time.Now()
		result.Duration = result.FinishTime.Sub(start)
		return result, err
	}

	// 设置请求头
	for key, value := range h.Headers {
		req.Header.Set(key, value)
	}

	// 如果有body且没有设置Content-Type，默认设置为application/json
	if h.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		result.FinishTime = time.Now()
		result.Duration = result.FinishTime.Sub(start)
		return result, err
	}

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err.Error()
		result.FinishTime = time.Now()
		result.Duration = result.FinishTime.Sub(start)
		return result, err
	}

	// 填充结果
	result.FinishTime = time.Now()
	result.Duration = result.FinishTime.Sub(start)
	result.Metadata["status_code"] = resp.StatusCode
	result.Metadata["headers"] = resp.Header

	// 构造输出
	response := map[string]interface{}{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		"body":        string(respBody),
	}
	result.Output = response

	// 检查状态码
	if resp.StatusCode >= 400 {
		err = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		result.Error = err.Error()
		return result, err
	}

	return result, nil
}
