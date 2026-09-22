## Why

社区发布页目前用「无媒体 / 图片 / 视频」芯片直接设 `media_type`，没有相册或相机选择体验，与示例图及用户预期不符。需要在**仍不真实上传**（服务端继续写默认外链）的前提下，补上真实选图/选视频/拍照/摄像流程。

## What Changes

- 发布页点击「图片」或「视频」后弹出来源选择（相册 / 相机），走系统真实选择流程
- 选中后在发布页展示本地预览（缩略图或视频封面占位）；`media_type` 仍为 image/video 互斥
- 发帖请求**不上传**本地文件；继续只传 `media_type`（及既有字段），由 BFF 填入默认图片或视频 URL
- 取消选择或清空媒体后可回到「无媒体」
- 权限拒绝时给出可理解提示，不崩溃
- **非 BREAKING**：Go 社区 API 契约不变

## Capabilities

### New Capabilities

- `community-publish-media-picker`: 发布动态时的真实媒体选择与本地预览；上传仍占位为默认外链

### Modified Capabilities

- （无）`openspec/specs/` 下尚无社区能力主规格；本变更为新能力

## Impact

- **Flutter**（`my_ai_project/features/community`）：发布页 / PublishViewModel；复用或接入已有 `image_picker`（根工程 / `module_utils`）
- **权限**：iOS/Android 相册与相机 Info.plist / Manifest 文案（若尚未覆盖社区入口）
- **Go BFF**：无 API / 表结构变更；继续 `docs/community-posts-api.md` 默认媒体行为
- **OpenSpec 根**：本仓库 `my_go_study`；实现跨仓时以本 change 为契约，改 Flutter 在兄弟仓执行
