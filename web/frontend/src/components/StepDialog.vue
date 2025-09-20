<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="800px"
    :close-on-click-modal="false"
    @update:model-value="$emit('update:visible', $event)"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="120px"
      :disabled="mode === 'view'"
    >
      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="步骤名称" prop="name">
            <el-input v-model="form.name" placeholder="请输入步骤名称" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="执行器类型" prop="executor_type">
            <el-select v-model="form.executor_type" placeholder="请选择执行器类型" @change="onExecutorTypeChange">
              <el-option label="HTTP" value="http" />
              <el-option label="Shell" value="shell" />
              <el-option label="函数" value="func" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item label="描述" prop="description">
        <el-input
          v-model="form.description"
          type="textarea"
          :rows="3"
          placeholder="请输入步骤描述"
        />
      </el-form-item>

      <el-form-item label="创建者邮箱" prop="created_by">
        <el-input v-model="form.created_by" placeholder="请输入创建者邮箱" />
      </el-form-item>

      <!-- HTTP配置 -->
      <template v-if="form.executor_type === 'http'">
        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="HTTP方法" prop="http_method">
              <el-select v-model="form.http_method" placeholder="请选择HTTP方法">
                <el-option label="GET" value="GET" />
                <el-option label="POST" value="POST" />
                <el-option label="PUT" value="PUT" />
                <el-option label="DELETE" value="DELETE" />
                <el-option label="PATCH" value="PATCH" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="16">
            <el-form-item label="HTTP URL" prop="http_url">
              <el-input v-model="form.http_url" placeholder="请输入HTTP URL" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="请求头">
          <el-input
            v-model="form.http_headers"
            type="textarea"
            :rows="3"
            placeholder='请输入JSON格式的请求头，例如: {"Content-Type": "application/json"}'
          />
        </el-form-item>

        <el-form-item label="请求体">
          <el-input
            v-model="form.http_body"
            type="textarea"
            :rows="4"
            placeholder="请输入请求体内容"
          />
        </el-form-item>
      </template>

      <!-- Shell配置 -->
      <template v-else-if="form.executor_type === 'shell'">
        <el-form-item label="Shell脚本" prop="shell_script">
          <el-input
            v-model="form.shell_script"
            type="textarea"
            :rows="6"
            placeholder="请输入Shell脚本"
          />
        </el-form-item>

        <el-form-item label="环境变量">
          <el-input
            v-model="form.shell_env"
            type="textarea"
            :rows="3"
            placeholder='请输入JSON格式的环境变量，例如: {"NODE_ENV": "production"}'
          />
        </el-form-item>
      </template>

      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item label="超时时间(秒)">
            <el-input-number v-model="form.timeout" :min="1" :max="3600" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="异常类型">
            <el-select v-model="form.exception_type" placeholder="请选择异常处理类型">
              <el-option label="中断执行" value="abort" />
              <el-option label="跳过继续" value="skip" />
              <el-option label="重试执行" value="retry" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="$emit('update:visible', false)">取消</el-button>
        <el-button v-if="mode !== 'view'" type="primary" @click="handleSubmit" :loading="submitting">
          {{ mode === 'create' ? '创建' : '更新' }}
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script>
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { createStep, updateStep } from '../api/steps.js'

export default {
  name: 'StepDialog',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    step: {
      type: Object,
      default: null
    },
    mode: {
      type: String,
      default: 'create' // create, edit, view
    }
  },
  emits: ['update:visible', 'success'],
  setup(props, { emit }) {
    const formRef = ref()
    const submitting = ref(false)

    const form = reactive({
      name: '',
      description: '',
      created_by: '',
      executor_type: '',
      exception_type: 'abort',
      timeout: 30,
      // HTTP相关
      http_method: 'GET',
      http_url: '',
      http_headers: '',
      http_body: '',
      // Shell相关
      shell_script: '',
      shell_env: ''
    })

    const rules = {
      name: [
        { required: true, message: '请输入步骤名称', trigger: 'blur' }
      ],
      executor_type: [
        { required: true, message: '请选择执行器类型', trigger: 'change' }
      ],
      created_by: [
        { required: true, message: '请输入创建者邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
      ],
      http_method: [
        { required: true, message: '请选择HTTP方法', trigger: 'change' }
      ],
      http_url: [
        { required: true, message: '请输入HTTP URL', trigger: 'blur' }
      ],
      shell_script: [
        { required: true, message: '请输入Shell脚本', trigger: 'blur' }
      ]
    }

    const dialogTitle = computed(() => {
      const titles = {
        create: '新建步骤',
        edit: '编辑步骤',
        view: '查看步骤'
      }
      return titles[props.mode] || '步骤'
    })

    // 监听步骤数据变化
    watch(() => props.step, (newStep) => {
      if (newStep) {
        Object.assign(form, {
          ...newStep,
          timeout: newStep.timeout || 30
        })
      } else {
        resetForm()
      }
    }, { immediate: true })

    // 监听对话框可见性
    watch(() => props.visible, (visible) => {
      if (!visible) {
        resetForm()
      }
    })

    // 重置表单
    const resetForm = () => {
      Object.assign(form, {
        name: '',
        description: '',
        created_by: '',
        executor_type: '',
        exception_type: 'abort',
        timeout: 30,
        http_method: 'GET',
        http_url: '',
        http_headers: '',
        http_body: '',
        shell_script: '',
        shell_env: ''
      })
      formRef.value?.clearValidate()
    }

    // 执行器类型改变
    const onExecutorTypeChange = () => {
      // 清除其他类型的配置
      if (form.executor_type !== 'http') {
        form.http_method = 'GET'
        form.http_url = ''
        form.http_headers = ''
        form.http_body = ''
      }
      if (form.executor_type !== 'shell') {
        form.shell_script = ''
        form.shell_env = ''
      }
    }

    // 提交表单
    const handleSubmit = async () => {
      try {
        await formRef.value.validate()
        
        // 验证JSON格式
        if (form.http_headers && !isValidJSON(form.http_headers)) {
          ElMessage.error('请求头格式不正确，请输入有效的JSON')
          return
        }
        if (form.shell_env && !isValidJSON(form.shell_env)) {
          ElMessage.error('环境变量格式不正确，请输入有效的JSON')
          return
        }

        submitting.value = true
        
        if (props.mode === 'create') {
          await createStep(form)
          ElMessage.success('创建成功')
        } else {
          await updateStep(props.step.id, form)
          ElMessage.success('更新成功')
        }
        
        emit('success')
      } catch (error) {
        ElMessage.error(error.message)
      } finally {
        submitting.value = false
      }
    }

    // 验证JSON格式
    const isValidJSON = (str) => {
      try {
        JSON.parse(str)
        return true
      } catch {
        return false
      }
    }

    return {
      formRef,
      form,
      rules,
      submitting,
      dialogTitle,
      onExecutorTypeChange,
      handleSubmit
    }
  }
}
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>