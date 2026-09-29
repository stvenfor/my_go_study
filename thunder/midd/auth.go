// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 语句或表达式
package midd // 包名 midd：本文件提供 Gin 鉴权中间件，供 server 注册使用

// import 多行导入块开始
import (
	// 语句或表达式
	"fmt" // 格式化字符串，用于拼接正则表达式模式
	// 语句或表达式
	"net/http" // HTTP 状态码，如 401 Unauthorized
	// 语句或表达式
	"regexp" // 正则匹配，用于路径与配置中的通配规则比对
	// 语句或表达式
	"strings" // 字符串处理：Bearer 前缀、** 替换为 .*

	// 语句或表达式
	"github.com/gin-gonic/gin" // Gin 框架：Context、HandlerFunc、JSON 响应
	// 语句或表达式
	"github.com/mszlu521/thunder/config" // 全局配置中的 Auth 段：是否开启、忽略路径、需登录路径等
	// 语句或表达式
	"github.com/mszlu521/thunder/tools/jwt" // JWT 解析，从 token 得到 claims（含 UserId）
	// 闭合括号（import/var/参数列表等）
)

// Auth 返回一个 Gin 中间件：根据配置校验 JWT，并把 userId/claims 写入 Context
// 定义函数 Auth（签名开始）
func Auth(authConf *config.Auth) gin.HandlerFunc {
	// 返回闭包，每个 HTTP 请求都会执行一次下面的函数体
	// 返回语句
	return func(c *gin.Context) {
		// 若未配置 isAuth，或配置为关闭鉴权，则直接 return（注意：此处未调用 c.Next()，请求不会进入后续 handler，若期望“关闭鉴权仍走业务”需改为 c.Next()）
		// 条件判断 if
		if authConf.IsAuth == nil || !*authConf.IsAuth {
			// 返回语句
			return
			// 语句或表达式
		}
		// 遍历配置中的“忽略鉴权”路径模式（如 /api/v1/auth/**）
		// for 循环
		for _, pattern := range authConf.GetIgnores() {
			// 若当前请求路径匹配某一忽略规则，则跳过 token 校验
			// 条件判断 if
			if isMatch(c.Request.URL.Path, pattern) {
				// 语句或表达式
				c.Next() // 继续执行后续中间件与路由
				// 返回语句
				return // 本中间件结束，不再做 JWT 校验
				// 语句或表达式
			}
			// 语句或表达式
		}
		// 从请求头 Authorization 取 Bearer Token（常见格式：Bearer <jwt>）
		// 短变量声明并赋值
		tokenString := c.GetHeader("Authorization")
		// 没有带 Authorization 头则视为未携带 token
		// 条件判断 if
		if tokenString == "" {
			// 语句或表达式
			reject(c, "Authorization header is missing", authConf.NeedLogins) // 可能 401，或 needLogins 匹配时放行
			// 返回语句
			return
			// 语句或表达式
		}
		// 若前 7 个字符（忽略大小写）为 "bearer "，则去掉前缀，只保留纯 JWT 字符串
		// 条件判断 if
		if len(tokenString) > 7 && strings.ToLower(tokenString[:7]) == "bearer " {
			// 赋值语句
			tokenString = tokenString[7:]
			// 语句或表达式
		}
		// 使用项目 JWT 工具解析 token，得到 claims（内含 UserId 等）
		// 短变量声明并赋值
		claims, err := jwt.ParseToken(tokenString)
		// 解析失败：token 过期、签名错误、格式不对等
		// 条件判断 if
		if err != nil {
			// 语句或表达式
			reject(c, "Invalid token", authConf.NeedLogins)
			// 返回语句
			return
			// 语句或表达式
		}
		// 将用户 ID 字符串写入 Gin Context，供业务里 ctx.Get("userId") 使用
		// 语句或表达式
		c.Set("userId", claims.UserId)
		// 将整个 claims 也写入，便于业务取更多 JWT 载荷字段
		// 语句或表达式
		c.Set("claims", claims)
		// 语句或表达式
		c.Next() // 放行到后续中间件与具体 Handler
		// 语句或表达式
	}
	// 语句或表达式
}

// reject：在“缺 token / 无效 token”时决定是返回 401 还是对部分路径放行（needLoginUrls）
// 定义函数 reject（签名开始）
func reject(ctx *gin.Context, errMsg string, needLoginUrls []string) {
	// 若当前路径匹配 needLoginUrls 中某条，则不返回 401，继续 Next（由业务自行处理未登录）
	// for 循环
	for _, v := range needLoginUrls {
		// 条件判断 if
		if isMatch(ctx.Request.URL.Path, v) {
			// 语句或表达式
			ctx.Next()
			// 返回语句
			return
			// 语句或表达式
		}
		// 语句或表达式
	}
	// 不在放行列表：返回 JSON 401，body 中带 error 文案
	// 语句或表达式
	ctx.JSON(http.StatusUnauthorized, gin.H{"error": errMsg})
	// 语句或表达式
	ctx.Abort() // 中止后续中间件与路由，避免未授权请求执行业务逻辑
	// 语句或表达式
}

// isMatch：将配置里的路径模式（支持 **）转为正则，判断 path 是否匹配
// 定义函数 isMatch（签名开始）
func isMatch(path string, pattern string) bool {
	// 将 glob 风格的 ** 转为正则的 .*，表示任意字符任意长度
	// 赋值语句
	pattern = strings.ReplaceAll(pattern, "**", ".*")
	// 用 ^ 与 $ 包裹，表示整段路径完全匹配，避免只匹配到子串
	// 短变量声明并赋值
	regexPattern := fmt.Sprintf("^%s$", pattern)
	// 语句或表达式
	fmt.Println(regexPattern) // 调试输出编译后的正则（生产环境可考虑删除或改为日志级别）
	// 短变量声明并赋值
	matched, _ := regexp.MatchString(regexPattern, path) // 判断 path 是否匹配该正则；忽略 MatchString 的 error（简写）
	// 返回语句
	return matched
	// 语句或表达式
}
