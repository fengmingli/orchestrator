<template>
  <a-modal
    title="模板DAG可视化"
    :open="visible"
    :width="1200"
    :footer="null"
    @cancel="handleCancel"
  >
    <div class="dag-container">
      <div class="dag-header">
        <h3>{{ templateName }}</h3>
        <a-space>
          <a-button @click="zoomIn" :disabled="zoomLevel >= maxZoom">
            <template #icon><PlusOutlined /></template>
            放大
          </a-button>
          <a-button @click="zoomOut" :disabled="zoomLevel <= minZoom">
            <template #icon><MinusOutlined /></template>
            缩小
          </a-button>
          <a-button @click="fitView">
            <template #icon><BorderOutlined /></template>
            适应视图
          </a-button>
          <a-button @click="resetView">
            <template #icon><ReloadOutlined /></template>
            重置
          </a-button>
          <span class="zoom-info">{{ Math.round(zoomLevel * 100) }}%</span>
        </a-space>
      </div>
      
      <div class="dag-content" ref="dagContainer" @wheel="handleWheel">
        <div v-if="loading" class="loading-container">
          <a-spin size="large" />
          <p>加载DAG数据中...</p>
        </div>
        
        <div v-else-if="error" class="error-container">
          <a-result status="error" :title="error" />
        </div>
        
        <div v-else-if="dagData" class="dag-viewer" ref="dagViewer">
          <svg 
            :width="svgWidth" 
            :height="svgHeight" 
            ref="svgElement"
            :style="{ transform: `scale(${zoomLevel}) translate(${panX}px, ${panY}px)` }"
            @mousedown="handleMouseDown"
          >
            <!-- 箭头标记定义 -->
            <defs>
              <marker id="arrowhead" markerWidth="12" markerHeight="8" 
                      refX="11" refY="4" orient="auto" markerUnits="strokeWidth">
                <polygon points="0 0, 12 4, 0 8" fill="#1890ff" />
              </marker>
              <marker id="arrowhead-green" markerWidth="12" markerHeight="8" 
                      refX="11" refY="4" orient="auto" markerUnits="strokeWidth">
                <polygon points="0 0, 12 4, 0 8" fill="#52c41a" />
              </marker>
              <marker id="arrowhead-orange" markerWidth="12" markerHeight="8" 
                      refX="11" refY="4" orient="auto" markerUnits="strokeWidth">
                <polygon points="0 0, 12 4, 0 8" fill="#fa8c16" />
              </marker>
            </defs>
            
            <!-- 连接线 -->
            <g class="edges">
              <path
                v-for="edge in dagData.edges"
                :key="edge.id"
                :d="getEdgePath(edge)"
                :stroke="getEdgeColor(edge)"
                stroke-width="3"
                fill="none"
                :marker-end="getArrowMarker(edge)"
                class="edge-path"
              />
            </g>
            
            <!-- 节点 -->
            <g class="nodes">
              <g v-for="node in dagData.nodes" :key="node.id">
                <rect
                  :x="getNodePosition(node.id).x"
                  :y="getNodePosition(node.id).y"
                  width="140"
                  height="80"
                  :rx="8"
                  :ry="8"
                  :fill="getNodeFill(node)"
                  :stroke="getNodeStroke(node)"
                  stroke-width="2"
                  class="node-rect"
                />
                <foreignObject
                  :x="getNodePosition(node.id).x + 5"
                  :y="getNodePosition(node.id).y + 5"
                  width="130"
                  height="70"
                >
                  <div class="dag-node" :class="getNodeClass(node)">
                    <div class="node-header">
                      <span class="node-type">{{ getTypeLabel(node.type) }}</span>
                    </div>
                    <div class="node-title" :title="node.name">{{ node.name }}</div>
                    <div class="node-mode">{{ getModeLabel(node.run_mode) }}</div>
                    <div v-if="node.dependencies && node.dependencies.length > 0" class="node-deps">
                      依赖: {{ node.dependencies.length }}个
                    </div>
                  </div>
                </foreignObject>
              </g>
            </g>
          </svg>
        </div>
        
        <div v-else class="empty-container">
          <a-empty description="暂无DAG数据" />
        </div>
      </div>
      
      <!-- 图例和统计信息 -->
      <div class="dag-footer">
        <div class="dag-legend">
          <h4>图例</h4>
          <div class="legend-items">
            <div class="legend-item">
              <div class="legend-node shell"></div>
              <span>Shell步骤</span>
            </div>
            <div class="legend-item">
              <div class="legend-node http"></div>
              <span>HTTP步骤</span>
            </div>
            <div class="legend-item">
              <div class="legend-node func"></div>
              <span>Function步骤</span>
            </div>
          </div>
        </div>
        <div v-if="dagData" class="dag-stats">
          <a-statistic title="总步骤数" :value="dagData.nodes?.length || 0" :value-style="{ fontSize: '16px' }" />
          <a-statistic title="依赖关系" :value="dagData.edges?.length || 0" :value-style="{ fontSize: '16px' }" />
          <a-statistic title="最大层级" :value="maxLevel + 1" :value-style="{ fontSize: '16px' }" />
        </div>
      </div>
    </div>
  </a-modal>
