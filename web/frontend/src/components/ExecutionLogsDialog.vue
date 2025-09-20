<template>
  <a-modal
    title="执行日志"
    :open="visible"
    :width="1200"
    @cancel="handleCancel"
    :footer="null"
  >
    <div class="logs-container">
      <!-- 工具栏 -->
      <div class="toolbar">
        <a-space>
          <a-button @click="refreshLogs" :loading="loading">
            <template #icon><ReloadOutlined /></template>
            刷新
          </a-button>
          <a-select
            v-model:value="statusFilter"
            placeholder="筛选状态"
            style="width: 120px"
            allow-clear
          >
            <a-select-option value="pending">等待中</a-select-option>
            <a-select-option value="running">运行中</a-select-option>
            <a-select-option value="success">成功</a-select-option>
            <a-select-option value="failed">失败</a-select-option>
            <a-select-option value="skipped">跳过</a-select-option>
          </a-select>
          <a-switch
            v-model:checked="autoRefresh"
            checked-children="自动刷新"
            un-checked-children="手动刷新"
          />
        </a-space>
      </div>

      <!-- 日志列表 -->
      <div class="logs-content">
        <a-timeline v-if="filteredLogs.length > 0">
          <a-timeline-item
            v-for="log in filteredLogs"
            :key="log.id"
            :color="getTimelineColor(log.status)"
          >
            <template #dot>
              <div class="timeline-dot" :class="log.status">
                <CheckCircleOutlined v-if="log.status === 'success'" />
                <CloseCircleOutlined v-else-if="log.status === 'failed'" />
                <SyncOutlined v-else-if="log.status === 'running'" :spin="true" />
                <ClockCircleOutlined v-else />
              </div>
            </template>
            
            <div class="log-item">
              <div class="log-header">
                <a-space>
                  <span class="step-name">{{ log.step?.name || log.step_id }}</span>
                  <a-tag :color="getExecutorColor(log.step?.executor_type)">
                    {{ getExecutorName(log.step?.executor_type) }}
                  </a-tag>
                  <a-tag :color="getStatusColor(log.status)">
                    {{ getStatusName(log.status) }}
                  </a-tag>
                  <span v-if="log.retry_count > 0" class="retry-info">
                    重试: {{ log.retry_count }}
                  </span>
                </a-space>
                <span class="log-time">{{ formatTime(log.started_at) }}</span>
              </div>
              
              <div v-if="log.step?.description" class="step-description">
                {{ log.step.description }}
              </div>

              <div class="log-details">
                <a-row :gutter="16">
                  <a-col :span="8">
                    <span class="detail-label">开始时间:</span>
                    <span>{{ formatTime(log.started_at) }}</span>
                  </a-col>
                  <a-col :span="8">
                    <span class="detail-label">结束时间:</span>
                    <span>{{ formatTime(log.finished_at) }}</span>
                  </a-col>
                  <a-col :span="8">
                    <span class="detail-label">执行时长:</span>
                    <span>{{ formatDuration(log.duration) }}</span>
                  </a-col>
                </a-row>
              </div>

              <!-- 输出内容 -->
              <div v-if="log.output || log.error" class="log-output">
                <a-collapse ghost>
                  <a-collapse-panel key="output" header="查看输出">
                    <div v-if="log.output" class="output-section">
                      <h4>标准输出</h4>
                      <pre class="output-content">{{ log.output }}</pre>
                    </div>
                    <div v-if="log.error" class="error-section">
                      <h4>错误输出</h4>
                      <pre class="error-content">{{ log.error }}</pre>
                    </div>
                  </a-collapse-panel>
                </a-collapse>
              </div>
            </div>
          </a-timeline-item>
        </a-timeline>

        <div v-else-if="loading" class="loading-container">
          <a-spin size="large" />
        </div>

        <div v-else class="empty-container">
          <a-empty description="暂无日志数据" />
        </div>
      </div>
    </div>
  </a-modal>
</template>

<script>
import { ref, computed, watch, onUnmounted } from 'vue'
import { 
  ReloadOutlined, 
  CheckCircleOutlined, 
  CloseCircleOutlined, 
  SyncOutlined, 
  ClockCircleOutlined 
} from '@ant-design/icons-vue'
import { getExecutionLogs } from '../api/executions.js'
import { message } from 'ant-design-vue'

