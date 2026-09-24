# ADR 0017: 融云 IM 消息直连客户端；BFF 只管凭证、关系与群

聊天消息由 Flutter IMLib 直连融云（国内数据中心）。Go BFF 负责：登录会话校验后签发 IM Token（业务 UUID = 融云 userId）、好友关系、门店群/自由群的服务端群操作、消息业务备份入库。不中转每条聊天消息，也不用 Realtime WS 或 SSE 承载 IM。App Secret 仅服务端。与 Flutter ADR 0005（AI SSE ↛ 融云）及本仓 Realtime 职责划分一致。
