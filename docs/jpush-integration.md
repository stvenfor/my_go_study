# 极光推送（JPush）接入说明与遗留清单

> 代码链路已打通（Go 登记/下发 + Flutter iOS/Android/Harmony 客户端 + 深链）。  
> **尚未具备真实 AppKey / 证书 / 厂商通道**，真机收推需完成本文「遗留清单」。

## 已完成（可不依赖 AppKey 的代码）

### Go BFF（`my_go_study`）

| 项 | 说明 |
|----|------|
| REST 客户端 | `pkg/jpush` → `POST https://api.jpush.cn/v3/push`（alias / registration_id + `extras.deeplink`） |
| 配置 | `JPUSH_APP_KEY` / `JPUSH_MASTER_SECRET` / `JPUSH_APNS_PRODUCTION`（`.env.example`） |
| 设备表 | `wys_push_devices` + migration `20260923160000_jpush_devices` |
| API | `POST /api/v1/push/devices`（登记）、`POST /api/v1/push/send`（调试下发） |
| 离线兜底 | `RealtimePushUsecase.DeliverPush` 成功后 best-effort 调极光（未配密钥则 skip） |
| Worker | `jpush:register` 任务写入设备表 |

### Flutter（`my_ai_project`）

| 项 | 说明 |
|----|------|
| `wys_push` | `JPushManager`：iOS/Android `addEventHandler` + Harmony `setCallBackHarmony`；隐私后 setup；alias 重试 |
| AppKey | `PushConfig` / `LinkingConfig` 使用 **PLACEHOLDER**（不含 tpj 密钥） |
| Coordinator | 点击通知 / Want URI 去重跳转；channel `com.xiaomao.flutter/deeplink` |
| linking | 隐私同意后 init；登录 `alias=user.id`；登出 `deleteAlias`；上报 BFF |
| Mock 自动 | AppKey 仍为占位时走 `MockPushService`，填真实 Key 后走 `JPushPushService` |
| 深链 | `enableDeeplink=true`；scheme `xiaomao://`；路由表 `DeeplinkRouteTable` |
| Harmony | `EntryAbility`：`offerWant` + `setClickWant` |
| iOS | `UIBackgroundModes=remote-notification`；`aps-environment=development` 占位；`xiaomao` URL Scheme |
| Android | `POST_NOTIFICATIONS` + `JPUSH_APPKEY` meta-data 占位 |

### 推送 Payload 约定

```json
{
  "title": "标题",
  "body": "正文",
  "extras": {
    "deeplink": "xiaomao://app/community"
  }
}
```

服务端 `extras.deeplink` 会写入 android / ios / hmos notification extras。  
Alias 约定：**业务 `user_id`（Flutter `User.id`）**。

### 联调（密钥配齐后）

```bash
# Go
# 填 .env：JPUSH_APP_KEY / JPUSH_MASTER_SECRET
make run
# 登录后：
curl -s -X POST http://127.0.0.1:8080/api/v1/push/send \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Session-ID: $SID" -H "X-Device-ID: $DID" \
  -H "Content-Type: application/json" \
  -d '{"body":"hello","title":"test","deeplink":"xiaomao://app/community"}'
```

Flutter：同意隐私 → 进首页 → 登录 → 日志应出现 `registrationID` / `setAlias`。

---

## 遗留清单（非代码 / 控制台 / 证书）

### 极光控制台（阻塞真机收推）

- [ ] 创建极光应用（建议 **dev / prod 两套** AppKey）
- [ ] 填写客户端：`PushConfig.testAppKey` / `productionAppKey`；`LinkingConfig.jpushAppKey`；AndroidManifest `JPUSH_APPKEY`
- [ ] 填写服务端：`.env` → `JPUSH_APP_KEY` + `JPUSH_MASTER_SECRET`
- [ ] 生产环境：`JPUSH_APNS_PRODUCTION=true`；客户端 product 环境使用正式 AppKey
- [ ] 控制台配置三端包名 / Bundle ID / Harmony 包名与签名指纹

### iOS APNs

- [ ] Apple Developer：Push Notifications 能力 + 描述文件
- [ ] 上传 **p8**（或 p12）到极光：TeamId / KeyId / BundleId（见 `LinkingConfig.apns*` 占位）
- [ ] Xcode：Release 将 `aps-environment` 改为 `production`
- [ ] Universal Links（`https://xiaomaomain.com/...`）：Associated Domains + apple-app-site-association（当前仅 custom scheme 可用）

### Android 厂商通道

- [x] Android 13+：`JPushPushService` 隐私同意后申请 `POST_NOTIFICATIONS`（经 `module_utils` / hosted `permission_handler`）
- [ ] 华为 / 小米 / OPPO / vivo / 荣耀：控制台开通 + 对应插件依赖（参考 tpj `android_vendor_push_migration.md`，**勿拷贝其 AppKey/包名**）
- [ ] `LinkingConfig.androidVendorChannelsEnabled` 打开并接入厂商参数
- [ ] 各厂商应用签名与包名对齐

### HarmonyOS

- [ ] AGC 推送 / 极光 Harmony 参数（client_id、签名指纹）
- [ ] 真机验证：冷启动 Want、点击通知、`setClickWant` → Dart 回调 → 深链
- [ ]（可选）后台自定义消息 Ability（进程不存在时），按极光 Harmony 文档

### 产品 / 业务

- [ ] 深链路由表按业务扩展（当前：短视频 / 社区 / 聊天）
- [ ] 调试球「发极光」按钮对接 `POST /api/v1/push/send`（可选）
- [ ] 生产关闭 Mock 路径后的回归：隐私门闩、alias 登录/登出、多端互踢与推送并存策略

### 运维

- [ ] 迁移：执行 `migrations/20260923160000_jpush_devices.up.sql`（或依赖 API `EnsurePushDeviceSchema`）
- [ ] Worker：异步 Realtime + `jpush:register` 需 `queue.enabled=true` + `make run-worker`

---

## 相关路径

| 仓库 | 路径 |
|------|------|
| Go | `pkg/jpush/`、`internal/usecase/jpush_usecase.go`、`internal/delivery/http/controller/jpush_controller.go` |
| Flutter | `components/wys_push/`、`components/linking/lib/push/`、`ohos/.../EntryAbility.ets` |
| 参考（勿引入资源） | `tpj-flt/components/tf_push`、`docs/jpush_business.md` |
