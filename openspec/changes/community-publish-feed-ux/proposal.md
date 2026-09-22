## Why

社区发布与列表的交互已能联调通，但与产品示例图差距大：公约弹窗简陋、列表图格留白、关联话题页粗糙；发布侧仍不支持多选续选、图视频切换不清空、缺少封面与放大预览。需要在**仍不真实上传**的前提下，把这些体验对齐示例图与常见盘友圈交互。

## What Changes

- 社区公约弹窗按示例图视觉与文案结构重做（标题、正文、完整公约链接、「我知道了」主按钮；仍每天最多弹一次、纯客户端）
- 社区列表图片九宫格 **铺满格子**（无上下大块白板）；视频仍只展示一个
- 关联话题页按示例图重做：搜索框、置顶「问大家」卡、列表 `#话题` + 右侧热度
- 发布选图支持**多选**，最多 9 张；已选 4 张后仍可继续加选直至上限
- 图片 ↔ 视频切换时**清空**另一侧已选内容（含预览与封面）
- 视频：可选/展示封面，支持预览播放入口；图片：点选后支持放大预览
- **非 BREAKING**：Go 社区 API 仍只收 `media_type` + 默认外链；本地多选仅影响客户端预览与发帖时 `media_type=image`

## Capabilities

### New Capabilities

- `community-publish-feed-ux`: 公约弹窗、关联话题页、列表图格铺满、发布多选续选、图视频互清、视频封面与媒体预览

### Modified Capabilities

- （无）`openspec/specs/` 下尚无已 archive 的社区主规格；媒体选择增量一并写入本能力 ADDED

## Impact

- **Flutter** `my_ai_project/features/community`：公约 Dialog、TopicSelectPage、ImageGridWidget、PublishPage / PublishViewModel；可能扩展 `ImagePickerUtils`（`pickMultiImage`）或引入相册多选组件
- **依赖**：优先 `image_picker.pickMultiImage`；若续选体验不足再评估 `wechat_assets_picker` 等（design 定夺）
- **Go BFF**：无强制 API 变更；可选后续按选中张数写入多条默认图 URL（非本 change 验收必需）
- **OpenSpec**：实现跨仓在 Flutter；契约落在本仓库 change
