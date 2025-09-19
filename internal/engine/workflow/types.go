package workflow

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ExecutionPolicy struct {
	MaxRetries int           // 最大重试次数（0 = 不重试）
	RetryDelay time.Duration // 重试间隔
	OnFailure  FailureAction // 失败后行为：Retry / Skip / Abort（默认）
}
type FailureAction int

const (
	FailureRetry         FailureAction = iota // 重试
	FailureSkip                               // 跳过，继续执行后续
	FailureAbort                              // 中止整个 DAG（默认）
	FailureSkipButReport                      // 跳过，但 Execute() 返回错误
)

// ID 类型别名，方便以后换 uint64 / uuid
type ID = string

// State 节点生命周期
type State uint8

const (
	StatePending State = iota
	StateRunning
	StateSucceeded
	StateFailed
	StateSkipped
)

// Node 是运行时节点，包含业务句柄
type Node struct {
	ID           ID
	Mode         RunMode         // serial / parallel / mixed
	Runner       func() error    // 业务函数
	State        atomic.Uint32   // 无锁读状态
	Successors   map[ID]*Node    // 出边
	Predecessors map[ID]*Node    // 入边（反向边，加速删节点）
	Data         any             // 任意业务数据
	mu           sync.Mutex      // 保护 Successors / Predecessors
	doneCh       chan struct{}   // 用于通知本节点完成
	once         sync.Once       //确保doneCh只关闭一次
	execCount    int             //已经支持次数
	policy       ExecutionPolicy // 执行策略
}

func (n *Node) MarkDone() {
	n.once.Do(func() {
		close(n.doneCh)
	})
}

// RunMode 运行模式
type RunMode string

const (
	// RunModeSerial 串行执行：按序执行当前 Node 的所有后继节点
	RunModeSerial RunMode = "serial"
	// RunModeParallel 并行执行：同时执行当前 Node 的所有后继节点
	RunModeParallel RunMode = "parallel"
	// RunModeMixed 混合执行：
	// 1. 如果当前 Node 有依赖，则按照依赖关系执行
	// 2. 如果当前 Node 无依赖，则并行执行所有后继节点
	RunModeMixed RunMode = "mixed"
)

func AsRunMode(runMode string) RunMode {
	runModeTmp := strings.TrimSpace(runMode)
	if runModeTmp == "" {
		return RunModeSerial // 默认串行
	}
	if runModeTmp == "serial" {
		return RunModeSerial
	}
	if runModeTmp == "parallel" {
		return RunModeParallel
	}
	if runModeTmp == "mixed" {
		return RunModeMixed
	}
	return RunModeSerial
}

// Desc 是外部（JSON/DB）描述的轻量级 DTO，不包含运行时状态
type Desc struct {
	ID     ID           // 节点唯一标识
	Mode   RunMode      // 运行模式：serial / parallel / mixed
	Deps   []ID         // 依赖节点 ID 列表
	Runner func() error // 业务函数，由调用方注入
	Policy ExecutionPolicy
}

// Runner 是用户业务函数签名
type Runner func() error

// DAGSnapshot 是 DAG 的结构快照，用于版本比对
type DAGSnapshot struct {
	Version   string         `json:"version"` // 可选：版本号或哈希
	Nodes     []NodeSnapshot `json:"nodes"`
	CreatedAt time.Time      `json:"created_at"`
}

// NodeSnapshot 是节点的结构快照
type NodeSnapshot struct {
	ID   ID      `json:"1"`
	Deps []ID    `json:"deps"` // 依赖的节点 ID 列表
	Mode RunMode `json:"mode"` // 运行模式
}

// Diff 表示一个差异点
type Diff struct {
	Type   DiffType `json:"type"` // ADD, REMOVE, MODIFY
	NodeID ID       `json:"node_id"`
	Field  string   `json:"field"` // "deps", "mode", etc.
	OldVal string   `json:"old_val"`
	NewVal string   `json:"new_val"`
}
type DiffType string

const (
	DiffTypeAdd    DiffType = "ADD"
	DiffTypeRemove DiffType = "REMOVE"
	DiffTypeModify DiffType = "MODIFY"
)
