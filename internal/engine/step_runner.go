package engine

import (
	"context"

	"github.com/fengmingli/orchestrator/internal/model"
)

type StepRunner interface {
	Run(ctx context.Context, step model.TemplateStep, executionID string) (output string, err error)
}

var runners = make(map[string]StepRunner)

func RegisterStepRunner(typ string, r StepRunner) {
	runners[typ] = r
}

func GetRunner(typ string) (StepRunner, bool) {
	r, ok := runners[typ]
	return r, ok
}
