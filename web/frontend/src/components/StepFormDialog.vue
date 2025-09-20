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
      <a-form-item label="步骤名称" name="name">
        <a-input v-model:value="form.name" placeholder="请输入步骤名称" />
      </a-form-item>
      
      <a-form-item label="步骤描述" name="description">
        <a-textarea 
          v-model:value="form.description" 
          placeholder="请输入步骤描述" 
          :rows="3"
        />
      </a-form-item>
      
      <a-form-item label="创建者邮箱" name="created_by">
        <a-input 
          v-model:value="form.created_by" 
          placeholder="请输入创建者邮箱"
          :disabled="mode === 'edit'"
        />
      </a-form-item>
      
      <a-form-item label="执行器类型" name="executor_type">
        <a-select 
          v-model:value="form.executor_type" 
          placeholder="请选择执行器类型"
          @change="onExecutorTypeChange"
          :disabled="mode === 'edit'"
        >
          <a-select-option value="shell">Shell</a-select-option>
          <a-select-option value="http">HTTP</a-select-option>
          <a-select-option value="func">Function</a-select-option>
        </a-select>
      </a-form-item>

      <!-- HTTP配置 -->
      <template v-if="form.executor_type === 'http'">
        <a-divider>HTTP配置</a-divider>
        
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="HTTP方法" name="http_method">
              <a-select v-model:value="form.http_method" placeholder="请选择HTTP方法">
                <a-select-option value="GET">GET</a-select-option>
                <a-select-option value="POST">POST</a-select-option>
                <a-select-option value="PUT">PUT</a-select-option>
                <a-select-option value="DELETE">DELETE</a-select-option>
                <a-select-option value="PATCH">PATCH</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="16">
            <a-form-item label="HTTP URL" name="http_url">
              <a-input v-model:value="form.http_url" placeholder="请输入HTTP URL" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="请求头">
          <a-textarea
            v-model:value="form.http_headers"
            :rows="3"
            placeholder='请输入JSON格式的请求头，例如: {"Content-Type": "application/json"}'
          />
        </a-form-item>

        <a-form-item label="请求体">
          <a-textarea
            v-model:value="form.http_body"
            :rows="4"
            placeholder="请输入请求体内容"
          />
        </a-form-item>
      </template>

      <!-- Shell配置 -->
      <template v-else-if="form.executor_type === 'shell'">
        <a-divider>Shell配置</a-divider>
        
        <a-form-item label="Shell脚本" name="shell_script">
          <a-textarea
            v-model:value="form.shell_script"
            :rows="6"
            placeholder="请输入Shell脚本"
          />
        </a-form-item>

        <a-form-item label="环境变量">
          <a-textarea
            v-model:value="form.shell_env"
            :rows="3"
            placeholder='请输入JSON格式的环境变量，例如: {"NODE_ENV": "production"}'
          />
        </a-form-item>
      </template>

      <!-- Function配置 -->
      <template v-else-if="form.executor_type === 'func'">
        <a-divider>Function配置</a-divider>
        
        <a-form-item label="函数名称" name="func_name">
          <a-input v-model:value="form.func_name" placeholder="请输入函数名称" />
        </a-form-item>

        <a-form-item label="函数参数">
          <a-textarea
            v-model:value="form.func_params"
            :rows="4"
            placeholder='请输入JSON格式的函数参数，例如: {"param1": "value1"}'
          />
        </a-form-item>
      </template>

      <a-divider>高级配置</a-divider>

      <a-form-item label="超时时间(秒)" name="timeout">
        <a-input-number 
          v-model:value="form.timeout" 
          :min="1" 
          :max="3600" 
          style="width: 100%"
        />
      </a-form-item>

      <a-form-item label="重试次数" name="retry_count">
        <a-input-number 
          v-model:value="form.retry_count" 
          :min="0" 
          :max="10" 
          style="width: 100%"
        />
      </a-form-item>

      <a-form-item label="失败策略" name="exception_type">
        <a-select v-model:value="form.exception_type" placeholder="请选择失败策略">
          <a-select-option value="abort">中断执行</a-select-option>
          <a-select-option value="skip">跳过继续</a-select-option>
          <a-select-option value="retry">重试执行</a-select-option>
        </a-select>
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script>
import { ref, reactive, computed, watch } from 'vue'
import { message } from 'ant-design-vue'
import { createStep, updateStep } from '../api/steps.js'

