package dag

import (
	"sync"
	"sync/atomic"
)

// ID 类型别名，方便以后换 uint64 / uuid
type ID = string

// State 节点生命周期
type State uint8

const (
	StatePending State = iota
	StateRunning
	StateSuccess
	StateFailed
	StateSkipped
)

// Node 是运行时节点，包含业务句柄
type Node struct {
	ID           ID
	Mode         RunMode       // serial / parallel / mixed
	Runner       func() error  // 业务函数
	State        atomic.Uint32 // 无锁读状态
	Successors   map[ID]*Node  // 出边
	Predecessors map[ID]*Node  // 入边（反向边，加速删节点）
	Data         any           // 任意业务数据
	mu           sync.Mutex    // 保护 Successors / Predecessors
}

// RunMode 运行模式
type RunMode uint8

const (
	RunModeSerial RunMode = iota
	RunModeParallel
	RunModeMixed
)

// Desc 是外部（JSON/DB）描述的轻量级 DTO，不包含运行时状态
type Desc struct {
	ID           ID           // 节点唯一标识
	Mode         RunMode      // 运行模式：serial / parallel / mixed
	Dependencies []ID         // 依赖节点 ID 列表
	Runner       func() error // 业务函数，由调用方注入
}
