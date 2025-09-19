package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fengmingli/orchestrator/internal/dal"
	"github.com/fengmingli/orchestrator/internal/engine"
	_ "github.com/fengmingli/orchestrator/internal/engine/builtin" // register step runners
	"github.com/fengmingli/orchestrator/internal/model"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "executor",
	Short: "预案编排一次性执行器",
	Run: func(cmd *cobra.Command, args []string) {
		tplID, _ := cmd.Flags().GetString("template")
		if tplID == "" {
			log.Fatal("--template required")
		}
		dal.Init()
		// 创建 execution
		execID := fmt.Sprintf("exec_%d", time.Now().Unix())
		if err := dal.DB.Create(&model.Execution{
			ID:         execID,
			TemplateID: tplID,
		}).Error; err != nil {
			log.Fatalf("create execution: %v", err)
		}
		ex := engine.NewExecutor()
		if err := ex.Run(context.Background(), execID); err != nil {
			log.Fatalf("execution failed: %v", err)
		}
		log.Printf("execution finished. executionID=%s", execID)
	},
}

func init() {
	root.Flags().StringP("template", "t", "", "template id")
}

func main() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
