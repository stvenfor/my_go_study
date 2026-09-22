// seed-home-todo — 为测试号 13400000000 写入首页待办装箱演示（1大+3中+4小）。
//
//	./scripts/load-env.sh go run ./cmd/seed-home-todo
package main

import (
	"fmt"
	"os"

	"github.com/stvenfor/my_go_study/internal/repository/postgres"
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

	if err := postgres.ConsolidateLocalUsers(db); err != nil {
		fatalf("consolidate: %v", err)
	}
	if err := postgres.EnsureHomeTodoPackingDemo(db); err != nil {
		fatalf("packing demo: %v", err)
	}
	fmt.Println("ok: packing demo seeded for 13400000000 (1L+3M+4S)")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
