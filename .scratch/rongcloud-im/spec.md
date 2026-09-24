# RongCloud IM（真实连接 · 单聊/群聊）

**Status:** ready-for-agent

用语：
- Go：`CONTEXT.md` §即时通讯（融云）
- Flutter：`docs/contexts/chat-im/CONTEXT.md`、`CONTEXT-MAP.md`

ADR：
- Flutter `my_ai_project/docs/adr/0012-chat-imlib-not-imkit.md`
- Go [0017-rongcloud-im-bff-token-not-relay](../../docs/adr/0017-rongcloud-im-bff-token-not-relay.md)
- Go [0018-im-message-business-backup](../../docs/adr/0018-im-message-business-backup.md)
- Flutter ADR 0005（AI SSE ↛ 融云）保持有效

跨仓：Go BFF 本仓库；Flutter `my_ai_project`（`my_code_study` 的 sibling）。实现按 `issues/` 阻塞边抓票；每票独立会话 `/implement`，票间 `/clear`。

| # | Ticket | Blocked by |
|---|--------|------------|
| 01 | [IM 真连（Token + 连接）](./issues/01-im-connect-token.md) | — |
| 02 | [单聊文本 + 会话列表](./issues/02-private-text-conversation-list.md) | 01 |
| 03 | [单聊图片 + 语音](./issues/03-private-image-voice.md) | 02 |
| 04 | [双向好友 + 单聊准入](./issues/04-friends-private-admission.md) | 01, 02 |
| 05 | [消息业务备份](./issues/05-message-business-backup.md) | 02 |
| 06 | [门店群（一店一群）](./issues/06-store-group.md) | 01 |
| 07 | [自由群（群主制）](./issues/07-free-group.md) | 01, 02 |
| 08 | [文档与联调验收](./issues/08-docs-acceptance.md) | 02（完整项随 04–07） |

**Frontier 起点：** 只开 **01**。01 完成后可开 02 与 06；02 完成后可开 03 / 04 / 05 / 07；08 至少等 02。

测试缝（已确认）：
1. **IM Session usecase（Go）**
2. **Friend / Group / Backup usecases（Go）**
3. **Rong engine + session wiring（Flutter）**
4. **Chat / Friend 体验（Flutter）**

---

## Problem Statement

聊天 Tab 仍是 Mock：连不上真实融云，好友页是占位，没有门店群/自由群，消息也不进业务库。控制台应用已建好，两端需要真实 IM：单聊与群聊可用，且与 Realtime 通知、AI 小石头 SSE 分开。

## Solution

Flutter 用 **IMLib** 直连融云（国内）；Go 在 SessionAuth 后签发 **IM Token**（业务 UUID = 融云 userId），并提供好友、门店群、自由群、消息业务备份等应用层 API。分阶段交付：先真连单聊，再好友与备份，再门店群与自由群；离线推送单独立项。

## User Stories

1. As a logged-in user, I want the chat tab to connect to real RongCloud after login, so that messages are not mock-only.
2. As a logged-out user, I want IM not to connect, so that tokens are not issued without a session.
3. As a logged-in user, I want the BFF to issue an IM Token bound to my account UUID, so that the SDK can connect without holding App Secret.
4. As a developer, I want App Secret only on the server, so that the client cannot mint tokens.
5. As a logged-in user, I want my RongCloud user id to be my business UUID, so that peers and groups map 1:1 to accounts.
6. As a logged-in user, I want to send and receive private text messages, so that basic chat works.
7. As a logged-in user, I want to send and receive private image messages, so that existing chat UI image entry is real.
8. As a logged-in user, I want to send and receive private voice messages, so that existing voice UI is real.
9. As a logged-in user, I want a conversation list from the SDK, so that I see real recent chats instead of seeded mocks.
10. As a logged-in user opening a private chat with a non-friend, I want to be blocked, so that private chat requires friendship.
11. As a logged-in user, I want to search another user by phone number, so that I can send a friend request.
12. As a logged-in user, I want to search by business UUID when I already know it, so that lab/debug flows work.
13. As a logged-in user, I want to send a friend request, so that the other person can accept or reject.
14. As a user receiving a friend request, I want to accept it, so that we become friends and can private-chat.
15. As a user receiving a friend request, I want to reject or ignore it, so that I am not forced into friendship.
16. As a logged-in user, I want a friend list, so that I can start private chats from Friend.
17. As a logged-in user who becomes friends, I want to open a private chat from the friend list, so that admission is satisfied.
18. As a logged-in user sending a chat message, I want send success not to wait on business backup, so that chat stays snappy.
19. As a logged-in user, I want outbound/inbound/recall events eventually stored in our business DB, so that we can search and meet compliance needs.
20. As an operator, I want backup records keyed by message uid and conversation, so that dedupe and audit are possible.
21. As a store member joining a store, I want to be pulled into that store’s group, so that store collaboration has a default group chat.
22. As a store member leaving a store, I want to be removed from that store’s group, so that ex-members do not keep store chat access.
23. As a store member logging in, I want a backfill into the store group if I was missing, so that sync is idempotent.
24. As a product owner, I want at most one group per store, so that store chat does not fragment.
25. As a logged-in user, I want to create a free group, so that I can chat with a custom set of people.
26. As a free-group owner, I want to invite members, so that the group can grow.
27. As a free-group owner, I want to remove members, so that I can moderate the group.
28. As a free-group owner, I want to dismiss the group, so that the group ends cleanly.
29. As a free-group member (not owner), I want to leave the group, so that I can exit without owner help.
30. As a free-group member (not owner), I want invite/kick/dismiss to be denied, so that only the owner controls membership.
31. As a group member who is not friends with another member, I want to still send group messages, so that private-chat admission does not block group chat.
32. As a user of AI 小石头, I want assistant streams to stay on SSE, so that IM is not polluted with bot tokens.
33. As a user receiving system banners, I want Realtime WS to keep working independently of IM, so that notify/presence is unchanged.
34. As a Flutter developer, I want `useMockIm` off for real builds with App Key configured, so that Mock is not the default production path.
35. As a QA engineer, I want usecase-level tests for token, friends, groups, and backup, so that behavior is locked without UI flake.
36. As a lab user on LAN, I want IM against the domestic RongCloud app already created in console, so that real-device chat works with the LAN BFF.
37. As a user logging out, I want IM to disconnect, so that the next account does not reuse the previous IM session.
38. As a user whose App Secret is missing on server, I want a clear IM unavailable error, so that misconfig is obvious.
39. As a future push owner, I want offline push tracked separately, so that JPush/RongCloud push does not block this program.

