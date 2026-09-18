// seed-analytics — 刷新 analytics_records 图表友好种子（seed_chart_v1）。
//
// 用法（仓库根目录）：
//
//	./scripts/load-env.sh go run ./cmd/seed-analytics
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/stvenfor/my_go_study/internal/repository/postgres"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
	"github.com/stvenfor/my_go_study/pkg/database"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	cfg, err := config.Load(config.ResolveConfigDir(), env)
	if err != nil {
		fatalf("加载配置失败: %v", err)
	}
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		fatalf("连接 Postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		fatalf("获取 sql.DB: %v", err)
	}
	defer sqlDB.Close()

	uc := usecase.NewAnalyticsUsecase(postgres.NewAnalyticsRepository(db))
	repo := postgres.NewAnalyticsRepository(db)
	if err := repo.DeleteBySourceSystem(context.Background(), "seed"); err != nil {
		fatalf("清空旧 seed: %v", err)
	}
	if err := uc.EnsureSeedData(context.Background()); err != nil {
		fatalf("写入种子失败: %v", err)
	}
	total, err := repo.Count(context.Background())
	if err != nil {
		fatalf("统计失败: %v", err)
	}
	v1, err := repo.CountNotesContaining(context.Background(), "seed_chart_v1")
	if err != nil {
		fatalf("统计 v1 失败: %v", err)
	}
	fmt.Printf("analytics_records 已刷新: total=%d seed_chart_v1=%d\n", total, v1)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
