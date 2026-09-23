# 华为账号一键登录（HarmonyOS Account Kit）

对标微信登录：鸿蒙端拿 Authorization Code → Go BFF 换票 → local Auth 发业务 session。

官方参考：
- [一键登录（手机号 + UnionID）](https://developer.huawei.com/consumer/cn/doc/harmonyos-guides/account-phone-unionid-login)
- [按钮视觉资源](https://developer.huawei.com/consumer/cn/design/resource/#012589f7-4906-4ffc-ac39-eb7957fb4d76)

## 已打通（代码）

| 层 | 内容 |
|----|------|
| Go BFF | `POST /api/v1/user/huawei/login`，body `{ code, device_id, platform }`；`platform` 同微信：`android` \| `ios`（鸿蒙端传 `android`） |
| 换票 | `oauth2/v3/token` → `GOpen.User.getInfo` → UnionID + 完整手机号 |
| 账号 | `LoginOrRegisterHuawei`：先 `hw_{unionID}@huawei.local`，再按手机号合并 |
| 鸿蒙壳 | `my_kmp_project/harmonyApp` 登录页：`LoginWithHuaweiIDButton` + 协议勾选 + 调 BFF |

环境变量（`.env` / `.env.example`）：

```bash
HUAWEI_CLIENT_ID=
HUAWEI_CLIENT_SECRET=
# HUAWEI_REDIRECT_URI=   # 可选，与 AGC OAuth 回调一致时再填
```

要求：`auth.provider=local`。

## 非代码端待办（上线前必须做）

1. **AGC 申请 scope `quickLoginMobilePhone`（华为账号一键登录）**  
   未批通过时客户端会报 `1001502014`（未申请 scopes）。这是当前唯一刻意留给人工的权限门槛。
2. **在 AppGallery Connect 创建/确认应用**，拿到 **Client ID / Client Secret**，写入服务端 `.env`（Secret 禁止进 App）。
3. **服务端部署在中国大陆**（华为要求：获取完整手机号的服务器须在境内）。
4. **签名与包名**与 AGC 应用一致（鸿蒙 `bundleName` 当前示例为 `com.example.harmonyapp`，上架前换成正式包名并在 AGC 登记）。
5. **按钮 UX**：使用官方红色一键登录样式（`Style.BUTTON_RED` / QUICK_LOGIN）；勿自绘违规按钮。
6. **协议**：登录页须可跳转《华为账号用户认证协议》  
   `https://privacy.consumer.huawei.com/legal/id/authentication-terms.htm?code=CN&language=zh-CN`
7. **局域网联调**：改 `harmonyApp/.../auth/AuthConfig.ets` 的 `BFF_BASE_URL`（或与 `LanHost` / `.env.lan` 同步）。

## 联调自检

```bash
# 未配 Client Secret 时应 503 文案提示未配置
curl -s -X POST http://127.0.0.1:8080/api/v1/user/huawei/login \
  -H 'Content-Type: application/json' \
  -d '{"code":"x","device_id":"dev-1","platform":"android"}'
```

真机：DevEco 跑 `harmonyApp` → 登录页勾选协议 → 点华为一键登录 → 成功后 `isLoggedIn` 且 preferences 写入 token/session。
