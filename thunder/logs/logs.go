// Package logs 提供 thunder 内部模块使用的最小日志入口。
// 原为缺失包，这里用标准库 log/slog 实现常用方法，满足 upload/req/midd 的引用。
package logs

import (
	"fmt"
	"log/slog"
	"os"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

// Info 输出普通信息。
func Info(msg string) {
	logger.Info(msg)
}

// Infof 格式化输出普通信息。
func Infof(format string, args ...any) {
	logger.Info(fmt.Sprintf(format, args...))
}

// Errorf 格式化输出错误。
func Errorf(format string, args ...any) {
	logger.Error(fmt.Sprintf(format, args...))
}
