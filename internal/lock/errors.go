package lock

import "errors"

var (
	// ErrLockAlreadyExists 锁已存在错误
	ErrLockAlreadyExists = errors.New("lock already exists")
	
	// ErrLockNotFound 锁不存在错误
	ErrLockNotFound = errors.New("lock not found")
	
	// ErrLockTimeout 获取锁超时错误
	ErrLockTimeout = errors.New("lock timeout")
	
	// ErrLockExpired 锁已过期错误
	ErrLockExpired = errors.New("lock expired")
	
	// ErrWorkflowAlreadyRunning 工作流已在运行错误
	ErrWorkflowAlreadyRunning = errors.New("workflow is already running on another instance")
)