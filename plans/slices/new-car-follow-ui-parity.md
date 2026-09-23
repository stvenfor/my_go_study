# Slice — new-car-follow-ui-parity

## Slice Brief
- SOURCE_MODULE: 新车跟进档案（C1 Flutter 壳）
- TARGET_MODULE: 配对仓 Flutter `my_ai_project`（对齐「新车成交」视觉与交互骨架）
- Source entry: 人批「对标新车成交；范围全部做；假上传即可」；成交页 `DealInvoiceDemoPage` + widgets
- Target entry: 首页「新车跟进」→ 列表 / 建档 / 详情（本轮仍本人×当前店；店管全店 → C4）
- 本轮 ONLY:
  - 列表页：顶栏 ProfileHeader 风格（展示名/职务/店名/头像 + 四格统计）、悬浮/吸顶 Tab、列表 item 卡片、空态、下拉刷新/上拉更多、底部主 CTA（建档 FAB 或等价）
  - 建档页：表单分区、级别 ABEH 选项（可读文案+字母）、意向车型等；**假上传**（本地选图或占位 URL，不接 OSS/OCR）
  - 详情页：客户头区、级别/意向/阶段/下次跟进结构化展示；改级别入口；假上传区只读或可换占位图
  - 复用/镜像 `deal_invoice_*` 布局模式与 `module_common_ui`（允许复制 widgets 到 `new_car_follow/widgets/`，不强抽跨模块公共库）
  - 首页入口卡若观感明显落后于成交入口，做同量级对齐（文案已是「新车跟进」则只调样式）
  - CONTEXT.md 补「新车跟进档案」术语节（若仍缺失）
- 不做:
  - 跟进流水表与 `/logs`（→ C2）
  - 店管全店可见 / summary 全店口径（→ C4）
  - 真 OSS 上传、发票 OCR、审核台、转交、跨店报表
  - 改 Go list/summary 语义（本轮纯 Flutter + 可选 CONTEXT）
  - C3 待办深链 `file_id`
- 验收:
  - 人证：列表首屏一眼可对上「新车成交」骨架（顶栏四格 + 吸顶 Tab + 卡片行 + 底 CTA），不再是裸 `ListTile`/`ChoiceChip` 堆叠
  - 建档可提交（假图可空或占位）；详情可改级别并刷新意向档展示
  - 无新依赖；不接真实对象存储
- 文件白名单:
  - （配对仓）my_ai_project/features/home/lib/new_car_follow/**
  - （配对仓）my_ai_project/features/home/lib/home/view/widgets/home_dashboard_widgets.dart（仅入口样式必要时）
  - （配对仓）my_ai_project/features/home/lib/home/navigation/new_car_follow_navigation.dart
  - （本仓）CONTEXT.md
  - plans/epics/new-car-follow.md
  - plans/slices/new-car-follow-ui-parity.md
  - docs/acceptance-records/2026-09-23-new-car-follow-ui-parity.md
- 文件黑名单:
  - internal/**（本 Slice 不改 BFF）
  - deal_invoice usecase / 成交审核 API
  - 真上传 SDK / OSS 配置
- 验证命令:
  - （配对仓）`dart analyze` 触及包或 `flutter analyze` 限 `features/home`
  - 人证：对照成交页截图/真机并排
- 证据: docs/acceptance-records/2026-09-23-new-car-follow-ui-parity.md
- Accept 模式: Partial（Flutter 人证为主；无 Go 机跑门槛）

## 对标要点（冻结）

| 区域 | 成交参考 | 跟进映射 |
|------|----------|----------|
| 顶栏 | `DealInvoiceProfileHeader` | summary：active/overdue/high_intent/lost |
| Tab | Sticky TabBar | 全部/逾期/高/中/低（或现有 `NewCarFollowTab`） |
| 行 | `DealInvoiceListItem` | 客户名、级别+意向、车型、下次跟进 |
| CTA | 上传 FAB | 建档 |
| 附件 | 占位图 URL | 假上传（同语义） |

## Context Card — new-car-follow-ui-parity
- 已完成: Flutter 列表/建档/详情对标成交骨架；假上传；CONTEXT 术语节；acceptance Partial
- 未做/Deferred: C2 流水；C4 店管全店；真上传；C3 深链；人证并排截图
- 关键文件: my_ai_project/.../new_car_follow/** + CONTEXT.md + acceptance-records
- harness: post ok? dart analyze lib/new_car_follow 绿（info only）
- 下一 Slice 建议: `new-car-follow-c2-follow-log`
- 已知坑: 成交 widgets 在 settings；跟进侧已复制 sticky/header 模式，勿硬依赖 settings
