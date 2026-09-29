// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 server
package server

// import 多行导入块开始
import (
	// 语句或表达式
	"context"
	// 语句或表达式
	"fmt"
	// 语句或表达式
	"log"
	// 语句或表达式
	"net/http"
	// 语句或表达式
	"os"
	// 语句或表达式
	"os/signal"
	// 语句或表达式
	"syscall"
	// 语句或表达式
	"time"

	// 语句或表达式
	"github.com/gin-gonic/gin"
	// 语句或表达式
	"github.com/mszlu521/thunder/config"
	// 语句或表达式
	"github.com/mszlu521/thunder/event"
	// 语句或表达式
	"github.com/mszlu521/thunder/pay/wxPay"
	// 闭合括号（import/var/参数列表等）
)

// Server 是我们应用的核心结构体
// 定义结构体类型 Server
type Server struct {
	// 语句或表达式
	Engine *gin.Engine
	// 语句或表达式
	httpServer *http.Server
	// 语句或表达式
	conf *config.Config
	// 语句或表达式
	Close func()
	// 语句或表达式
}

// NewServer 创建一个新的 Server 实例
// 定义函数 NewServer（签名开始）
func NewServer(conf *config.Config) *Server {
	// 条件判断 if
	if conf.Server == nil {
		// 语句或表达式
		panic("Server config is required ")
		// 语句或表达式
	}
	// 根据配置设置 Gin 模式
	// 语句或表达式
	gin.SetMode(conf.Server.GetMode())

	// 初始化微信支付
	// 条件判断 if
	if conf.Pay != nil {
		// 短变量声明并赋值
		wxPayConfig := conf.Pay.WxPay
		// 条件判断 if
		if wxPayConfig != nil {
			// 语句或表达式
			wxPay.Init(wxPayConfig)
			// 语句或表达式
		}
		// 语句或表达式
	}

	// 短变量声明并赋值
	engine := gin.Default() // 使用默认的中间件 (Logger, Recovery)
	//自定义的一些中间件，可通过配置文件开启，减少代码重复书写
	// 语句或表达式
	UseCustomMidd(conf, engine)
	// 返回语句
	return &Server{
		// 列表项（字段/参数/元素），行末逗号
		Engine: engine,
		// 列表项（字段/参数/元素），行末逗号
		conf: conf,
		// 语句或表达式
	}
	// 语句或表达式
}

// RegisterRouters 批量注册路由
// 参数是实现了 IRouter 接口的实例
// 定义函数 RegisterRouters（签名开始）
func (s *Server) RegisterRouters(event event.IEvent, routers ...IRouter) []func() error {
	//注册 事件
	// 语句或表达式
	event.Register()
	// 变量声明
	var closeFuncs []func() error
	// for 循环
	for _, r := range routers {
		// 语句或表达式
		r.Register(s.Engine)
		// 条件判断 if
		if closer, ok := r.(interface{ Close() error }); ok {
			// 赋值语句
			closeFuncs = append(closeFuncs, closer.Close)
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 语句或表达式
	log.Println("Routers registered successfully.")
	// 返回语句
	return closeFuncs
	// 语句或表达式
}

// Start 启动服务并实现优雅启停
// 定义函数 Start（签名开始）
func (s *Server) Start() {
	// 从配置中获取服务器地址和超时设置
	// 短变量声明并赋值
	address := fmt.Sprintf("%s:%d", s.conf.Server.GetHost(), s.conf.Server.GetPort())
	// 短变量声明并赋值
	readTimeout := s.conf.Server.GetReadTimeout()
	// 短变量声明并赋值
	writeTimeout := s.conf.Server.GetWriteTimeout()

	// 语句或表达式
	s.httpServer = &http.Server{
		// 列表项（字段/参数/元素），行末逗号
		Addr: address,
		// 列表项（字段/参数/元素），行末逗号
		Handler: s.Engine,
		// 列表项（字段/参数/元素），行末逗号
		ReadTimeout: readTimeout,
		// 列表项（字段/参数/元素），行末逗号
		WriteTimeout: writeTimeout,
		// 语句或表达式
	}

	// 启动 http server (goroutine)
	// 启动 goroutine
	go func() {
		// 语句或表达式
		log.Printf("Server starting on http://%s", address)
		// 条件判断 if
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 语句或表达式
			log.Fatalf("Failed to start server: %v", err)
			// 语句或表达式
		}
		// 语句或表达式
	}()

	// ---- 优雅启停逻辑 ----
	// 创建一个 channel 用于接收系统信号
	// 短变量声明并赋值
	quit := make(chan os.Signal, 1)
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM 信号
	// 语句或表达式
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞在此，直到接收到信号
	// 语句或表达式
	<-quit
	// 语句或表达式
	log.Println("Shutting down server...")
	// 条件判断 if
	if s.Close != nil {
		// 语句或表达式
		s.Close()
		// 语句或表达式
	}
	// 创建一个有5秒超时的 context
	// 短变量声明并赋值
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer 延迟执行
	defer cancel()

	// 调用 Shutdown() 优雅地关闭服务器
	// 这会等待正在处理的请求完成，但不会接受新请求
	// 条件判断 if
	if err := s.httpServer.Shutdown(ctx); err != nil {
		// 语句或表达式
		log.Fatalf("Server shutdown failed: %v", err)
		// 语句或表达式
	}

	// 语句或表达式
	log.Println("Server exited gracefully.")
	// 语句或表达式
}
