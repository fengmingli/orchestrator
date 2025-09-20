import api from './index.js'

// 获取执行列表
export const getExecutions = (params = {}) => {
  return api.get('/executions', { params })
}

// 获取执行详情
export const getExecution = (id) => {
  return api.get(`/executions/${id}`)
}

// 创建执行
export const createExecution = (data) => {
  return api.post('/executions', data)
}

// 启动执行
export const startExecution = (id) => {
  return api.post(`/executions/${id}/start`)
}

// 取消执行
export const cancelExecution = (id) => {
  return api.post(`/executions/${id}/cancel`)
}

// 获取执行状态
export const getExecutionStatus = (id) => {
  return api.get(`/executions/${id}/status`)
}

// 获取执行日志
export const getExecutionLogs = (id) => {
  return api.get(`/executions/${id}/logs`)
}