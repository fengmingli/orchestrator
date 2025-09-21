package dal

import (
	"fmt"
	"time"

	"github.com/fengmingli/orchestrator/internal/config"
	"github.com/fengmingli/orchestrator/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitMySQL 初始化MySQL数据库
func InitMySQL(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
		cfg.Database.Charset)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		// 如果MySQL连接失败，尝试使用SQLite
		fmt.Printf("MySQL连接失败，尝试使用SQLite: %v\n", err)
		return InitSQLite()
	}

	// 获取底层sql.DB对象
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移数据库表结构
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return db, nil
}

// InitSQLite 初始化SQLite数据库
func InitSQLite() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("orchestrator.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("连接SQLite数据库失败: %w", err)
	}

	// 自动迁移数据库表结构
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return db, nil
}

// autoMigrate 自动迁移数据库表结构
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Step{},
		&model.WorkflowTemplate{},
		&model.WorkflowTemplateStep{},
		&model.WorkflowExecution{},
		&model.WorkflowStepExecution{},
	)
}
