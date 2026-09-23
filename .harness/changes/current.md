# Current change (harness pointer)

| 字段 | 值 |
|------|-----|
| Active slice | JPush 前后端推送（iOS/Android/Harmony + 深链） |
| Last slice | 华为账号一键登录（人侧 AGC 待办） |
| Epic | 极光推送 |
| Program | — |
| Role | executor 已落地代码 → **人** 填极光控制台 / 证书 |
| agent:pre | — |
| agent:post | `go test` pkg/jpush + usecase 已绿；Flutter 待本地 `flutter test` |
| 验收 tick | Partial — 代码打通；AppKey/APNs/厂商通道未配 |
| Next | 人：极光建应用 + 填密钥；见 `docs/jpush-integration.md` 遗留清单 |

## Notes

- Go: `POST /api/v1/push/devices` · `POST /api/v1/push/send`；Realtime 离线兜底极光
- Flutter: `wys_push` + `module_linking`；AppKey 占位则自动 Mock
- 遗留清单：`docs/jpush-integration.md`
- 旁路已完成（非本 Epic）：社区 `VideoPlayPage` → CPF Chewie + tpc `video_player` OHOS；`pigeon_runtime_stub`；小视频未改
