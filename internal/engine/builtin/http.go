package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fengmingli/orchestrator/internal/engine"
	"github.com/fengmingli/orchestrator/internal/model"
)

func init() {
	engine.RegisterStepRunner("http", &HTTPRunner{})
}

type HTTPConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type HTTPRunner struct{}

func (h *HTTPRunner) Run(ctx context.Context, step model.TemplateStep, executionID string) (string, error) {
	var cfg HTTPConfig
	if err := json.Unmarshal(step.Parameters, &cfg); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, bytes.NewReader([]byte(cfg.Body)))
	if err != nil {
		return "", err
	}
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}
	return string(b), nil
}
