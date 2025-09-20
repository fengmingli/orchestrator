package service

import "gorm.io/gorm"

// Services 服务集合
type Services struct {
	StepService         *StepService
	TemplateService     *TemplateService
	ExecutionService    *ExecutionService
	OrchestratorService *OrchestratorService
}

// NewServices 创建服务集合
func NewServices(db *gorm.DB) *Services {
	return &Services{
		StepService:         NewStepService(db),
		TemplateService:     NewTemplateService(db),
		ExecutionService:    NewExecutionService(db),
		OrchestratorService: NewOrchestratorService(db),
	}
}