export default {
  name: 'ExecutionLogsDialog',
  components: {
    ReloadOutlined,
    CheckCircleOutlined,
    CloseCircleOutlined,
    SyncOutlined,
    ClockCircleOutlined
  },
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    executionId: {
      type: String,
      default: ''
    }
  },
  emits: ['update:visible'],
  setup(props, { emit }) {
    const logs = ref([])
    const loading = ref(false)
    const statusFilter = ref('')
    const autoRefresh = ref(false)
    let refreshTimer = null

    // 过滤后的日志
    const filteredLogs = computed(() => {
      if (!statusFilter.value) return logs.value
      return logs.value.filter(log => log.status === statusFilter.value)
    })

    // 监听对话框显示状态
    watch([() => props.visible, () => props.executionId], ([visible, executionId]) => {
      if (visible && executionId) {
        refreshLogs()
        startAutoRefresh()
      } else {
        stopAutoRefresh()
      }
    })

    // 监听自动刷新开关
    watch(autoRefresh, (enabled) => {
      if (enabled && props.visible) {
        startAutoRefresh()
      } else {
        stopAutoRefresh()
      }
    })

    // 加载日志
    const refreshLogs = async () => {
      if (!props.executionId) return

      loading.value = true
      try {
        const response = await getExecutionLogs(props.executionId)
        logs.value = response.data || []
      } catch (error) {
        message.error('加载执行日志失败: ' + error.message)
      } finally {
        loading.value = false
      }
    }

    // 开始自动刷新
    const startAutoRefresh = () => {
      if (refreshTimer) return
      refreshTimer = setInterval(() => {
        if (props.visible && autoRefresh.value) {
          refreshLogs()
        }
      }, 5000) // 每5秒刷新一次
    }

    // 停止自动刷新
    const stopAutoRefresh = () => {
      if (refreshTimer) {
        clearInterval(refreshTimer)
        refreshTimer = null
      }
    }

    // 取消
    const handleCancel = () => {
      emit('update:visible', false)
      logs.value = []
      statusFilter.value = ''
      autoRefresh.value = false
      stopAutoRefresh()
    }

    // 获取时间轴颜色
    const getTimelineColor = (status) => {
      const colors = {
        pending: 'blue',
        running: 'orange',
        success: 'green',
        failed: 'red',
        cancelled: 'gray',
        skipped: 'purple'
      }
      return colors[status] || 'blue'
    }

    // 获取状态颜色
    const getStatusColor = (status) => {
      const colors = {
        pending: 'blue',
        running: 'orange',
        success: 'green',
        failed: 'red',
        cancelled: 'gray',
        skipped: 'purple'
      }
      return colors[status] || 'blue'
    }

    // 获取状态名称
    const getStatusName = (status) => {
      const names = {
        pending: '等待中',
        running: '运行中',
        success: '成功',
        failed: '失败',
        cancelled: '已取消',
        skipped: '跳过'
      }
      return names[status] || status
    }

    // 获取执行器颜色
    const getExecutorColor = (type) => {
      const colors = {
        shell: 'green',
        http: 'blue',
        func: 'orange'
      }
      return colors[type] || 'default'
    }

    // 获取执行器名称
    const getExecutorName = (type) => {
      const names = {
        shell: 'Shell',
        http: 'HTTP',
        func: 'Function'
      }
      return names[type] || type
    }

    // 格式化时间
    const formatTime = (time) => {
      if (!time) return '-'
      return new Date(time).toLocaleString('zh-CN')
    }

    // 格式化执行时长
    const formatDuration = (duration) => {
      if (!duration) return '-'
      const seconds = Math.floor(duration / 1000)
      const minutes = Math.floor(seconds / 60)
      const hours = Math.floor(minutes / 60)
      
      if (hours > 0) {
        return `${hours}h ${minutes % 60}m ${seconds % 60}s`
      } else if (minutes > 0) {
        return `${minutes}m ${seconds % 60}s`
      } else {
        return `${seconds}s`
      }
    }

    // 清理定时器
    onUnmounted(() => {
      stopAutoRefresh()
    })

    return {
      logs,
      loading,
      statusFilter,
      autoRefresh,
      filteredLogs,
      refreshLogs,
      handleCancel,
      getTimelineColor,
      getStatusColor,
      getStatusName,
      getExecutorColor,
      getExecutorName,
      formatTime,
      formatDuration
    }
  }
}
</script>

<style scoped>
.logs-container {
  max-height: 70vh;
  display: flex;
  flex-direction: column;
}

.toolbar {
  padding: 16px 0;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 16px;
}

.logs-content {
  flex: 1;
  overflow-y: auto;
}

.timeline-dot {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: white;
  border: 2px solid;
}

.timeline-dot.success {
  border-color: #52c41a;
  color: #52c41a;
}

.timeline-dot.failed {
  border-color: #ff4d4f;
  color: #ff4d4f;
}

.timeline-dot.running {
  border-color: #fa8c16;
  color: #fa8c16;
}

.timeline-dot.pending {
  border-color: #1890ff;
  color: #1890ff;
}

.log-item {
  background: #fafafa;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 8px;
}

.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.step-name {
  font-weight: 500;
  font-size: 16px;
}

.log-time {
  color: #8c8c8c;
  font-size: 12px;
}

.retry-info {
  color: #fa8c16;
  font-size: 12px;
}

.step-description {
  color: #666;
  margin-bottom: 12px;
  font-style: italic;
}

.log-details {
  margin-bottom: 12px;
}

.detail-label {
  color: #8c8c8c;
  margin-right: 8px;
}

.log-output {
  border-top: 1px solid #f0f0f0;
  padding-top: 12px;
}

.output-content, .error-content {
  background: #f5f5f5;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  padding: 12px;
  max-height: 200px;
  overflow: auto;
  font-family: 'Courier New', Courier, monospace;
  font-size: 12px;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-all;
}

.error-content {
  background: #fff2f0;
  border-color: #ffccc7;
  color: #ff4d4f;
}

.loading-container, .empty-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
}
</style>