## Implementation Decisions

### Modules & seams

1. **IM Session usecase (Go)** — issue IM Token after SessionAuth; userId = account UUID; fail closed if RongCloud not configured.
2. **Friend / Group / Backup usecases (Go)** — friends (request/accept/reject/list/search phone|uuid); store-group ensure + member sync; free-group create/invite/kick/quit/dismiss with owner rules; async backup ingest + query as needed.
3. **Rong engine + session wiring (Flutter)** — real `rongcloud_im_wrapper_plugin`; login → BFF session → connect; logout disconnect; drop `im_u_*` registry for real mode.
4. **Chat / Friend experience (Flutter)** — replace Mock store with SDK-backed conversations; enforce private-chat admission; wire friend UI; store/free group entry points; backup flush non-blocking on send/recall.

### Architecture (frozen)

- Client SDK ↔ RongCloud for realtime messages; BFF never relays chat payloads for delivery.
- Domestic data center; App Key on Flutter env; App Secret on Go `.env.local` only.
- Planned BFF surfaces (directional): `POST /api/v1/im/session`, friend routes, group routes, `POST /api/v1/im/messages/backup` (align Flutter stubs under `/im/...` with `/api/v1` prefix convention used elsewhere).
- Snake_case JSON; SessionAuth on all IM business routes.

### Schema (directional)

- Friend request + friendship tables (`wys_` prefix).
- Store ↔ RongCloud group id mapping (one group id per store).
- Free group metadata (owner, rong group id) if not solely on RongCloud.
- Message backup table(s): direction, im/biz user id, conversation id, message uid, type, payload, timestamps; unique on message uid (+ direction) for idempotent flush.
- Prefer server-side RongCloud group APIs for membership changes (App Secret).

### Flutter

- Keep IMLib + existing chat UI (ADR 0012).
- Flip `RongImConfig.useMockIm` when real credentials present; Mock path may remain for CI without native SDK if explicitly gated.
- Friend feature stops being a placeholder page.
- Do not route IM through `module_realtime` or `module_ai`.

### Phasing

| Phase | Scope |
|-------|--------|
| P0 | Token + IMLib connect + private chat text/image/voice + conversation list |
| P1 | Mutual friends + phone/uuid search + **message business backup** |
| P2 | Store auto groups + free groups (owner rules) |
| Side track | Offline push (RongCloud / vendors vs JPush) — out of this spec’s critical path |

## Testing Decisions

Good tests assert **external behavior** at seams, not SQL shapes or private helpers.

1. **IM Session usecase** — authed user gets token + userId=UUID; unauthenticated rejected; missing secret → clear error.
2. **Friend / Group / Backup usecases** — request/accept/search; non-friend private admission enforced at API where applicable; store join/leave sync + login backfill idempotent; free-group owner-only invite/kick/dismiss; backup ingest idempotent by message uid; backup failure must not be required for “message sent” on client.
3. **Rong engine + session wiring** — connect after token; disconnect on logout; real mode does not invent `im_u_*` ids.
4. **Chat / Friend experience** — friend then private chat media types; block non-friend private open; store group visible to members; free-group owner controls; send path does not await backup success.

Prior art: SessionAuth controller tests; usecase table-driven tests (points/wallet); Flutter module tests around auth session and chat mock (replace assertions for real wiring where feasible without requiring live RongCloud in CI — use fakes at engine boundary).

## Out of Scope

- IMKit / replacing chat UI with official kit.
- Super groups, chat rooms, RTC/calls/meetings.
- Offline push / dual JPush+RongCloud push design (side track).
- Relaying chat messages through Go or Realtime WS.
- AI assistant messages in RongCloud history.
- Client-side App Secret or client-minted tokens.
- Keeping `im_u_*` local mapping in real mode.
- Multi group per store; non-owner free-group admin roles.
- Private chat without friendship (including “store colleagues bypass”).
- Cross-region / overseas RongCloud data center.
- Full message search product UI (backup storage is in; rich search UX may follow).
- Migrating historical Mock conversations.

## Further Notes

- Console app already created; wire real App Key/Secret via local env — never commit secrets.
- Skill entry: `.agents/skills/rongcloud-im/` — fetch official Flutter IMLib + server Token docs before coding each ticket.
- Next: `/to-tickets` to materialize `issues/*.md` with blocking edges; then `/implement` from frontier ticket 01.
- Companion paths: Flutter `features/chat`, `features/friend`, `components/rongcloud_im`.
