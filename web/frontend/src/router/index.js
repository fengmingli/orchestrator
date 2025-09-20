import { createRouter, createWebHistory } from 'vue-router'
import StepList from '../views/StepList.vue'
import TemplateList from '../views/TemplateList.vue'
import ExecutionList from '../views/ExecutionList.vue'

const routes = [
  {
    path: '/',
    redirect: '/steps'
  },
  {
    path: '/steps',
    name: 'Steps',
    component: StepList
  },
  {
    path: '/templates',
    name: 'Templates',
    component: TemplateList
  },
  {
    path: '/executions',
    name: 'Executions',
    component: ExecutionList
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router