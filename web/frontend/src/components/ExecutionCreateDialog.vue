<template>
  <a-modal
    title="创建执行"
    :open="visible"
    :width="600"
    @ok="handleOk"
    @cancel="handleCancel"
    :confirm-loading="loading"
    ok-text="创建"
    cancel-text="取消"
  >
    <a-form
      ref="formRef"
      :model="form"
      :rules="rules"
      :label-col="{ span: 6 }"
      :wrapper-col="{ span: 18 }"
    >
      <a-form-item label="选择模板" name="template_id">
        <a-select 
          v-model:value="form.template_id" 
          placeholder="请选择模板"
          show-search
          option-filter-prop="children"
          @change="onTemplateChange"
        >
          <a-select-option 
            v-for="template in availableTemplates" 
            :key="template.id" 
            :value="template.id"
          >
            {{ template.name }}
          </a-select-option>
        </a-select>
      </a-form-item>

      <a-form-item label="执行者" name="created_by">
        <a-input v-model:value="form.created_by" placeholder="请输入执行者姓名" />
      </a-form-item>

      <a-form-item label="备注" name="description">
        <a-textarea 
          v-model:value="form.description" 
          placeholder="请输入执行备注（可选）"
          :rows="3"
        />
      </a-form-item>
    </a-form>

    <div v-if="selectedTemplate" class="template-info">
      <a-divider>模板信息</a-divider>
      <a-descriptions :column="1" size="small">
        <a-descriptions-item label="模板名称">{{ selectedTemplate.name }}</a-descriptions-item>
        <a-descriptions-item label="描述">{{ selectedTemplate.description }}</a-descriptions-item>
        <a-descriptions-item label="步骤数量">{{ selectedTemplate.steps?.length || 0 }}</a-descriptions-item>
        <a-descriptions-item label="创建者">{{ selectedTemplate.created_by }}</a-descriptions-item>
      </a-descriptions>

      <div v-if="selectedTemplate.steps && selectedTemplate.steps.length > 0" style="margin-top: 16px">
        <h4>步骤预览</h4>
        <a-table 
          :dataSource="selectedTemplate.steps" 
          :pagination="false" 
          size="small"
          rowKey="step_id"
        >
          <a-table-column title="步骤名称" dataIndex="step_name" />
          <a-table-column title="执行模式" dataIndex="run_mode">
            <template #default="{ record }">
              <a-tag :color="record.run_mode === 'parallel' ? 'orange' : 'blue'">
                {{ record.run_mode === 'parallel' ? '并行' : '串行' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="失败策略" dataIndex="on_failure">
            <template #default="{ record }">
              {{ getFailureStrategyName(record.on_failure) }}
            </template>
          </a-table-column>
          <a-table-column title="依赖" dataIndex="dependencies">
            <template #default="{ record }">
              <span v-if="record.dependencies && record.dependencies.length > 0">
                {{ record.dependencies.join(', ') }}
              </span>
              <span v-else style="color: #ccc">无</span>
            </template>
          </a-table-column>
        </a-table>
      </div>
    </div>
  </a-modal>
</template>

<script>
import { ref, reactive, computed, watch } from 'vue'
import { message } from 'ant-design-vue'
import { createExecution } from '../api/executions.js'

export default {
  name: 'ExecutionCreateDialog',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    availableTemplates: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:visible', 'success'],
  setup(props, { emit }) {
    const formRef = ref()
    const loading = ref(false)

    const form = reactive({
      template_id: '',
      created_by: '',
      description: ''
    })

    const rules = {
      template_id: [
        { required: true, message: '请选择模板', trigger: 'change' }
      ],
      created_by: [
        { required: true, message: '请输入执行者姓名', trigger: 'blur' }
      ]
    }

    // 选中的模板信息
    const selectedTemplate = computed(() => {
      return props.availableTemplates.find(template => template.id === form.template_id)
    })

    // 监听对话框显示状态
    watch(() => props.visible, (visible) => {
      if (visible) {
        resetForm()
      }
    })

    // 重置表单
    const resetForm = () => {
      Object.assign(form, {
        template_id: '',
        created_by: '',
        description: ''
      })
      formRef.value?.resetFields()
    }

    // 模板选择变化
    const onTemplateChange = (templateId) => {
      // 可以在这里做一些额外的处理
      console.log('Selected template:', templateId)
    }

    // 确定
    const handleOk = async () => {
      try {
        await formRef.value.validate()

        loading.value = true
        const requestData = {
          template_id: form.template_id,
          created_by: form.created_by,
          description: form.description
        }

        const response = await createExecution(requestData)
        message.success('执行创建成功')
        emit('success', response.data)
        handleCancel()
      } catch (error) {
        if (error.message) {
          message.error('创建失败: ' + error.message)
        }
      } finally {
        loading.value = false
      }
    }

    // 取消
    const handleCancel = () => {
      emit('update:visible', false)
      resetForm()
    }

    // 获取失败策略名称
    const getFailureStrategyName = (strategy) => {
      const names = {
        abort: '中止执行',
        skip: '跳过继续',
        skip_but_report: '跳过但报告'
      }
      return names[strategy] || strategy
    }

    return {
      formRef,
      loading,
      form,
      rules,
      selectedTemplate,
      onTemplateChange,
      handleOk,
      handleCancel,
      getFailureStrategyName
    }
  }
}
</script>

<style scoped>
.template-info {
  margin-top: 16px;
  padding: 16px;
  background-color: #fafafa;
  border-radius: 6px;
}
</style>