</template>

<script>
import { ref, reactive, watch, nextTick, computed } from 'vue'
import { message } from 'ant-design-vue'
import { 
  PlusOutlined, 
  MinusOutlined, 
  BorderOutlined, 
  ReloadOutlined 
} from '@ant-design/icons-vue'
import { getTemplateDAG } from '../api/templates.js'

export default {
  name: 'DAGViewer',
  components: {
    PlusOutlined,
    MinusOutlined,
    BorderOutlined,
    ReloadOutlined
  },
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    templateId: {
      type: String,
      default: ''
    },
    templateName: {
      type: String,
      default: ''
    }
  },
  emits: ['update:visible'],
  setup(props, { emit }) {
    const loading = ref(false)
    const error = ref('')
    const dagData = ref(null)
    const dagContainer = ref()
    const dagViewer = ref()
    const svgElement = ref()

    // 缩放和平移状态
    const zoomLevel = ref(1)
    const minZoom = 0.2
    const maxZoom = 3
    const panX = ref(0)
    const panY = ref(0)
    const isDragging = ref(false)
    const lastPanPoint = ref({ x: 0, y: 0 })

    // 节点位置缓存
    const nodePositions = reactive({})
    const maxLevel = ref(0)

    // SVG尺寸
    const svgWidth = computed(() => {
      const levels = Object.keys(nodePositions).length > 0 ? 
        Math.max(...Object.values(nodePositions).map(pos => Math.floor(pos.x / 200))) + 1 : 1
      return Math.max(1000, levels * 200 + 100)
    })

    const svgHeight = computed(() => {
      return Math.max(500, (maxLevel.value + 1) * 150 + 100)
    })

    // 监听模板ID变化，加载DAG数据
    watch(() => [props.visible, props.templateId], ([visible, templateId]) => {
      if (visible && templateId) {
        resetView()
        loadDAGData()
      }
    })

    // 加载DAG数据
    const loadDAGData = async () => {
      if (!props.templateId) return

      loading.value = true
      error.value = ''
      try {
        const response = await getTemplateDAG(props.templateId)
        dagData.value = response.data
        
        // 计算节点位置
        await nextTick()
        calculateNodePositions()
      } catch (err) {
        error.value = '加载DAG数据失败: ' + err.message
        message.error(error.value)
      } finally {
        loading.value = false
      }
    }

    // 计算节点位置（水平布局）
    const calculateNodePositions = () => {
      if (!dagData.value || !dagData.value.nodes) return

      const nodes = dagData.value.nodes
      const edges = dagData.value.edges || []
      
      // 计算每个节点的层级
      const levels = {}
      const inDegree = {}
      
      // 初始化入度
      nodes.forEach(node => {
        inDegree[node.id] = 0
      })
      
      // 计算入度
      edges.forEach(edge => {
        inDegree[edge.target] = (inDegree[edge.target] || 0) + 1
      })
      
      // 使用拓扑排序计算层级
      const queue = []
      nodes.forEach(node => {
        if (inDegree[node.id] === 0) {
          levels[node.id] = 0
          queue.push(node.id)
        }
      })
      
      while (queue.length > 0) {
        const current = queue.shift()
        const currentLevel = levels[current]
        
        edges.forEach(edge => {
          if (edge.source === current) {
            inDegree[edge.target]--
            if (inDegree[edge.target] === 0) {
              levels[edge.target] = currentLevel + 1
              queue.push(edge.target)
            }
          }
        })
      }
      
      // 计算最大层级
      maxLevel.value = Math.max(...Object.values(levels))
      
      // 按层级分组
      const levelGroups = {}
      Object.keys(levels).forEach(nodeId => {
        const level = levels[nodeId]
        if (!levelGroups[level]) {
          levelGroups[level] = []
        }
        levelGroups[level].push(nodeId)
      })
      
      // 水平布局：从左到右排列
      const levelWidth = 200  // 层级间距
      const nodeHeight = 120  // 节点间距
      const startX = 80
      const startY = 60
      
      Object.keys(levelGroups).forEach(level => {
        const nodesInLevel = levelGroups[level]
        const levelX = startX + parseInt(level) * levelWidth
        
        // 计算该层级的总高度，居中排列
        const totalHeight = nodesInLevel.length * nodeHeight
        const levelStartY = startY + Math.max(0, (svgHeight.value - totalHeight) / 2)
        
        nodesInLevel.forEach((nodeId, index) => {
          const nodeY = levelStartY + index * nodeHeight
          nodePositions[nodeId] = {
            x: levelX,
            y: nodeY
          }
        })
      })
    }

    // 获取节点位置
    const getNodePosition = (nodeId) => {
      return nodePositions[nodeId] || { x: 50, y: 50 }
    }

    // 获取边路径（贝塞尔曲线）
    const getEdgePath = (edge) => {
      const sourcePos = getNodePosition(edge.source)
      const targetPos = getNodePosition(edge.target)
      
      const x1 = sourcePos.x + 140  // 从节点右侧出发
      const y1 = sourcePos.y + 40   // 节点中心高度
      const x2 = targetPos.x        // 到节点左侧
      const y2 = targetPos.y + 40   // 节点中心高度
      
      // 计算控制点，创建水平方向的贝塞尔曲线
      const controlOffset = Math.min(100, Math.abs(x2 - x1) / 2)
      const cx1 = x1 + controlOffset
      const cy1 = y1
      const cx2 = x2 - controlOffset
      const cy2 = y2
      
      return `M ${x1} ${y1} C ${cx1} ${cy1}, ${cx2} ${cy2}, ${x2} ${y2}`
    }

    // 获取边颜色
    const getEdgeColor = (edge) => {
      return '#1890ff'
    }

    // 获取箭头标记
    const getArrowMarker = (edge) => {
      return 'url(#arrowhead)'
    }

    // 获取节点填充色
    const getNodeFill = (node) => {
      const fills = {
        shell: '#f6ffed',
        http: '#e6f7ff',
        func: '#fff7e6'
      }
      return fills[node.type] || '#fafafa'
    }

    // 获取节点边框色
    const getNodeStroke = (node) => {
      const strokes = {
        shell: '#52c41a',
        http: '#1890ff',
        func: '#fa8c16'
      }
      return strokes[node.type] || '#d9d9d9'
    }

    // 获取节点样式类
    const getNodeClass = (node) => {
      return [`node-${node.type}`, `mode-${node.run_mode}`]
    }

    // 获取类型标签
    const getTypeLabel = (type) => {
      const labels = {
        shell: 'Shell',
        http: 'HTTP',
        func: 'Function'
      }
      return labels[type] || type
    }

    // 获取模式标签
    const getModeLabel = (mode) => {
      const labels = {
        serial: '串行',
        parallel: '并行'
      }
      return labels[mode] || mode
    }

    // 缩放功能
    const zoomIn = () => {
      if (zoomLevel.value < maxZoom) {
        zoomLevel.value = Math.min(maxZoom, zoomLevel.value * 1.2)
      }
    }

    const zoomOut = () => {
      if (zoomLevel.value > minZoom) {
        zoomLevel.value = Math.max(minZoom, zoomLevel.value / 1.2)
      }
    }

    // 鼠标滚轮缩放
    const handleWheel = (event) => {
      event.preventDefault()
      const delta = event.deltaY > 0 ? -1 : 1
      const zoomFactor = 1 + (delta * 0.1)
      const newZoom = zoomLevel.value * zoomFactor
      
      if (newZoom >= minZoom && newZoom <= maxZoom) {
        zoomLevel.value = newZoom
      }
    }

    // 鼠标拖拽
    const handleMouseDown = (event) => {
      isDragging.value = true
      lastPanPoint.value = { x: event.clientX, y: event.clientY }
      
      const handleMouseMove = (e) => {
        if (isDragging.value) {
          const deltaX = e.clientX - lastPanPoint.value.x
          const deltaY = e.clientY - lastPanPoint.value.y
          
          panX.value += deltaX / zoomLevel.value
          panY.value += deltaY / zoomLevel.value
          
          lastPanPoint.value = { x: e.clientX, y: e.clientY }
        }
      }
      
      const handleMouseUp = () => {
        isDragging.value = false
        document.removeEventListener('mousemove', handleMouseMove)
        document.removeEventListener('mouseup', handleMouseUp)
      }
      
      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)
    }

    // 适应视图
    const fitView = () => {
      if (!dagData.value || !dagData.value.nodes.length) return
      
      // 计算所有节点的边界
      const positions = Object.values(nodePositions)
      if (positions.length === 0) return
      
      const minX = Math.min(...positions.map(p => p.x))
      const maxX = Math.max(...positions.map(p => p.x)) + 140
      const minY = Math.min(...positions.map(p => p.y))
      const maxY = Math.max(...positions.map(p => p.y)) + 80
      
      const contentWidth = maxX - minX
      const contentHeight = maxY - minY
      
      // 计算适合的缩放比例
      const containerWidth = dagContainer.value?.clientWidth || 1000
      const containerHeight = dagContainer.value?.clientHeight || 500
      
      const scaleX = (containerWidth - 100) / contentWidth
      const scaleY = (containerHeight - 100) / contentHeight
      const scale = Math.min(scaleX, scaleY, 1)
      
      zoomLevel.value = Math.max(minZoom, Math.min(maxZoom, scale))
      
      // 居中显示
      panX.value = (containerWidth / 2 - (minX + contentWidth / 2) * zoomLevel.value) / zoomLevel.value
      panY.value = (containerHeight / 2 - (minY + contentHeight / 2) * zoomLevel.value) / zoomLevel.value
    }

    // 重置视图
    const resetView = () => {
      zoomLevel.value = 1
      panX.value = 0
      panY.value = 0
    }

    // 关闭对话框
    const handleCancel = () => {
      emit('update:visible', false)
      dagData.value = null
      error.value = ''
      resetView()
    }

    return {
      loading,
      error,
      dagData,
      dagContainer,
      dagViewer,
      svgElement,
      zoomLevel,
      minZoom,
      maxZoom,
      panX,
      panY,
      svgWidth,
      svgHeight,
      maxLevel,
      getNodePosition,
      getEdgePath,
      getEdgeColor,
      getArrowMarker,
      getNodeFill,
      getNodeStroke,
      getNodeClass,
      getTypeLabel,
      getModeLabel,
      zoomIn,
      zoomOut,
      handleWheel,
      handleMouseDown,
      fitView,
      resetView,
      handleCancel
    }
  }
}
</script>

