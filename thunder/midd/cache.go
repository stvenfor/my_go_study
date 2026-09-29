// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 midd
package midd

// import 多行导入块开始
import (
	// 语句或表达式
	"bytes"
	// 语句或表达式
	"encoding/json"
	// 语句或表达式
	"fmt"

	// 语句或表达式
	"github.com/gin-gonic/gin"
	// 语句或表达式
	"github.com/mszlu521/thunder/cache"
	// 语句或表达式
	"github.com/mszlu521/thunder/config"
	// 语句或表达式
	"github.com/mszlu521/thunder/logs"
	// 语句或表达式
	"github.com/mszlu521/thunder/res"
	// 语句或表达式
	"github.com/mszlu521/thunder/tools/crypro"
	// 语句或表达式
	"github.com/mszlu521/thunder/tools/gptr"

	// 语句或表达式
	"io"
	// 语句或表达式
	"net/http"
	// 语句或表达式
	"time"
	// 闭合括号（import/var/参数列表等）
)

// CustomResponseWriter 自定义 ResponseWriter 来捕获响应内容
// 定义结构体类型 CustomResponseWriter
type CustomResponseWriter struct {
	// 语句或表达式
	gin.ResponseWriter
	// 语句或表达式
	body *bytes.Buffer
	// 语句或表达式
}

// 定义函数 Write（签名开始）
func (w *CustomResponseWriter) Write(p []byte) (n int, err error) {
	// 将数据写入 body
	// 语句或表达式
	w.body.Write(p)
	// 使用 gin 原有的 ResponseWriter 将数据写回客户端
	// 返回语句
	return w.ResponseWriter.Write(p)
	// 语句或表达式
}

// 定义函数 Cache（签名开始）
func Cache(cacheConfig *config.Cache) gin.HandlerFunc {
	// 返回语句
	return func(c *gin.Context) {
		//打印超时时间
		// 短变量声明并赋值
		start := time.Now()
		// for 循环
		for _, pattern := range cacheConfig.GetNeedCache() {
			// 条件判断 if
			if isMatch(c.Request.URL.Path, pattern) {
				//对数据进行缓存
				// 条件判断 if
				if c.Request.Method == http.MethodPost {
					// 短变量声明并赋值
					body, err := io.ReadAll(c.Request.Body)
					// 条件判断 if
					if err != nil {
						// 语句或表达式
						c.AbortWithStatus(http.StatusInternalServerError)
						// 返回语句
						return
						// 语句或表达式
					}
					// 语句或表达式
					c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
					// 短变量声明并赋值
					cacheKey := fmt.Sprintf("CACHE:%s:%s", c.Request.RequestURI, crypro.Md5(body))
					// 短变量声明并赋值
					redisCache := cache.NewRedisCache()
					// 短变量声明并赋值
					writer := &CustomResponseWriter{body: bytes.NewBuffer([]byte{}), ResponseWriter: c.Writer}
					// 语句或表达式
					c.Writer = writer
					// 语句或表达式
					logs.Infof("cache -------start----- time: %s", time.Since(start))
					// 条件判断 if
					if redisCache.Exist(cacheKey) {
						// 语句或表达式
						logs.Infof("cache Exist time: %s", time.Since(start))
						// 短变量声明并赋值
						cacheData, err := redisCache.Get(cacheKey)
						// 条件判断 if
						if err == nil {
							// 语句或表达式
							c.Data(http.StatusOK, "application/json", []byte(cacheData))
							// 语句或表达式
							c.Abort()
							// 语句或表达式
							logs.Infof("cache time: %s", time.Since(start))
							// 返回语句
							return
							// 语句或表达式
						}
						// 语句或表达式
						logs.Errorf("get cache err: %v", err)
						// 语句或表达式
						c.Next()
						// 语句或表达式
					} else {
						// 语句或表达式
						c.Next()
						// 语句或表达式
					}
					// 条件判断 if
					if c.Writer.Status() == 200 {
						// 短变量声明并赋值
						responseBody := writer.body
						// 变量声明
						var result res.Result
						// 短变量声明并赋值
						err := json.Unmarshal(responseBody.Bytes(), &result)
						// 条件判断 if
						if err != nil {
							// 语句或表达式
							logs.Errorf("cache json Unmarshal err: %v", err)
							// 语句或表达式
						} else {
							// 条件判断 if
							if result.Code == res.OK {
								// 条件判断 if
								if cacheConfig.Expire == nil {
									// 语句或表达式
									cacheConfig.Expire = gptr.Of(int64(5 * 60)) //默认5分钟
									// 语句或表达式
								}
								// 短变量声明并赋值
								err := redisCache.Set(cacheKey, string(responseBody.Bytes()), cacheConfig.GetExpire())
								// 条件判断 if
								if err != nil {
									// 语句或表达式
									logs.Errorf("cache redisCache.Set err: %v", err)
									// 语句或表达式
								}
								// 语句或表达式
							}
							// 语句或表达式
						}
						// 语句或表达式
					}
					// 语句或表达式
				}
				// 语句或表达式
			}
			// 语句或表达式
		}
		// 语句或表达式
		c.Next()
		// 语句或表达式
	}
	// 语句或表达式
}