export default {
  name: 'StepFormDialog',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    mode: {
      type: String,
      default: 'create' // create | edit | view
    },
    stepData: {
      type: Object,
      default: () => ({})
    }
  },
  emits: ['update:visible', 'success'],
  setup(props, { emit }) {
    const formRef = ref()
    const loading = ref(false)

    const form = reactive({
      name: '',
      description: '',
      created_by: '',
      executor_type: '',
      timeout: 30,
      retry_count: 0,
      exception_type: 'abort',
      // HTTP相关
      http_method: 'GET',
      http_url: '',
      http_headers: '',
      http_body: '',
      // Shell相关
      shell_script: '',
      shell_env: '',
      // Function相关
      func_name: '',
      func_params: ''
    })

    const rules = {
      name: [
        { required: true, message: '请输入步骤名称', trigger: 'blur' },
        { min: 1, max: 128, message: '步骤名称长度在1-128个字符', trigger: 'blur' },
        { pattern: /^[a-zA-Z0-9_-]+$/, message: '步骤名称只能包含字母、数字、下划线、横线', trigger: 'blur' }
      ],
      description: [
        { required: true, message: '请输入步骤描述', trigger: 'blur' }
      ],
      created_by: [
        { required: true, message: '请输入创建者邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
      ],
      executor_type: [
        { required: true, message: '请选择执行器类型', trigger: 'change' }
      ],
      http_method: [
        { required: true, message: '请选择HTTP方法', trigger: 'change' }
      ],
      http_url: [
        { required: true, message: '请输入HTTP URL', trigger: 'blur' },
        { type: 'url', message: '请输入正确的URL格式', trigger: 'blur' }
      ],
      shell_script: [
        { required: true, message: '请输入Shell脚本', trigger: 'blur' }
      ],
      func_name: [
        { required: true, message: '请输入函数名称', trigger: 'blur' }
      ]
    }

    const dialogTitle = computed(() => {
      const titles = {
        create: '创建步骤',
        edit: '编辑步骤',
        view: '查看步骤'
      }
      return titles[props.mode] || '步骤'
    })

    // 监听步骤数据变化
    watch(() => props.stepData, (newData) => {
      if (newData && Object.keys(newData).length > 0) {
        // 解析配置数据
        let config = {}
        try {
          config = newData.config ? JSON.parse(newData.config) : {}
        } catch (e) {
          console.warn('解析步骤配置失败:', e)
        }

        Object.assign(form, {
          name: newData.name || '',
          description: newData.description || '',
          created_by: newData.created_by || '',
          executor_type: newData.executor_type || '',
          timeout: newData.timeout || 30,
          retry_count: newData.retry_count || 0,
          exception_type: newData.exception_type || 'abort',
          // HTTP配置
          http_method: config.method || 'GET',
          http_url: config.url || '',
          http_headers: config.headers ? JSON.stringify(config.headers, null, 2) : '',
          http_body: config.body || '',
          // Shell配置
          shell_script: config.script || '',
          shell_env: config.env ? JSON.stringify(config.env, null, 2) : '',
          // Function配置
          func_name: config.name || '',
          func_params: config.params ? JSON.stringify(config.params, null, 2) : ''
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
        name: '',
        description: '',
        created_by: '',
        executor_type: '',
        timeout: 30,
        retry_count: 0,
        exception_type: 'abort',
        http_method: 'GET',
        http_url: '',
        http_headers: '',
        http_body: '',
        shell_script: '',
        shell_env: '',
        func_name: '',
        func_params: ''
      })
      formRef.value?.resetFields()
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
      if (form.executor_type !== 'func') {
        form.func_name = ''
        form.func_params = ''
      }
    }

    // 验证JSON格式
    const isValidJSON = (str) => {
      if (!str) return true
      try {
        JSON.parse(str)
        return true
      } catch {
        return false
      }
    }

    // 构建配置对象
    const buildConfig = () => {
      const config = {}
      
      if (form.executor_type === 'http') {
        config.method = form.http_method
        config.url = form.http_url
        if (form.http_headers) {
          config.headers = JSON.parse(form.http_headers)
        }
        if (form.http_body) {
          config.body = form.http_body
        }
      } else if (form.executor_type === 'shell') {
        config.script = form.shell_script
        if (form.shell_env) {
          config.env = JSON.parse(form.shell_env)
        }
      } else if (form.executor_type === 'func') {
        config.name = form.func_name
        if (form.func_params) {
          config.params = JSON.parse(form.func_params)
        }
      }
      
      return config
    }

    // 确定
    const handleOk = async () => {
      try {
        await formRef.value.validate()

        // 验证JSON格式
        if (form.http_headers && !isValidJSON(form.http_headers)) {
          message.error('请求头格式不正确，请输入有效的JSON')
          return
        }
        if (form.shell_env && !isValidJSON(form.shell_env)) {
          message.error('环境变量格式不正确，请输入有效的JSON')
          return
        }
        if (form.func_params && !isValidJSON(form.func_params)) {
          message.error('函数参数格式不正确，请输入有效的JSON')
          return
        }

        loading.value = true

        const stepData = {
          name: form.name,
          description: form.description,
          created_by: form.created_by,
          executor_type: form.executor_type,
          timeout: form.timeout,
          retry_count: form.retry_count,
          exception_type: form.exception_type,
          config: JSON.stringify(buildConfig())
        }

        if (props.mode === 'create') {
          await createStep(stepData)
          message.success('步骤创建成功')
        } else if (props.mode === 'edit') {
          await updateStep(props.stepData.id, stepData)
          message.success('步骤更新成功')
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

    return {
      formRef,
      loading,
      form,
      rules,
      dialogTitle,
      handleOk,
      handleCancel,
      onExecutorTypeChange
    }
  }
}
</script>

<style scoped>
</style>