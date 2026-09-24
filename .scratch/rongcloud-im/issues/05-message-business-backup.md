# 05 — 消息业务备份

**What to build:** 单聊（及后续群聊复用同一备份通道）在收、发、撤回后，事件异步写入业务库；发送成功不依赖备份成功。同 message uid 重复提交幂等。

**Blocked by:** 02 — 单聊文本 + 会话列表

**Status:** done

- [x] Go 接收备份事件并持久化（含方向、会话、message uid、类型、载荷、时间）
- [x] 重复 message uid 不产生重复有效记录（幂等）
- [x] Flutter 发送/撤回路径不阻塞等待备份成功；失败可入队重试或可观测
- [x] Backup usecase 缝有自动化测试
