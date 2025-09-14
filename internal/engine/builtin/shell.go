package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"

	"github.com/fengmingli/orchestrator/internal/engine"
	"github.com/fengmingli/orchestrator/internal/model"
)

func init() {
	engine.RegisterStepRunner("shell", &ShellRunner{})
}

type ShellRunner struct{}

func (s *ShellRunner) Run(ctx context.Context, step model.TemplateStep, executionID string) (string, error) {
	var cfg struct {
		Script string `json:"script"`
	}
	json.Unmarshal(step.Parameters, &cfg)
	cmd := exec.CommandContext(ctx, "sh", "-c", cfg.Script)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
