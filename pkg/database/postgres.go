// postgres.go 负责初始化 GORM PostgreSQL 连接与连接池。
package database

import (
	"fmt"
	"time"

	"github.com/stvenfor/my_go_study/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgres 创建 PostgreSQL 连接。
func NewPostgres(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("连接 PostgreSQL 失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime())

	if err := pingWithRetry(sqlDB, 5, 2*time.Second); err != nil {
		return nil, fmt.Errorf("PostgreSQL ping 失败: %w", err)
	}

	return db, nil
}
