import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000
})

// 请求拦截器
api.interceptors.request.use(
  config => {
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  response => {
    const { data } = response
    if (data.code === 200 || data.code === 201) {
      return data
    } else {
      throw new Error(data.message || '请求失败')
    }
  },
  error => {
    const message = error.response?.data?.message || error.message || '网络错误'
    throw new Error(message)
  }
)

export default api