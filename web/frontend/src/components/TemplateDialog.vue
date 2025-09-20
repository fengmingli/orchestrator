<template>
  <a-modal
    :title="dialogTitle"
    :open="visible"
    :width="800"
    @ok="handleOk"
    @cancel="handleCancel"
    :confirm-loading="loading"
    ok-text="确定"
    cancel-text="取消"
  >
    <a-form
      ref="formRef"
      :model="form"
      :rules="rules"
      :label-col="{ span: 6 }"
      :wrapper-col="{ span: 18 }"
    >
      <a-form-item label="模板名称" name="name">
        <a-input v-model:value="form.name" placeholder="请输入模板名称" />
      </a-form-item>
      
      <a-form-item label="模板描述" name="description">
        <a-textarea 
          v-model:value="form.description" 
          placeholder="请输入模板描述" 
          :rows="3"
        />
      </a-form-item>
      
      <a-form-item label="创建者邮箱" name="creator_email">
        <a-input 
          v-model:value="form.creator_email" 
          placeholder="请输入创建者邮箱"
          :disabled="mode === 'edit'"
        />
      </a-form-item>
      
      <a-form-item label="状态" name="is_active" v-if="mode === 'edit'">
        <a-switch v-model:checked="form.is_active" />
        <span style="margin-left: 8px">{{ form.is_active ? '启用' : '禁用' }}</span>
      </a-form-item>

      <a-divider v-if="mode === 'create'">步骤配置</a-divider>
      
      <div v-if="mode === 'create'" class="steps-section">
        <div class="steps-header">
          <span>模板步骤</span>
          <a-button type="primary" size="small" @click="showAddStepDialog">
            <template #icon>
              <PlusOutlined />
            </template>
            添加步骤
          </a-button>
        </div>
        
        <a-table
          :dataSource="form.steps"
          :pagination="false"
          :scroll="{ y: 300 }"
          rowKey="step_id"
          size="small"
        >
          <a-table-column title="步骤名称" dataIndex="step_name" :width="150" />
          <a-table-column title="执行模式" dataIndex="run_mode" :width="100">
            <template #default="{ record }">
              <a-tag :color="record.run_mode === 'parallel' ? 'blue' : 'green'">
                {{ record.run_mode === 'parallel' ? '并行' : '串行' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="失败策略" dataIndex="on_failure" :width="100">
            <template #default="{ record }">
              <a-tag :color="getFailureColor(record.on_failure)">
                {{ getFailureText(record.on_failure) }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="依赖" dataIndex="dependencies" :width="120">
            <template #default="{ record }">
              <span v-if="record.dependencies && record.dependencies.length > 0">
                {{ record.dependencies.join(', ') }}
              </span>
              <span v-else style="color: #999">无</span>
            </template>
          </a-table-column>
          <a-table-column title="排序" dataIndex="order" :width="80" />
          <a-table-column title="操作" :width="120" fixed="right">
            <template #default="{ record, index }">
              <a-space>
                <a-button size="small" @click="editTemplateStep(record, index)">编辑</a-button>
                <a-button size="small" danger @click="removeTemplateStep(index)">删除</a-button>
              </a-space>
            </template>
          </a-table-column>
        </a-table>
      </div>
    </a-form>

    <!-- 添加步骤对话框 -->
    <TemplateStepDialog
      v-model:visible="stepDialogVisible"
      :mode="stepDialogMode"
      :step-data="currentStepData"
      :available-steps="availableSteps"
      :existing-steps="form.steps"
      @success="handleStepSuccess"
    />
  </a-modal>
</template>

<script>
import { ref, reactive, computed, watch } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { createTemplate, updateTemplate } from '../api/templates.js'
import { getSteps } from '../api/steps.js'
import TemplateStepDialog from './TemplateStepDialog.vue'

export default {
  name: 'TemplateDialog',
  components: {
    PlusOutlined,
    TemplateStepDialog
  },
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    mode: {
      type: String,
      default: 'create' // create | edit | view
    },
    templateData: {
      type: Object,
      default: () => ({})
    }
  },
  emits: ['update:visible', 'success'],
  setup(props, { emit }) {
    const formRef = ref()
    const loading = ref(false)
    const stepDialogVisible = ref(false)
    const stepDialogMode = ref('create')
    const currentStepData = ref({})
    const currentStepIndex = ref(-1)
    const availableSteps = ref([])

    const form = reactive({
      name: '',
      description: '',
      creator_email: '',
      is_active: true,
      steps: []
    })

    const rules = {
      name: [
        { required: true, message: '请输入模板名称', trigger: 'blur' }
      ],
      description: [
        { required: true, message: '请输入模板描述', trigger: 'blur' }
      ],
      creator_email: [
        { required: true, message: '请输入创建者邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
      ]
    }

    const dialogTitle = computed(() => {
      const titles = {
        create: '创建模板',
        edit: '编辑模板',
        view: '查看模板'
      }
      return titles[props.mode] || '模板'
    })

    // 监听模板数据变化
    watch(() => props.templateData, (newData) => {
      if (newData && Object.keys(newData).length > 0) {
        Object.assign(form, {
          name: newData.name || '',
          description: newData.description || '',
          creator_email: newData.creator_email || '',
          is_active: newData.is_active !== undefined ? newData.is_active : true,
          steps: newData.steps ? newData.steps.map(step => ({
            step_id: step.step_id,
            step_name: step.step?.name || '',
            run_mode: step.run_mode || 'serial',
            on_failure: step.on_failure || 'abort',
            dependencies: step.dependencies ? JSON.parse(step.dependencies) : [],
            order: step.order || 0
          })) : []
        })
      }
    }, { immediate: true })

    // 监听对话框显示状态
    watch(() => props.visible, (visible) => {
      if (visible) {
        loadAvailableSteps()
        if (props.mode === 'create') {
          resetForm()
        }
      }
    })

    // 重置表单
    const resetForm = () => {
      Object.assign(form, {
        name: '',
        description: '',
        creator_email: '',
        is_active: true,
        steps: []
      })
      formRef.value?.resetFields()
    }

    // 加载可用步骤
    const loadAvailableSteps = async () => {
      try {
        const response = await getSteps({ page: 1, size: 1000 })
        availableSteps.value = response.data.items || []
      } catch (error) {
        message.error('加载步骤列表失败: ' + error.message)
      }
    }

    // 确定
    const handleOk = async () => {
      try {
        await formRef.value.validate()
        loading.value = true

        const templateData = {
          name: form.name,
          description: form.description,
          creator_email: form.creator_email
        }

        if (props.mode === 'create') {
          templateData.steps = form.steps.map(step => ({
            step_id: step.step_id,
            run_mode: step.run_mode,
            on_failure: step.on_failure,
            dependencies: step.dependencies,
            order: step.order
          }))
          await createTemplate(templateData)
          message.success('模板创建成功')
        } else if (props.mode === 'edit') {
          const updateData = {
            name: form.name,
            description: form.description,
            is_active: form.is_active
          }
          await updateTemplate(props.templateData.id, updateData)
          message.success('模板更新成功')
        }

        emit('success')
        handleCancel()
      } catch (error) {
        message.error('操作失败: ' + error.message)
      } finally {
        loading.value = false
      }
    }

    // 取消
    const handleCancel = () => {
      emit('update:visible', false)
      resetForm()
    }

    // 显示添加步骤对话框
    const showAddStepDialog = () => {
      stepDialogMode.value = 'create'
      currentStepData.value = {}
      currentStepIndex.value = -1
      stepDialogVisible.value = true
    }

    // 编辑模板步骤
    const editTemplateStep = (step, index) => {
      stepDialogMode.value = 'edit'
      currentStepData.value = { ...step }
      currentStepIndex.value = index
      stepDialogVisible.value = true
    }

    // 删除模板步骤
    const removeTemplateStep = (index) => {
      form.steps.splice(index, 1)
    }

    // 步骤操作成功
    const handleStepSuccess = (stepData) => {
      if (stepDialogMode.value === 'create') {
        form.steps.push(stepData)
      } else if (stepDialogMode.value === 'edit') {
        form.steps[currentStepIndex.value] = stepData
      }
    }

    // 获取失败策略颜色
    const getFailureColor = (onFailure) => {
      const colors = {
        abort: 'red',
        skip: 'orange',
        skip_but_report: 'yellow'
      }
      return colors[onFailure] || 'default'
    }

    // 获取失败策略文本
    const getFailureText = (onFailure) => {
      const texts = {
        abort: '中止',
        skip: '跳过',
        skip_but_report: '跳过但报告'
      }
      return texts[onFailure] || onFailure
    }

    return {
      formRef,
      loading,
      form,
      rules,
      dialogTitle,
      stepDialogVisible,
      stepDialogMode,
      currentStepData,
      availableSteps,
      handleOk,
      handleCancel,
      showAddStepDialog,
      editTemplateStep,
      removeTemplateStep,
      handleStepSuccess,
      getFailureColor,
      getFailureText
    }
  }
}
</script>

<style scoped>
.steps-section {
  margin-top: 16px;
}

.steps-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-weight: 500;
}
</style>