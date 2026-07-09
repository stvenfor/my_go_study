// =============================================================================
// main.go — 手动触发定时广播（开发联调：make trigger-hourly-notify）
// =============================================================================
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/stvenfor/my_go_study/pkg/config"
	"github.com/stvenfor/my_go_study/pkg/queue"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	cfg, err := config.Load(config.ResolveConfigDir(), env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}
	if !cfg.Queue.Enabled {
		fmt.Fprintln(os.Stderr, "queue.enabled=false，请先启用队列")
		os.Exit(1)
	}

	client := queue.NewAsynqClient(*cfg)
	defer client.Close()

	taskID, err := client.EnqueueBroadcastNotify(context.Background(), queue.BroadcastNotifyPayload{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "入队失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("已入队 scheduled:broadcast_notify，taskId=%s\n", taskID)
	fmt.Println("请确认 make run-worker 正在运行以消费任务")
}
