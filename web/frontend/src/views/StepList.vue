<template>
  <div class="step-list">
    <div class="header">
      <h2>步骤管理</h2>
      <a-button type="primary" @click="showCreateDialog">
        <template #icon>
          <PlusOutlined />
        </template>
        新建步骤
      </a-button>
    </div>

    <!-- 搜索和筛选 -->
    <a-card class="filter-card">
      <a-form :model="filters" layout="inline">
        <a-form-item label="步骤名称">
          <a-input v-model:value="filters.name" placeholder="请输入步骤名称" allow-clear />
        </a-form-item>
        <a-form-item label="执行器类型">
          <a-select v-model:value="filters.executor_type" placeholder="请选择执行器类型" allow-clear style="width: 150px">
            <a-select-option value="shell">Shell</a-select-option>
            <a-select-option value="http">HTTP</a-select-option>
            <a-select-option value="func">Function</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="loadSteps">搜索</a-button>
          <a-button @click="resetFilters" style="margin-left: 8px">重置</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 步骤列表 -->
    <a-card class="table-card">
      <a-table :dataSource="steps" :loading="loading" :pagination="false" rowKey="id">
        <a-table-column title="步骤名称" dataIndex="name" :width="150" />
        <a-table-column title="描述" dataIndex="description" :width="200" :ellipsis="true" />
        <a-table-column title="执行器类型" dataIndex="executor_type" :width="120">
          <template #default="{ record }">
            <a-tag :color="getExecutorColor(record.executor_type)">
              {{ getExecutorName(record.executor_type) }}
            </a-tag>
          </template>
        </a-table-column>
        <a-table-column title="创建者" dataIndex="created_by" :width="120" />
        <a-table-column title="创建时间" dataIndex="created_at" :width="180">
          <template #default="{ record }">
            {{ formatTime(record.created_at) }}
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="250" fixed="right">
          <template #default="{ record }">
            <a-space>
              <a-button size="small" @click="viewStep(record)">查看</a-button>
              <a-button size="small" type="primary" @click="editStep(record)">编辑</a-button>
              <a-button size="small" @click="validateStep(record)">验证</a-button>
              <a-button size="small" danger @click="deleteStepConfirm(record)">删除</a-button>
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
          @change="loadSteps"
          @showSizeChange="loadSteps"
        />
      </div>
    </a-card>

    <!-- 步骤表单对话框 -->
    <StepFormDialog
      v-model:visible="stepDialogVisible"
      :mode="stepDialogMode"
      :step-data="currentStep"
      @success="handleStepSuccess"
    />
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { getSteps, deleteStep, validateStep } from '../api/steps.js'
import StepFormDialog from '../components/StepFormDialog.vue'

export default {
  name: 'StepList',
  components: {
    PlusOutlined,
    StepFormDialog
  },
  setup() {
    const loading = ref(false)
    const steps = ref([])
    const stepDialogVisible = ref(false)
    const stepDialogMode = ref('create')
    const currentStep = ref({})

    const filters = reactive({
      name: '',
      executor_type: ''
    })

    const pagination = reactive({
      page: 1,
      size: 10,
      total: 0
    })

    // 加载步骤列表
    const loadSteps = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          size: pagination.size,
          ...filters
        }
        const response = await getSteps(params)
        steps.value = response.data.items || []
        pagination.total = response.data.total || 0
      } catch (error) {
        message.error('加载步骤列表失败: ' + error.message)
      } finally {
        loading.value = false
      }
    }

    // 重置筛选条件
    const resetFilters = () => {
      filters.name = ''
      filters.executor_type = ''
      pagination.page = 1
      loadSteps()
    }

    // 显示创建对话框
    const showCreateDialog = () => {
      stepDialogMode.value = 'create'
      currentStep.value = {}
      stepDialogVisible.value = true
    }

    // 查看步骤
    const viewStep = (step) => {
      stepDialogMode.value = 'view'
      currentStep.value = step
      stepDialogVisible.value = true
    }

    // 编辑步骤
    const editStep = (step) => {
      stepDialogMode.value = 'edit'
      currentStep.value = step
      stepDialogVisible.value = true
    }

    // 验证步骤
    const validateStepAction = async (step) => {
      try {
        const response = await validateStep(step.id)
        if (response.data.valid) {
          message.success('步骤验证通过')
        } else {
          message.warning('步骤验证失败')
        }
      } catch (error) {
        message.error('验证失败: ' + error.message)
      }
    }

    // 删除步骤确认
    const deleteStepConfirm = (step) => {
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除步骤 "${step.name}" 吗？`,
        okText: '确定',
        cancelText: '取消',
        okType: 'danger',
        onOk() {
          deleteStepAction(step.id)
        }
      })
    }

    // 删除步骤
    const deleteStepAction = async (id) => {
      try {
        await deleteStep(id)
        message.success('删除成功')
        loadSteps()
      } catch (error) {
        message.error('删除失败: ' + error.message)
      }
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

    // 步骤操作成功回调
    const handleStepSuccess = () => {
      stepDialogVisible.value = false
      loadSteps()
    }

    onMounted(() => {
      loadSteps()
    })

    return {
      loading,
      steps,
      filters,
      pagination,
      stepDialogVisible,
      stepDialogMode,
      currentStep,
      loadSteps,
      resetFilters,
      showCreateDialog,
      viewStep,
      editStep,
      validateStep: validateStepAction,
      deleteStepConfirm,
      getExecutorColor,
      getExecutorName,
      formatTime,
      handleStepSuccess
    }
  }
}
</script>

<style scoped>
.step-list {
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
</style>