package dal

import (
	"fmt"
	"log"
	"os"

	"github.com/fengmingli/orchestrator/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"), os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("mysql open: %v", err)
	}
	if err = DB.AutoMigrate(&model.Template{}, &model.TemplateStep{},
		&model.Execution{}, &model.StepExecution{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}
}
