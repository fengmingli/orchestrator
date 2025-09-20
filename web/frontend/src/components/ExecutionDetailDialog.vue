<template>
  <a-modal
    title="执行详情"
    :open="visible"
    :width="1000"
    @cancel="handleCancel"
    :footer="null"
  >
    <div v-if="execution" class="execution-detail">
      <!-- 基本信息 -->
      <a-card title="基本信息" size="small" style="margin-bottom: 16px">
        <a-descriptions :column="2" size="small">
          <a-descriptions-item label="执行ID">{{ execution.id }}</a-descriptions-item>
          <a-descriptions-item label="模板名称">{{ execution.template?.name }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="getStatusColor(execution.status)">
              {{ getStatusName(execution.status) }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="创建者">{{ execution.created_by }}</a-descriptions-item>
          <a-descriptions-item label="开始时间">{{ formatTime(execution.started_at) }}</a-descriptions-item>
          <a-descriptions-item label="结束时间">{{ formatTime(execution.finished_at) }}</a-descriptions-item>
          <a-descriptions-item label="执行时长">{{ formatDuration(execution.duration) }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatTime(execution.created_at) }}</a-descriptions-item>
        </a-descriptions>
        <div v-if="execution.error" style="margin-top: 16px">
          <a-alert
            message="执行错误"
            :description="execution.error"
            type="error"
            show-icon
          />
        </div>
      </a-card>

      <!-- 执行进度 -->
      <a-card title="执行进度" size="small" style="margin-bottom: 16px">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-progress
              type="circle"
              :percent="progressPercentage"
              :status="getProgressStatus(execution.status)"
              :stroke-width="8"
            >
              <template #format="percent">
                <span style="font-size: 16px">{{ percent }}%</span>
              </template>
            </a-progress>
          </a-col>
          <a-col :span="12">
            <a-space direction="vertical" style="width: 100%">
              <a-statistic
                title="总步骤数"
                :value="execution.steps?.length || 0"
                :value-style="{ color: '#1890ff' }"
              />
              <a-statistic
                title="成功步骤"
                :value="successSteps"
                :value-style="{ color: '#52c41a' }"
              />
              <a-statistic
                title="失败步骤"
                :value="failedSteps"
                :value-style="{ color: '#ff4d4f' }"
              />
            </a-space>
          </a-col>
        </a-row>
      </a-card>

      <!-- 步骤详情 -->
      <a-card title="步骤详情" size="small">
        <a-table 
          :dataSource="execution.steps" 
          :pagination="false" 
          size="small"
          rowKey="id"
        >
          <a-table-column title="步骤名称" dataIndex="step_name" width="150">
            <template #default="{ record }">
              {{ record.step?.name || record.step_id }}
            </template>
          </a-table-column>
          <a-table-column title="类型" width="100">
            <template #default="{ record }">
              <a-tag :color="getExecutorColor(record.step?.executor_type)">
                {{ getExecutorName(record.step?.executor_type) }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="状态" dataIndex="status" width="100">
            <template #default="{ record }">
              <a-tag :color="getStatusColor(record.status)">
                {{ getStatusName(record.status) }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="开始时间" width="150">
            <template #default="{ record }">
              {{ formatTime(record.started_at) }}
            </template>
          </a-table-column>
          <a-table-column title="结束时间" width="150">
            <template #default="{ record }">
              {{ formatTime(record.finished_at) }}
            </template>
          </a-table-column>
          <a-table-column title="时长" width="100">
            <template #default="{ record }">
              {{ formatDuration(record.duration) }}
            </template>
          </a-table-column>
          <a-table-column title="重试次数" dataIndex="retry_count" width="80" />
          <a-table-column title="操作" width="100" fixed="right">
            <template #default="{ record }">
              <a-button size="small" @click="viewStepDetail(record)">详情</a-button>
            </template>
          </a-table-column>
        </a-table>
      </a-card>
    </div>

    <div v-else class="loading-container">
      <a-spin size="large" />
    </div>

    <!-- 步骤详情对话框 -->
    <step-detail-dialog
      v-model:visible="stepDetailVisible"
      :step="selectedStep"
    />
  </a-modal>
</template>

<script>
import { ref, computed, watch } from 'vue'
import { getExecution } from '../api/executions.js'
import { message } from 'ant-design-vue'
import StepDetailDialog from './StepDetailDialog.vue'

export default {
  name: 'ExecutionDetailDialog',
  components: {
    StepDetailDialog
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
    const execution = ref(null)
    const loading = ref(false)
    const stepDetailVisible = ref(false)
    const selectedStep = ref(null)

    // 成功步骤数
    const successSteps = computed(() => {
      return execution.value?.steps?.filter(step => step.status === 'success').length || 0
    })

    // 失败步骤数
    const failedSteps = computed(() => {
      return execution.value?.steps?.filter(step => step.status === 'failed').length || 0
    })

    // 进度百分比
    const progressPercentage = computed(() => {
      if (!execution.value?.steps || execution.value.steps.length === 0) return 0
      const completed = execution.value.steps.filter(step => 
        ['success', 'failed', 'skipped'].includes(step.status)
      ).length
      return Math.round((completed / execution.value.steps.length) * 100)
    })

    // 监听执行ID变化
    watch([() => props.visible, () => props.executionId], ([visible, executionId]) => {
      if (visible && executionId) {
        loadExecution()
      }
    })

    // 加载执行详情
    const loadExecution = async () => {
      if (!props.executionId) return

      loading.value = true
      try {
        const response = await getExecution(props.executionId)
        execution.value = response.data
      } catch (error) {
        message.error('加载执行详情失败: ' + error.message)
      } finally {
        loading.value = false
      }
    }

    // 查看步骤详情
    const viewStepDetail = (step) => {
      selectedStep.value = step
      stepDetailVisible.value = true
    }

    // 取消
    const handleCancel = () => {
      emit('update:visible', false)
      execution.value = null
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

    // 获取进度状态
    const getProgressStatus = (status) => {
      if (status === 'success') return 'success'
      if (status === 'failed') return 'exception'
      return 'active'
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
      execution,
      loading,
      stepDetailVisible,
      selectedStep,
      successSteps,
      failedSteps,
      progressPercentage,
      viewStepDetail,
      handleCancel,
      getStatusColor,
      getStatusName,
      getProgressStatus,
      getExecutorColor,
      getExecutorName,
      formatTime,
      formatDuration
    }
  }
}
</script>

<style scoped>
.execution-detail {
  max-height: 70vh;
  overflow-y: auto;
}

.loading-container {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
}
</style>