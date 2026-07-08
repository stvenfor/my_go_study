// ping.go 提供数据库连通性重试检测。
package database

import (
	"database/sql"
	"fmt"
	"time"
)

// pingWithRetry 带重试的数据库 ping。
func pingWithRetry(db *sql.DB, attempts int, interval time.Duration) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := db.Ping(); err != nil {
			lastErr = err
			time.Sleep(interval)
			continue
		}
		return nil
	}
	return fmt.Errorf("重试 %d 次后仍无法连接: %w", attempts, lastErr)
}
