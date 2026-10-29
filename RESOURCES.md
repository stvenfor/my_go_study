# Go 后端分层 / Clean Architecture Resources

## Knowledge

- [Effective Go — Interfaces（Go 官方）](https://go.dev/doc/effective_go)
  官方口径：接口描述行为而非身份、隐式满足、单方法接口更通用、「接口越大抽象越弱」。用来说「Go 为什么这样设计接口」，别引二手博客。
- [The Clean Architecture Dependency Rule（Prentice Hall 书摘）](https://www.informit.com/articles/article.aspx?p=2832399)
  Uncle Bob《Clean Architecture》第 22 章官方节选：Dependency Rule 原文、四层（Entities / Use Cases / Interface Adapters / Frameworks & Drivers）、控制流与源码依赖方向相反。面试引「原文」时就引这篇。
- [The Clean Architecture（作者原站）](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
  同心圆图的出处。需要「那张图本来是什么意思」时用它，而不是转述版。
- [Go: Under the Hood — 4.2 Interfaces](https://golang.design/under-the-hood/en/part2lang/ch04type/interface/)
  学院派、含接口值内部表示（itab / iface）与设计取舍。被追问「interface 底层是什么、代价多少」时用。
- [What "accept interfaces, return structs" means in Go](https://medium.com/@cep21/what-accept-interfaces-return-structs-means-in-go-2fe879e25ee8)
  对这条口号的细致反驳/限定。适合准备「这句话到底对不对」的追问，避免只会背口号。
- 本仓库 [`docs/architecture-learning-guide.md`](./docs/architecture-learning-guide.md)
  分层、优缺点、认证/交易/Realtime 时序。**讲自己项目时的第一手依据**，面试答案的骨架应来自这里而非通用教程。
- 本仓库 [`docs/feishu/go-deep-dive/01-工程结构与go-work多模块.md`](./docs/feishu/go-deep-dive/01-工程结构与go-work多模块.md) · [`03-启动流程与依赖注入.md`](./docs/feishu/go-deep-dive/03-启动流程与依赖注入.md)
  模块划分与装配（`cmd/api/main.go` 里怎么把 Redis 实现塞进接口）。

## Wisdom (Communities)

- [r/golang](https://reddit.com/r/golang)
  帖量大、回复快；架构类问题会收到「过度设计」的诚实批评。用来自检分层说法是否站得住。
- [Go Forum](https://forum.golangbridge.org/)
  官方社区论坛，语言细节问题回答权威。用于接口/依赖方向的语法级疑问。
- Gophers Slack（`#architecture`、`#newbies`）
  需要邀请链接入口；用于长对话式设计与代码评审。

## Gaps（下次课要补）

- **依赖方向自动化检查工具**：`go list -deps` 手工验证可用，但还没有权威文档支撑「CI 里怎么强制不反向依赖」。下次查证 `go-arch-lint` / `depcheck` / 自定义 script 的官方文档，再写进课程。
- **中文面试语料**：目前只有英文一手资料；若目标是中文表达，需要找高质量中文源码/架构文章作对照。
