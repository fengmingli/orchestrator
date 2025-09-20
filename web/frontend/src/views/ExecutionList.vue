<template>
  <div class="execution-list">
    <div class="header">
      <h2>执行记录</h2>
      <a-button type="primary" @click="showCreateDialog">
        <template #icon><PlusOutlined /></template>
        创建执行
      </a-button>
    </div>

    <!-- 搜索和筛选 -->
    <a-card class="filter-card">
      <a-form :model="filters" layout="inline">
        <a-form-item label="模板">
          <a-select v-model:value="filters.template_id" placeholder="请选择模板" allow-clear style="width: 200px">
            <a-select-option
              v-for="template in availableTemplates"
              :key="template.id"
              :value="template.id"
            >
              {{ template.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="filters.status" placeholder="请选择状态" allow-clear style="width: 120px">
            <a-select-option value="pending">等待中</a-select-option>
            <a-select-option value="running">运行中</a-select-option>
            <a-select-option value="success">成功</a-select-option>
            <a-select-option value="failed">失败</a-select-option>
            <a-select-option value="cancelled">已取消</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="loadExecutions">搜索</a-button>
          <a-button @click="resetFilters" style="margin-left: 8px">重置</a-button>
          <a-switch
            v-model:checked="autoRefresh"
            checked-children="自动刷新"
            un-checked-children="手动刷新"
            style="margin-left: 16px"
          />
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 执行列表 -->
    <a-card class="table-card">
      <a-table :dataSource="executions" :loading="loading" :pagination="false" rowKey="id">
        <a-table-column title="模板名称" dataIndex="template_name" :width="150" />
        <a-table-column title="状态" dataIndex="status" :width="100">
          <template #default="{ record }">
            <a-tag :color="getStatusColor(record.status)">
              {{ getStatusName(record.status) }}
            </a-tag>
          </template>
        </a-table-column>
        <a-table-column title="进度" :width="150">
          <template #default="{ record }">
            <div class="progress-info">
              <a-progress
                :percent="getProgressPercentage(record)"
                :status="getProgressStatus(record.status)"
                :stroke-width="8"
              />
              <span class="progress-text">
                {{ record.success_steps || 0 }}/{{ record.total_steps || 0 }}
              </span>
            </div>
          </template>
        </a-table-column>
        <a-table-column title="执行者" dataIndex="created_by" :width="120" />
        <a-table-column title="开始时间" dataIndex="started_at" :width="180">
          <template #default="{ record }">
            {{ formatTime(record.started_at) }}
          </template>
        </a-table-column>
        <a-table-column title="执行时长" dataIndex="duration" :width="100">
          <template #default="{ record }">
            {{ formatDuration(record.duration) }}
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="200" fixed="right">
          <template #default="{ record }">
            <a-space>
              <a-button size="small" @click="viewExecution(record)">查看</a-button>
              <a-button
                v-if="record.status === 'pending'"
                size="small"
                type="primary"
                @click="startExecution(record)"
              >
                启动
              </a-button>
              <a-button
                v-if="record.status === 'running'"
                size="small"
                danger
                @click="cancelExecution(record)"
              >
                取消
              </a-button>
              <a-button size="small" @click="viewLogs(record)">日志</a-button>
            </a-space>
          </template>
        </a-table-column>
      </a-table>

      <!-- 分页 -->
      <div class="pagination">
        <a-pagination
          v-model:current="pagination.page"
          v-model:pageSize="pagination.size"
          :total="pagination.total"
          :show-size-changer="true"
          :show-quick-jumper="true"
          :show-total="total => `共 ${total} 条记录`"
          @change="loadExecutions"
          @showSizeChange="loadExecutions"
        />
      </div>
    </a-card>

    <!-- 执行详情对话框 -->
    <execution-detail-dialog
      v-model:visible="detailVisible"
      :execution-id="selectedExecutionId"
    />

    <!-- 执行日志对话框 -->
    <execution-logs-dialog
      v-model:visible="logsVisible"
      :execution-id="selectedExecutionId"
    />

    <!-- 创建执行对话框 -->
    <execution-create-dialog
      v-model:visible="createVisible"
      :available-templates="availableTemplates"
      @success="onExecutionCreated"
    />
  </div>
</template>

<script>
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { getExecutions, startExecution, cancelExecution } from '../api/executions.js'
import { getTemplates } from '../api/templates.js'
import ExecutionDetailDialog from '../components/ExecutionDetailDialog.vue'
import ExecutionLogsDialog from '../components/ExecutionLogsDialog.vue'
import ExecutionCreateDialog from '../components/ExecutionCreateDialog.vue'

export default {
  name: 'ExecutionList',
  components: {
    ExecutionDetailDialog,
    ExecutionLogsDialog,
    ExecutionCreateDialog,
    PlusOutlined
  },
  setup() {
    const loading = ref(false)
    const executions = ref([])
    const availableTemplates = ref([])
    const detailVisible = ref(false)
    const logsVisible = ref(false)
    const createVisible = ref(false)
    const selectedExecutionId = ref('')
    const autoRefresh = ref(false)
    let refreshTimer = null

    const filters = reactive({
      template_id: '',
      status: ''
    })

    const pagination = reactive({
      page: 1,
      size: 10,
      total: 0
    })

    // 加载执行列表
    const loadExecutions = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          size: pagination.size,
          ...filters
        }
        const response = await getExecutions(params)
        executions.value = response.data.items || []
        pagination.total = response.data.total || 0
      } catch (error) {
        message.error('加载执行列表失败: ' + error.message)
      } finally {
        loading.value = false
      }
    }

    // 加载可用模板
    const loadTemplates = async () => {
      try {
        const response = await getTemplates({ size: 1000 })
        availableTemplates.value = response.data.items || []
      } catch (error) {
        console.error('加载模板列表失败:', error)
      }
    }

    // 重置筛选条件
    const resetFilters = () => {
      filters.template_id = ''
      filters.status = ''
      pagination.page = 1
      loadExecutions()
    }

    // 查看执行
    const viewExecution = (execution) => {
      selectedExecutionId.value = execution.id
      detailVisible.value = true
    }

    // 启动执行
    const startExecutionAction = async (execution) => {
      try {
        await startExecution(execution.id)
        message.success('执行已启动')
        loadExecutions()
      } catch (error) {
        message.error('启动失败: ' + error.message)
      }
    }

    // 取消执行
    const cancelExecutionAction = async (execution) => {
      try {
        await cancelExecution(execution.id)
        message.success('执行已取消')
        loadExecutions()
      } catch (error) {
        message.error('取消失败: ' + error.message)
      }
    }

    // 查看日志
    const viewLogs = (execution) => {
      selectedExecutionId.value = execution.id
      logsVisible.value = true
    }

    // 显示创建对话框
    const showCreateDialog = () => {
      createVisible.value = true
    }

    // 执行创建成功
    const onExecutionCreated = (execution) => {
      message.success('执行创建成功，正在刷新列表...')
      loadExecutions()
    }

    // 获取状态颜色
    const getStatusColor = (status) => {
      const colors = {
        pending: 'blue',
        running: 'orange',
        success: 'green',
        failed: 'red',
        cancelled: 'gray'
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
        cancelled: '已取消'
      }
      return names[status] || status
    }

    // 获取进度百分比
    const getProgressPercentage = (row) => {
      if (!row.total_steps || row.total_steps === 0) return 0
      const completed = (row.success_steps || 0) + (row.failed_steps || 0)
      return Math.round((completed / row.total_steps) * 100)
    }

    // 获取进度状态
    const getProgressStatus = (status) => {
      if (status === 'success') return 'success'
      if (status === 'failed') return 'exception'
      return 'active'
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

    // 监听自动刷新开关
    watch(autoRefresh, (enabled) => {
      if (enabled) {
        startAutoRefresh()
      } else {
        stopAutoRefresh()
      }
    })

    // 开始自动刷新
    const startAutoRefresh = () => {
      if (refreshTimer) return
      refreshTimer = setInterval(() => {
        if (autoRefresh.value) {
          loadExecutions()
        }
      }, 10000) // 每10秒刷新一次
    }

    // 停止自动刷新
    const stopAutoRefresh = () => {
      if (refreshTimer) {
        clearInterval(refreshTimer)
        refreshTimer = null
      }
    }

    onMounted(() => {
      loadTemplates()
      loadExecutions()
    })

    onUnmounted(() => {
      stopAutoRefresh()
    })

    return {
      loading,
      executions,
      availableTemplates,
      filters,
      pagination,
      detailVisible,
      logsVisible,
      createVisible,
      selectedExecutionId,
      autoRefresh,
      loadExecutions,
      resetFilters,
      viewExecution,
      startExecution: startExecutionAction,
      cancelExecution: cancelExecutionAction,
      viewLogs,
      showCreateDialog,
      onExecutionCreated,
      getStatusColor,
      getStatusName,
      getProgressPercentage,
      getProgressStatus,
      formatTime,
      formatDuration
    }
  }
}
</script>

<style scoped>
.execution-list {
  padding: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header h2 {
  margin: 0;
  color: #262626;
}

.filter-card {
  margin-bottom: 20px;
}

.table-card {
  margin-bottom: 20px;
}

.pagination {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.progress-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
}

.progress-text {
  font-size: 12px;
  color: #666;
}
</style>