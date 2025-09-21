package service

import (
	"encoding/json"
	"fmt"

	"github.com/fengmingli/orchestrator/internal/model"
	"gorm.io/gorm"
)

// TemplateService 模板服务
type TemplateService struct {
	db *gorm.DB
}

// NewTemplateService 创建模板服务
func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{db: db}
}

// CreateTemplate 创建模板
func (s *TemplateService) CreateTemplate(req *model.TemplateCreateRequest) (*model.WorkflowTemplate, error) {
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建模板
	template := &model.WorkflowTemplate{
		Name:         req.Name,
		Description:  req.Description,
		CreatorEmail: req.CreatorEmail,
		IsActive:     true,
	}

	if err := tx.Create(template).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建模板失败: %w", err)
	}

	// 添加步骤
	for _, stepReq := range req.Steps {
		// 检查步骤是否存在
		var step model.Step
		if err := tx.Where("id = ?", stepReq.StepID).First(&step).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("步骤 %s 不存在", stepReq.StepID)
		}

		// 序列化依赖关系
		depsJSON, _ := json.Marshal(stepReq.Dependencies)

		templateStep := &model.WorkflowTemplateStep{
			TemplateID:   template.ID,
			StepID:       stepReq.StepID,
			Dependencies: string(depsJSON),
			RunMode:      stepReq.RunMode,
			OnFailure:    stepReq.OnFailure,
			Order:        stepReq.Order,
		}

		if err := tx.Create(templateStep).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("添加步骤失败: %w", err)
		}
	}

	// 验证DAG是否有环
	if err := s.validateDAG(tx, template.ID); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("DAG验证失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 返回完整的模板信息
	return s.GetTemplate(template.ID)
}

// ListTemplates 获取模板列表
func (s *TemplateService) ListTemplates(page, size int, creatorEmail, isActive string) ([]*model.WorkflowTemplate, int64, error) {
	var templates []*model.WorkflowTemplate
	var total int64

	query := s.db.Model(&model.WorkflowTemplate{})

	// 筛选条件
	if creatorEmail != "" {
		query = query.Where("creator_email = ?", creatorEmail)
	}
	if isActive != "" {
		query = query.Where("is_active = ?", isActive == "true")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取模板总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at desc").Find(&templates).Error; err != nil {
		return nil, 0, fmt.Errorf("获取模板列表失败: %w", err)
	}

	// 手动加载关联数据
	for _, template := range templates {
		if err := s.loadTemplateSteps(template); err != nil {
			return nil, 0, fmt.Errorf("加载模板步骤失败: %w", err)
		}
	}

	return templates, total, nil
}

// GetTemplate 获取模板详情
func (s *TemplateService) GetTemplate(id string) (*model.WorkflowTemplate, error) {
	var template model.WorkflowTemplate
	if err := s.db.Where("id = ?", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("模板不存在")
		}
		return nil, fmt.Errorf("获取模板失败: %w", err)
	}

	// 手动加载关联数据
	if err := s.loadTemplateSteps(&template); err != nil {
		return nil, fmt.Errorf("加载模板步骤失败: %w", err)
	}

	return &template, nil
}

// UpdateTemplate 更新模板
func (s *TemplateService) UpdateTemplate(id string, req *model.TemplateUpdateRequest) (*model.WorkflowTemplate, error) {
	var template model.WorkflowTemplate
	if err := s.db.Where("id = ?", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("模板不存在")
		}
		return nil, fmt.Errorf("获取模板失败: %w", err)
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := s.db.Model(&template).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新模板失败: %w", err)
	}

	return s.GetTemplate(id)
}

