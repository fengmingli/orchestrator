import api from './index.js'

// 获取步骤列表
export const getSteps = (params = {}) => {
  return api.get('/steps', { params })
}

// 获取步骤详情
export const getStep = (id) => {
  return api.get(`/steps/${id}`)
}

// 创建步骤
export const createStep = (data) => {
  return api.post('/steps', data)
}

// 更新步骤
export const updateStep = (id, data) => {
  return api.put(`/steps/${id}`, data)
}

// 删除步骤
export const deleteStep = (id) => {
  return api.delete(`/steps/${id}`)
}

// 验证步骤
export const validateStep = (id) => {
  return api.post(`/steps/${id}/validate`)
}