<style scoped>
.dag-container {
  height: 700px;
  display: flex;
  flex-direction: column;
}

.dag-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 0;
  border-bottom: 1px solid #f0f0f0;
}

.dag-header h3 {
  margin: 0;
  color: #262626;
  font-size: 18px;
}

.zoom-info {
  font-size: 14px;
  color: #666;
  min-width: 40px;
}

.dag-content {
  flex: 1;
  position: relative;
  overflow: hidden;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  margin: 16px 0;
  cursor: grab;
}

.dag-content:active {
  cursor: grabbing;
}

.loading-container,
.error-container,
.empty-container {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  height: 100%;
  min-height: 400px;
}

.dag-viewer {
  width: 100%;
  height: 100%;
  min-height: 500px;
  position: relative;
}

.dag-viewer svg {
  transform-origin: 0 0;
  transition: transform 0.3s ease;
}

.edge-path {
  cursor: pointer;
  transition: stroke-width 0.2s ease;
}

.edge-path:hover {
  stroke-width: 4;
}

.node-rect {
  cursor: pointer;
  transition: all 0.3s ease;
  filter: drop-shadow(0 2px 4px rgba(0,0,0,0.1));
}

.node-rect:hover {
  filter: drop-shadow(0 4px 8px rgba(0,0,0,0.15));
  transform: translateY(-1px);
}