// DeleteTemplate 删除模板
func (s *TemplateService) DeleteTemplate(id string) error {
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查是否有执行记录
	var count int64
	if err := tx.Model(&model.WorkflowExecution{}).Where("template_id = ?", id).Count(&count).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("检查执行记录失败: %w", err)
	}

	if count > 0 {
		tx.Rollback()
		return fmt.Errorf("模板存在执行记录，无法删除")
	}

	// 删除模板步骤
	if err := tx.Where("template_id = ?", id).Delete(&model.WorkflowTemplateStep{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除模板步骤失败: %w", err)
	}

	// 删除模板
	if err := tx.Where("id = ?", id).Delete(&model.WorkflowTemplate{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除模板失败: %w", err)
	}

	return tx.Commit().Error
}

// GetTemplateDAG 获取模板DAG结构
func (s *TemplateService) GetTemplateDAG(id string) (*model.DAGDefinition, error) {
	template, err := s.GetTemplate(id)
	if err != nil {
		return nil, err
	}

	nodes := make([]model.DAGNode, 0, len(template.Steps))
	edges := make([]model.DAGEdge, 0)

	for _, templateStep := range template.Steps {
		// 解析依赖关系
		var dependencies []string
		if templateStep.Dependencies != "" {
			json.Unmarshal([]byte(templateStep.Dependencies), &dependencies)
		}

		// 创建节点
		node := model.DAGNode{
			ID:           templateStep.StepID,
			Name:         templateStep.Step.Name,
			Type:         templateStep.Step.ExecutorType,
			Dependencies: dependencies,
			RunMode:      templateStep.RunMode,
			OnFailure:    templateStep.OnFailure,
			Data: map[string]interface{}{
				"step":        templateStep.Step,
				"order":       templateStep.Order,
				"description": templateStep.Step.Description,
			},
		}
		nodes = append(nodes, node)

		// 创建边
		for _, dep := range dependencies {
			edge := model.DAGEdge{
				ID:     fmt.Sprintf("%s-%s", dep, templateStep.StepID),
				Source: dep,
				Target: templateStep.StepID,
				Type:   "default",
			}
			edges = append(edges, edge)
		}
	}

	return &model.DAGDefinition{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// AddStepToTemplate 向模板添加步骤
func (s *TemplateService) AddStepToTemplate(templateID string, req *model.TemplateStepCreateRequest) error {
	// 检查模板是否存在
	var template model.WorkflowTemplate
	if err := s.db.Where("id = ?", templateID).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("模板不存在")
		}
		return fmt.Errorf("获取模板失败: %w", err)
	}

	// 检查步骤是否存在
	var step model.Step
	if err := s.db.Where("id = ?", req.StepID).First(&step).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("步骤不存在")
		}
		return fmt.Errorf("获取步骤失败: %w", err)
	}

	// 检查步骤是否已经在模板中
	var count int64
	if err := s.db.Model(&model.WorkflowTemplateStep{}).Where("template_id = ? AND step_id = ?", templateID, req.StepID).Count(&count).Error; err != nil {
		return fmt.Errorf("检查步骤是否存在失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("步骤已存在于模板中")
	}

	// 序列化依赖关系
	depsJSON, _ := json.Marshal(req.Dependencies)

	templateStep := &model.WorkflowTemplateStep{
		TemplateID:   templateID,
		StepID:       req.StepID,
		Dependencies: string(depsJSON),
		RunMode:      req.RunMode,
		OnFailure:    req.OnFailure,
		Order:        req.Order,
	}

	if err := s.db.Create(templateStep).Error; err != nil {
		return fmt.Errorf("添加步骤失败: %w", err)
	}

	// 验证DAG
	if err := s.validateDAG(s.db, templateID); err != nil {
		// 如果DAG验证失败，删除刚添加的步骤
		s.db.Where("id = ?", templateStep.ID).Delete(&model.WorkflowTemplateStep{})
		return fmt.Errorf("DAG验证失败: %w", err)
	}

	return nil
}

// RemoveStepFromTemplate 从模板移除步骤
func (s *TemplateService) RemoveStepFromTemplate(templateID, stepID string) error {
	// 检查步骤是否在模板中
	var templateStep model.WorkflowTemplateStep
	if err := s.db.Where("template_id = ? AND step_id = ?", templateID, stepID).First(&templateStep).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("步骤不在模板中")
		}
		return fmt.Errorf("查找模板步骤失败: %w", err)
	}

	// 检查是否有其他步骤依赖此步骤
	var count int64
	if err := s.db.Model(&model.WorkflowTemplateStep{}).Where("template_id = ? AND dependencies LIKE ?", templateID, "%"+stepID+"%").Count(&count).Error; err != nil {
		return fmt.Errorf("检查依赖关系失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("有其他步骤依赖此步骤，无法删除")
	}

	if err := s.db.Delete(&templateStep).Error; err != nil {
		return fmt.Errorf("删除步骤失败: %w", err)
	}

	return nil
}

// validateDAG 验证DAG是否有环
func (s *TemplateService) validateDAG(db *gorm.DB, templateID string) error {
	var templateSteps []model.WorkflowTemplateStep
	if err := db.Where("template_id = ?", templateID).Find(&templateSteps).Error; err != nil {
		return fmt.Errorf("获取模板步骤失败: %w", err)
	}

	// 构建依赖图
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, step := range templateSteps {
		var dependencies []string
		if step.Dependencies != "" {
			json.Unmarshal([]byte(step.Dependencies), &dependencies)
		}

		// 初始化入度
		if _, exists := inDegree[step.StepID]; !exists {
			inDegree[step.StepID] = 0
		}

		// 构建依赖关系
		for _, dep := range dependencies {
			graph[dep] = append(graph[dep], step.StepID)
			inDegree[step.StepID]++
			
			// 确保依赖的步骤也在图中
			if _, exists := inDegree[dep]; !exists {
				inDegree[dep] = 0
			}
		}
	}

	// 使用Kahn算法检测环
	queue := make([]string, 0)
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	processed := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		processed++

		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if processed != len(inDegree) {
		return fmt.Errorf("DAG中存在环路")
	}

	return nil
}

// loadTemplateSteps 手动加载模板的步骤信息
func (s *TemplateService) loadTemplateSteps(template *model.WorkflowTemplate) error {
	// 查询模板步骤
	var templateSteps []model.WorkflowTemplateStep
	if err := s.db.Where("template_id = ?", template.ID).Order("\"order\" asc").Find(&templateSteps).Error; err != nil {
		return fmt.Errorf("查询模板步骤失败: %w", err)
	}

	// 收集所有步骤ID
	stepIDs := make([]string, 0, len(templateSteps))
	for _, ts := range templateSteps {
		stepIDs = append(stepIDs, ts.StepID)
	}

	// 批量查询步骤信息
	var steps []model.Step
	if len(stepIDs) > 0 {
		if err := s.db.Where("id IN ?", stepIDs).Find(&steps).Error; err != nil {
			return fmt.Errorf("查询步骤信息失败: %w", err)
		}
	}

	// 创建步骤映射
	stepMap := make(map[string]*model.Step)
	for i := range steps {
		stepMap[steps[i].ID] = &steps[i]
	}

	// 重新构建关联关系 - 填充Step字段
	for i := range templateSteps {
		if step, exists := stepMap[templateSteps[i].StepID]; exists {
			templateSteps[i].Step = step
		}
	}

	// 设置模板的Steps字段
	template.Steps = templateSteps
	
	return nil
}