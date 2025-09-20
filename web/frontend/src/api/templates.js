import api from './index.js'

// 获取模板列表
export const getTemplates = (params = {}) => {
  return api.get('/templates', { params })
}

// 获取模板详情
export const getTemplate = (id) => {
  return api.get(`/templates/${id}`)
}

// 创建模板
export const createTemplate = (data) => {
  return api.post('/templates', data)
}

// 更新模板
export const updateTemplate = (id, data) => {
  return api.put(`/templates/${id}`, data)
}

// 删除模板
export const deleteTemplate = (id) => {
  return api.delete(`/templates/${id}`)
}

// 获取模板的DAG可视化数据
export const getTemplateDAG = (id) => {
  return api.get(`/templates/${id}/dag`)
}