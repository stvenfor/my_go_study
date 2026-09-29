// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 db
package db

// import 多行导入块开始
import (
	// 语句或表达式
	"context"
	// 语句或表达式
	"fmt"
	// 语句或表达式
	"gorm.io/driver/mysql"
	// 语句或表达式
	"gorm.io/gorm"
	// 语句或表达式
	"gorm.io/gorm/logger"
	// 语句或表达式
	"log"
	// 语句或表达式
	"os"
	// 语句或表达式
	"time"
	// 闭合括号（import/var/参数列表等）
)

// 定义结构体类型 MySQL
type MySQL struct {
	// 语句或表达式
	GormConfig *gorm.Config
	// 语句或表达式
	Username string
	// 语句或表达式
	Password string
	// 语句或表达式
	Host string
	// 语句或表达式
	Port int
	// 语句或表达式
	Database string
	// 语句或表达式
	PingTimeout time.Duration
	// 语句或表达式
	MaxIdleConns int
	// 语句或表达式
	MaxOpenConns int
	// 语句或表达式
	GormDB *gorm.DB
	// 语句或表达式
}

// 定义函数 Init（签名开始）
func (m *MySQL) Init() error {
	// 短变量声明并赋值
	dsn := "%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local"
	// 赋值语句
	dsn = fmt.Sprintf(dsn, m.Username, m.Password, m.Host, m.Port, m.Database)

	// 条件判断 if
	if m.GormConfig == nil {
		// 短变量声明并赋值
		newLogger := logger.New(
			// 语句或表达式
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			// 语句或表达式
			logger.Config{
				// 语句或表达式
				SlowThreshold: time.Second, // Slow SQL threshold
				// 语句或表达式
				LogLevel: logger.Info, // Log level
				// 语句或表达式
				IgnoreRecordNotFoundError: true, // Ignore ErrRecordNotFound error for logger
				// 语句或表达式
				ParameterizedQueries: true, // Don't include params in the SQL logs
				// 语句或表达式
				Colorful: false, // Disable color
				// 列表项（字段/参数/元素），行末逗号
			},
		// 闭合括号（import/var/参数列表等）
		)
		// 语句或表达式
		m.GormConfig = &gorm.Config{
			// 列表项（字段/参数/元素），行末逗号
			Logger: newLogger,
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 短变量声明并赋值
	db, err := gorm.Open(mysql.Open(dsn), m.GormConfig)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return err
		// 语句或表达式
	}
	// 语句或表达式
	m.GormDB = db
	// 短变量声明并赋值
	conn, err := db.DB()
	// 条件判断 if
	if err != nil {
		// 返回语句
		return err
		// 语句或表达式
	}
	// 短变量声明并赋值
	ctx, cancel := context.WithTimeout(context.Background(), m.PingTimeout)
	// defer 延迟执行
	defer cancel()
	// 赋值语句
	err = conn.PingContext(ctx)
	// 条件判断 if
	if err != nil {
		// 返回语句
		return err
		// 语句或表达式
	}
	// 条件判断 if
	if m.MaxIdleConns != 0 {
		// 语句或表达式
		conn.SetMaxIdleConns(m.MaxIdleConns)
		// 语句或表达式
	}
	// 条件判断 if
	if m.MaxOpenConns != 0 {
		// 语句或表达式
		conn.SetMaxOpenConns(m.MaxOpenConns)
		// 语句或表达式
	}
	// 返回语句
	return nil
	// 语句或表达式
}
