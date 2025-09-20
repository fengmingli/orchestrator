package main

import (
	"log"

	"github.com/fengmingli/orchestrator/internal/api"
	"github.com/fengmingli/orchestrator/internal/config"
	"github.com/fengmingli/orchestrator/internal/validator"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化自定义验证器
	validator.InitValidators()

	// 启动API服务器
	server := api.NewServer(cfg)
	log.Printf("启动服务器在端口 %s", cfg.Server.Port)
	
	if err := server.Run(); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}