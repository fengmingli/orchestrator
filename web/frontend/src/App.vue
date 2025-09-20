<template>
  <div id="app">
    <a-layout class="container">
      <a-layout-header class="header">
        <h1>工作流编排系统</h1>
      </a-layout-header>
      
      <a-layout>
        <a-layout-sider class="sidebar" width="200px">
          <a-menu
            mode="inline"
            :selected-keys="[$route.path]"
            class="nav-menu"
            theme="dark"
          >
            <a-menu-item key="/steps" @click="$router.push('/steps')">
              <span>步骤管理</span>
            </a-menu-item>
            <a-menu-item key="/templates" @click="$router.push('/templates')">
              <span>模板管理</span>
            </a-menu-item>
            <a-menu-item key="/executions" @click="$router.push('/executions')">
              <span>执行记录</span>
            </a-menu-item>
          </a-menu>
        </a-layout-sider>
        
        <a-layout-content class="main">
          <router-view />
        </a-layout-content>
      </a-layout>
    </a-layout>
  </div>
</template>

<script>
export default {
  name: 'App'
}
</script>

<style>
#app {
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', SimSun, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

.container {
  height: 100vh;
}

.header {
  background-color: #1890ff;
  color: white;
  display: flex;
  align-items: center;
  padding: 0 20px;
  height: 64px;
}

.header h1 {
  margin: 0;
  font-size: 24px;
  color: white;
}

.sidebar {
  min-height: calc(100vh - 64px);
}

.nav-menu {
  height: 100%;
  border-right: 0;
}

.main {
  padding: 24px;
  background-color: #f0f2f5;
  min-height: calc(100vh - 64px);
  overflow-y: auto;
}
</style>