.dag-node {
  width: 130px;
  height: 70px;
  background: transparent;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  font-size: 11px;
  pointer-events: none;
  padding: 4px;
}

.node-header {
  font-weight: 600;
  color: #262626;
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.node-title {
  font-weight: 700;
  color: #262626;
  text-align: center;
  margin: 2px 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 120px;
  font-size: 12px;
  line-height: 1.2;
}

.node-mode {
  font-size: 9px;
  color: #8c8c8c;
  font-weight: 500;
}

.node-deps {
  font-size: 8px;
  color: #fa8c16;
  font-weight: 500;
  margin-top: 1px;
}

.dag-footer {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 16px 0;
  border-top: 1px solid #f0f0f0;
  gap: 32px;
}

.dag-legend {
  flex: 1;
}

.dag-legend h4 {
  margin: 0 0 12px 0;
  color: #262626;
  font-size: 14px;
}

.legend-items {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.legend-node {
  width: 16px;
  height: 16px;
  border: 2px solid;
  border-radius: 3px;
  flex-shrink: 0;
}

.legend-node.shell {
  border-color: #52c41a;
  background: #f6ffed;
}

.legend-node.http {
  border-color: #1890ff;
  background: #e6f7ff;
}

.legend-node.func {
  border-color: #fa8c16;
  background: #fff7e6;
}

.dag-stats {
  display: flex;
  gap: 24px;
  align-items: center;
}

.dag-stats :deep(.ant-statistic) {
  text-align: center;
  min-width: 80px;
}

.dag-stats :deep(.ant-statistic-title) {
  font-size: 12px;
  color: #8c8c8c;
}

.dag-stats :deep(.ant-statistic-content) {
  color: #1890ff;
  font-weight: 600;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .dag-footer {
    flex-direction: column;
    gap: 16px;
  }
  
  .dag-stats {
    justify-content: center;
  }
  
  .legend-items {
    justify-content: center;
  }
}
</style>