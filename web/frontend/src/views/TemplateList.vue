<template>
  <div class="template-list">
    <div class="header">
      <h2>模板管理</h2>
      <a-button type="primary" @click="showCreateDialog">
        <template #icon>
          <PlusOutlined />
        </template>
        新建模板
      </a-button>
    </div>

    <!-- 搜索和筛选 -->
    <a-card class="filter-card">
      <a-form :model="filters" layout="inline">
        <a-form-item label="模板名称">
          <a-input v-model:value="filters.name" placeholder="请输入模板名称" allow-clear />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="filters.is_active" placeholder="请选择状态" allow-clear style="width: 120px">
            <a-select-option :value="true">启用</a-select-option>
            <a-select-option :value="false">禁用</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="loadTemplates">搜索</a-button>
          <a-button @click="resetFilters" style="margin-left: 8px">重置</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 模板列表 -->
    <a-card class="table-card">
      <a-table :dataSource="templates" :loading="loading" :pagination="false" rowKey="id">
        <a-table-column title="模板名称" dataIndex="name" :width="150" />
        <a-table-column title="描述" dataIndex="description" :width="200" :ellipsis="true" />
        <a-table-column title="状态" dataIndex="is_active" :width="80">
          <template #default="{ record }">
            <a-tag :color="record.is_active ? 'green' : 'red'">
              {{ record.is_active ? '启用' : '禁用' }}
            </a-tag>
          </template>
        </a-table-column>
        <a-table-column title="创建者" dataIndex="created_by" :width="120" />
        <a-table-column title="创建时间" dataIndex="created_at" :width="180">
          <template #default="{ record }">
            {{ formatTime(record.created_at) }}
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="280" fixed="right">
          <template #default="{ record }">
            <a-space>
              <a-button size="small" @click="viewTemplate(record)">查看</a-button>
              <a-button size="small" type="primary" @click="editTemplate(record)">编辑</a-button>
              <a-button size="small" @click="viewDAG(record)">DAG图</a-button>
              <a-button size="small" type="primary" ghost @click="executeTemplate(record)">执行</a-button>
              <a-button size="small" danger @click="deleteTemplateConfirm(record)">删除</a-button>
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
          @change="loadTemplates"
          @showSizeChange="loadTemplates"
        />
      </div>
    </a-card>

    <!-- 模板对话框 -->
    <TemplateDialog
      v-model:visible="templateDialogVisible"
      :mode="templateDialogMode"
      :template-data="currentTemplate"
      @success="handleTemplateSuccess"
    />

    <!-- DAG查看器 -->
    <DAGViewer
      v-model:visible="dagViewerVisible"
      :template-id="dagTemplateId"
      :template-name="dagTemplateName"
    />
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { getTemplates, deleteTemplate } from '../api/templates.js'
import { createExecution } from '../api/executions.js'
import TemplateDialog from '../components/TemplateDialog.vue'
import DAGViewer from '../components/DAGViewer.vue'

export default {
  name: 'TemplateList',
  components: {
    PlusOutlined,
    TemplateDialog,
    DAGViewer
  },
  setup() {
    const loading = ref(false)
    const templates = ref([])
    const templateDialogVisible = ref(false)
    const templateDialogMode = ref('create')
    const currentTemplate = ref({})
    const dagViewerVisible = ref(false)
    const dagTemplateId = ref('')
    const dagTemplateName = ref('')

    const filters = reactive({
      name: '',
      is_active: null
    })

    const pagination = reactive({
      page: 1,
      size: 10,
      total: 0
    })

    // 加载模板列表
    const loadTemplates = async () => {
      loading.value = true
      try {
        const params = {
          page: pagination.page,
          size: pagination.size,
          ...filters
        }
        const response = await getTemplates(params)
        templates.value = response.data.items || []
        pagination.total = response.data.total || 0
      } catch (error) {
        message.error('加载模板列表失败: ' + error.message)
      } finally {
        loading.value = false
      }
    }

    // 重置筛选条件
    const resetFilters = () => {
      filters.name = ''
      filters.is_active = null
      pagination.page = 1
      loadTemplates()
    }

    // 显示创建对话框
    const showCreateDialog = () => {
      templateDialogMode.value = 'create'
      currentTemplate.value = {}
      templateDialogVisible.value = true
    }

    // 查看模板
    const viewTemplate = (template) => {
      templateDialogMode.value = 'view'
      currentTemplate.value = template
      templateDialogVisible.value = true
    }

    // 编辑模板
    const editTemplate = (template) => {
      templateDialogMode.value = 'edit'
      currentTemplate.value = template
      templateDialogVisible.value = true
    }

    // 查看DAG图
    const viewDAG = (template) => {
      dagTemplateId.value = template.id
      dagTemplateName.value = template.name
      dagViewerVisible.value = true
    }

    // 执行模板
    const executeTemplate = async (template) => {
      Modal.confirm({
        title: '创建执行',
        content: '确定要执行此模板吗？',
        okText: '确定',
        cancelText: '取消',
        async onOk() {
          try {
            await createExecution({
              template_id: template.id,
              created_by: 'system@example.com'
            })
            message.success('执行任务已创建，请到执行记录中查看')
          } catch (error) {
            message.error('创建执行失败: ' + error.message)
          }
        }
      })
    }

    // 删除模板确认
    const deleteTemplateConfirm = (template) => {
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除模板 "${template.name}" 吗？`,
        okText: '确定',
        cancelText: '取消',
        okType: 'danger',
        onOk() {
          deleteTemplateAction(template.id)
        }
      })
    }

    // 删除模板
    const deleteTemplateAction = async (id) => {
      try {
        await deleteTemplate(id)
        message.success('删除成功')
        loadTemplates()
      } catch (error) {
        message.error('删除失败: ' + error.message)
      }
    }

    // 格式化时间
    const formatTime = (time) => {
      if (!time) return '-'
      return new Date(time).toLocaleString('zh-CN')
    }

    // 模板操作成功回调
    const handleTemplateSuccess = () => {
      templateDialogVisible.value = false
      loadTemplates()
    }

    onMounted(() => {
      loadTemplates()
    })

    return {
      loading,
      templates,
      filters,
      pagination,
      templateDialogVisible,
      templateDialogMode,
      currentTemplate,
      dagViewerVisible,
      dagTemplateId,
      dagTemplateName,
      loadTemplates,
      resetFilters,
      showCreateDialog,
      viewTemplate,
      editTemplate,
      viewDAG,
      executeTemplate,
      deleteTemplateConfirm,
      formatTime,
      handleTemplateSuccess
    }
  }
}
</script>

<style scoped>
.template-list {
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
  color: #303133;
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