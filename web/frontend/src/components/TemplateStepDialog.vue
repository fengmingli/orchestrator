<template>
  <a-modal
    :title="dialogTitle"
    :open="visible"
    :width="600"
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
      <a-form-item label="选择步骤" name="step_id">
        <a-select 
          v-model:value="form.step_id" 
          placeholder="请选择步骤"
          show-search
          option-filter-prop="children"
          @change="onStepChange"
        >
          <a-select-option 
            v-for="step in availableSteps" 
            :key="step.id" 
            :value="step.id"
          >
            {{ step.name }} ({{ step.executor_type }})
          </a-select-option>
        </a-select>
      </a-form-item>

      <a-form-item label="执行模式" name="run_mode">
        <a-radio-group v-model:value="form.run_mode">
          <a-radio value="serial">串行</a-radio>
          <a-radio value="parallel">并行</a-radio>
        </a-radio-group>
      </a-form-item>

      <a-form-item label="失败策略" name="on_failure">
        <a-select v-model:value="form.on_failure" placeholder="请选择失败策略">
          <a-select-option value="abort">中止执行</a-select-option>
          <a-select-option value="skip">跳过继续</a-select-option>
          <a-select-option value="skip_but_report">跳过但报告</a-select-option>
        </a-select>
      </a-form-item>

      <a-form-item label="依赖步骤" name="dependencies">
        <a-select
          v-model:value="form.dependencies"
          mode="multiple"
          placeholder="请选择依赖步骤（可多选）"
          :options="dependencyOptions"
          :max-tag-count="3"
          show-search
          option-filter-prop="label"
        />
        <div v-if="form.dependencies.length > 0" class="dependency-info">
          <a-alert
            message="依赖关系说明"
            :description="dependencyDescription"
            type="info"
            show-icon
            style="margin-top: 8px"
          />
        </div>
      </a-form-item>

      <a-form-item label="执行顺序" name="order">
        <a-input-number 
          v-model:value="form.order" 
          :min="1" 
          :max="100" 
          style="width: 100%"
        />
      </a-form-item>
    </a-form>

    <div v-if="selectedStep" class="step-info">
      <a-divider>步骤信息</a-divider>
      <a-descriptions :column="1" size="small">
        <a-descriptions-item label="步骤名称">{{ selectedStep.name }}</a-descriptions-item>
        <a-descriptions-item label="描述">{{ selectedStep.description }}</a-descriptions-item>
        <a-descriptions-item label="执行器类型">
          <a-tag :color="getExecutorColor(selectedStep.executor_type)">
            {{ getExecutorName(selectedStep.executor_type) }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="创建者">{{ selectedStep.created_by }}</a-descriptions-item>
      </a-descriptions>
    </div>
  </a-modal>
</template>

<script>
import { ref, reactive, computed, watch } from 'vue'
import { message } from 'ant-design-vue'

export default {
  name: 'TemplateStepDialog',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    mode: {
      type: String,
      default: 'create' // create | edit
    },
    stepData: {
      type: Object,
      default: () => ({})
    },
    availableSteps: {
      type: Array,
      default: () => []
    },
    existingSteps: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:visible', 'success'],
  setup(props, { emit }) {
    const formRef = ref()
    const loading = ref(false)

    const form = reactive({
      step_id: '',
      step_name: '',
      run_mode: 'serial',
      on_failure: 'abort',
      dependencies: [],
      order: 1
    })

    const rules = {
      step_id: [
        { required: true, message: '请选择步骤', trigger: 'change' }
      ],
      run_mode: [
        { required: true, message: '请选择执行模式', trigger: 'change' }
      ],
      on_failure: [
        { required: true, message: '请选择失败策略', trigger: 'change' }
      ],
      order: [
        { required: true, message: '请输入执行顺序', trigger: 'blur' }
      ]
    }

    const dialogTitle = computed(() => {
      return props.mode === 'create' ? '添加步骤' : '编辑步骤'
    })

    // 选中的步骤信息
    const selectedStep = computed(() => {
      return props.availableSteps.find(step => step.id === form.step_id)
    })

    // 依赖选项（排除当前步骤和已添加但还未保存的步骤）
    const dependencyOptions = computed(() => {
      return props.existingSteps
        .filter(step => step.step_id !== form.step_id)
        .map(step => ({
          label: step.step_name || step.step_id,
          value: step.step_id
        }))
    })

    // 依赖关系说明
    const dependencyDescription = computed(() => {
      if (form.dependencies.length === 0) return ''
      
      const depNames = form.dependencies.map(depId => {
        const step = props.existingSteps.find(s => s.step_id === depId)
        return step?.step_name || depId
      })
      
      const currentStepName = form.step_name || '当前步骤'
      
      if (form.dependencies.length === 1) {
        return `${currentStepName} 将在 "${depNames[0]}" 完成后执行`
      } else {
        return `${currentStepName} 将在 "${depNames.join('", "')}" 全部完成后执行（AND关系）`
      }
    })

    // 监听步骤数据变化
    watch(() => props.stepData, (newData) => {
      if (newData && Object.keys(newData).length > 0) {
        Object.assign(form, {
          step_id: newData.step_id || '',
          step_name: newData.step_name || '',
          run_mode: newData.run_mode || 'serial',
          on_failure: newData.on_failure || 'abort',
          dependencies: newData.dependencies || [],
          order: newData.order || 1
        })
      }
    }, { immediate: true })

    // 监听对话框显示状态
    watch(() => props.visible, (visible) => {
      if (visible && props.mode === 'create') {
        resetForm()
      }
    })

    // 重置表单
    const resetForm = () => {
      Object.assign(form, {
        step_id: '',
        step_name: '',
        run_mode: 'serial',
        on_failure: 'abort',
        dependencies: [],
        order: props.existingSteps.length + 1
      })
      formRef.value?.resetFields()
    }

    // 步骤选择变化
    const onStepChange = (stepId) => {
      const step = props.availableSteps.find(s => s.id === stepId)
      if (step) {
        form.step_name = step.name
      }
    }

    // 确定
    const handleOk = async () => {
      try {
        await formRef.value.validate()

        // 检查步骤是否已存在
        if (props.mode === 'create') {
          const exists = props.existingSteps.some(step => step.step_id === form.step_id)
          if (exists) {
            message.error('该步骤已添加，请选择其他步骤')
            return
          }
        }

        const stepData = {
          step_id: form.step_id,
          step_name: form.step_name,
          run_mode: form.run_mode,
          on_failure: form.on_failure,
          dependencies: form.dependencies,
          order: form.order
        }

        emit('success', stepData)
        handleCancel()
      } catch (error) {
        console.error('表单验证失败:', error)
      }
    }

    // 取消
    const handleCancel = () => {
      emit('update:visible', false)
      resetForm()
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

    return {
      formRef,
      loading,
      form,
      rules,
      dialogTitle,
      selectedStep,
      dependencyOptions,
      dependencyDescription,
      handleOk,
      handleCancel,
      onStepChange,
      getExecutorColor,
      getExecutorName
    }
  }
}
</script>

<style scoped>
.step-info {
  margin-top: 16px;
  padding: 16px;
  background-color: #fafafa;
  border-radius: 6px;
}
</style>