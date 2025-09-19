package task

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ShellTask Shell脚本任务实现
type ShellTask struct {
	BaseTask
	Script      string            `json:"script"`
	WorkingDir  string            `json:"working_dir"`
	Environment map[string]string `json:"environment"`
}

// NewShellTask 创建新的Shell任务
func NewShellTask(id, name, script string) *ShellTask {
	return &ShellTask{
		BaseTask: BaseTask{
			ID:      id,
			Name:    name,
			Type:    TaskTypeShell,
			Timeout: 60 * time.Second, // Shell任务默认60秒超时
		},
		Script:      script,
		Environment: make(map[string]string),
	}
}

// WithWorkingDir 设置工作目录
func (s *ShellTask) WithWorkingDir(dir string) *ShellTask {
	s.WorkingDir = dir
	return s
}

// WithEnvironment 设置环境变量
func (s *ShellTask) WithEnvironment(env map[string]string) *ShellTask {
	s.Environment = env
	return s
}

// AddEnvironment 添加单个环境变量
func (s *ShellTask) AddEnvironment(key, value string) *ShellTask {
	if s.Environment == nil {
		s.Environment = make(map[string]string)
	}
	s.Environment[key] = value
	return s
}

// Validate 验证Shell任务参数
func (s *ShellTask) Validate() error {
	if err := s.BaseTask.Validate(); err != nil {
		return err
	}

	if strings.TrimSpace(s.Script) == "" {
		return fmt.Errorf("script cannot be empty")
	}

	// 检查工作目录是否存在
	if s.WorkingDir != "" {
		if info, err := os.Stat(s.WorkingDir); err != nil {
			return fmt.Errorf("working directory does not exist: %s", s.WorkingDir)
		} else if !info.IsDir() {
			return fmt.Errorf("working directory is not a directory: %s", s.WorkingDir)
		}
	}

	return nil
}

// Execute 执行Shell任务
func (s *ShellTask) Execute(ctx context.Context) (ExecResult, error) {
	start := time.Now()
	result := ExecResult{
		TaskID:    s.ID,
		StartTime: start,
		Metadata:  make(map[string]interface{}),
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, "sh", "-c", s.Script)

	// 设置工作目录
	if s.WorkingDir != "" {
		cmd.Dir = s.WorkingDir
		result.Metadata["working_dir"] = s.WorkingDir
	}

	// 设置环境变量
	if len(s.Environment) > 0 {
		env := os.Environ()
		for key, value := range s.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
		result.Metadata["environment"] = s.Environment
	}

	// 执行命令并捕获输出
	output, err := cmd.CombinedOutput()

	// 填充结果
	result.FinishTime = time.Now()
	result.Duration = result.FinishTime.Sub(start)
	result.Output = string(output)
	result.Metadata["script"] = s.Script

	if err != nil {
		result.Error = err.Error()
		// 添加退出码信息
		if exitError, ok := err.(*exec.ExitError); ok {
			result.Metadata["exit_code"] = exitError.ExitCode()
		}
		return result, err
	}

	result.Metadata["exit_code"] = 0
	return result, nil
}

// ExecuteWithOutput 执行Shell任务并分别返回stdout和stderr
func (s *ShellTask) ExecuteWithOutput(ctx context.Context) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", s.Script)

	if s.WorkingDir != "" {
		cmd.Dir = s.WorkingDir
	}

	if len(s.Environment) > 0 {
		env := os.Environ()
		for key, value := range s.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	return stdout, stderr, err
}
