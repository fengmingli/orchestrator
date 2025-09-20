<template>
  <a-modal
    title="步骤详情"
    :open="visible"
    :width="800"
    @cancel="handleCancel"
    :footer="null"
  >
    <div v-if="step" class="step-detail">
      <!-- 基本信息 -->
      <a-card title="基本信息" size="small" style="margin-bottom: 16px">
        <a-descriptions :column="2" size="small">
          <a-descriptions-item label="步骤ID">{{ step.step_id }}</a-descriptions-item>
          <a-descriptions-item label="步骤名称">{{ step.step?.name || step.step_id }}</a-descriptions-item>
          <a-descriptions-item label="执行器类型">
            <a-tag :color="getExecutorColor(step.step?.executor_type)">
              {{ getExecutorName(step.step?.executor_type) }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="getStatusColor(step.status)">
              {{ getStatusName(step.status) }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="开始时间">{{ formatTime(step.started_at) }}</a-descriptions-item>
          <a-descriptions-item label="结束时间">{{ formatTime(step.finished_at) }}</a-descriptions-item>
          <a-descriptions-item label="执行时长">{{ formatDuration(step.duration) }}</a-descriptions-item>
          <a-descriptions-item label="重试次数">{{ step.retry_count || 0 }}</a-descriptions-item>
        </a-descriptions>
        
        <div v-if="step.step?.description" style="margin-top: 16px">
          <a-divider orientation="left" orientation-margin="0">描述</a-divider>
          <p>{{ step.step.description }}</p>
        </div>
      </a-card>

      <!-- 配置信息 -->
      <a-card v-if="step.step" title="配置信息" size="small" style="margin-bottom: 16px">
        <a-descriptions :column="1" size="small">
          <a-descriptions-item label="命令/URL">
            <code style="background: #f5f5f5; padding: 2px 6px; border-radius: 3px">
              {{ step.step.command || step.step.url || '-' }}
            </code>
          </a-descriptions-item>
          <a-descriptions-item v-if="step.step.method" label="HTTP方法">
            <a-tag>{{ step.step.method }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item v-if="step.step.timeout" label="超时时间">
            {{ step.step.timeout }}秒
          </a-descriptions-item>
          <a-descriptions-item v-if="step.step.retry_count" label="重试次数">
            {{ step.step.retry_count }}
          </a-descriptions-item>
        </a-descriptions>

        <!-- 环境变量 -->
        <div v-if="step.step.env && Object.keys(step.step.env).length > 0" style="margin-top: 16px">
          <a-divider orientation="left" orientation-margin="0">环境变量</a-divider>
          <a-table
            :dataSource="envData"
            :pagination="false"
            size="small"
            :show-header="false"
          >
            <a-table-column title="键" dataIndex="key" width="200">
              <template #default="{ record }">
                <code>{{ record.key }}</code>
              </template>
            </a-table-column>
            <a-table-column title="值" dataIndex="value">
              <template #default="{ record }">
                <code>{{ record.value }}</code>
              </template>
            </a-table-column>
          </a-table>
        </div>

        <!-- HTTP Headers -->
        <div v-if="step.step.headers && Object.keys(step.step.headers).length > 0" style="margin-top: 16px">
          <a-divider orientation="left" orientation-margin="0">HTTP Headers</a-divider>
          <a-table
            :dataSource="headersData"
            :pagination="false"
            size="small"
            :show-header="false"
          >
            <a-table-column title="Header" dataIndex="key" width="200">
              <template #default="{ record }">
                <code>{{ record.key }}</code>
              </template>
            </a-table-column>
            <a-table-column title="值" dataIndex="value">
              <template #default="{ record }">
                <code>{{ record.value }}</code>
              </template>
            </a-table-column>
          </a-table>
        </div>

        <!-- HTTP Body -->
        <div v-if="step.step.body" style="margin-top: 16px">
          <a-divider orientation="left" orientation-margin="0">请求体</a-divider>
          <pre style="background: #f5f5f5; padding: 12px; border-radius: 6px; max-height: 200px; overflow: auto;">{{ step.step.body }}</pre>
        </div>
      </a-card>

      <!-- 执行输出 -->
      <a-card title="执行输出" size="small" style="margin-bottom: 16px">
        <div v-if="step.output" style="margin-bottom: 16px">
          <h4>标准输出</h4>
          <pre class="output-content">{{ step.output }}</pre>
        </div>
        
        <div v-if="step.error">
          <h4>错误输出</h4>
          <pre class="error-content">{{ step.error }}</pre>
        </div>
        
        <div v-if="!step.output && !step.error">
          <a-empty description="暂无输出信息" />
        </div>
      </a-card>
    </div>

    <div v-else class="loading-container">
      <a-empty description="暂无数据" />
    </div>
  </a-modal>
</template>

<script>
import { computed } from 'vue'

export default {
  name: 'StepDetailDialog',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    step: {
      type: Object,
      default: null
    }
  },
  emits: ['update:visible'],
  setup(props, { emit }) {
    // 环境变量数据
    const envData = computed(() => {
      if (!props.step?.step?.env) return []
      return Object.entries(props.step.step.env).map(([key, value]) => ({
        key,
        value
      }))
    })

    // Headers数据
    const headersData = computed(() => {
      if (!props.step?.step?.headers) return []
      return Object.entries(props.step.step.headers).map(([key, value]) => ({
        key,
        value
      }))
    })

    // 取消
    const handleCancel = () => {
      emit('update:visible', false)
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

    return {
      envData,
      headersData,
      handleCancel,
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
.step-detail {
  max-height: 70vh;
  overflow-y: auto;
}

.loading-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
}

.output-content, .error-content {
  background: #f5f5f5;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  padding: 12px;
  max-height: 300px;
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
</style>