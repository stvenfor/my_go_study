## Context

See `proposal.md` — Why. Publish UI today toggles `media_type` via chips without opening gallery/camera (`my_ai_project` community `PublishPage` / `PublishViewModel`). Go create-post already fills default image/video URLs when `media_type` is set and local URLs are empty (`docs/community-posts-api.md`). Root Flutter app and `module_utils` already depend on `image_picker`.

Constraints: OpenSpec root is `my_go_study`; UI work lands in sibling `my_ai_project`. No BFF upload API in this change.

## Goals / Non-Goals

**Goals:**

- Replace chip-only media selection with album/camera chooser + real `image_picker` flows
- Keep local `XFile`/path for preview only; create-post payload stays `media_type` without multipart
- Preserve image/video mutual exclusion and clear/cancel behavior

**Non-Goals:**

- Real media upload, CDN, or signed URL storage
- Multi-image picker UX beyond what current backend default list implies (single pick is enough; server still may attach multiple default URLs)
- Changing Go community routes or DTO shape
- Deep-linking into system Settings beyond a toast/message on permission denial (can deep-link if already patterned elsewhere)

## Decisions

1. **Picker library: reuse existing `image_picker`**  
   Already in the Flutter workspace; avoid a second picker stack.  
   _Alt considered:_ `wechat_assets_picker` / `photo_manager` — richer UX, heavier deps and OHOS path risk; YAGNI for MVP.

2. **UX: bottom sheet source chooser, then platform UI**  
   Tap 图片 → sheet「从相册选择」「拍照」；Tap 视频 →「从相册选择」「拍摄」. Then `ImagePicker.pickImage` / `pickVideo` / `ImageSource.gallery|camera`.  
   _Alt:_ single combined media sheet — clearer separation matches 图/视频互斥.

3. **Draft state: `mediaType` + optional local preview path**  
   ViewModel holds `localPreviewPath` / `XFile?` for UI only; `createPost` still sends `media_type` only (empty `image_urls`). After success, discard local file reference.  
   _Alt:_ upload then send URL — out of scope.

4. **Permissions**  
   Rely on `image_picker` permission prompts; ensure iOS usage descriptions and Android permissions exist for the host app. On `PlatformException` / null return, toast and keep prior draft.

5. **Cross-repo apply**  
   Tasks explicitly mark Flutter paths under `my_ai_project`; Go repo change artifacts stay the contract. No Go code expected unless a docs note is desired.

## Risks / Trade-offs

- [Preview ≠ feed] Local preview then remote default URL may surprise users → Mitigation: short subtitle on publish preview「发布后使用默认示例媒体」or accept as known MVP ceiling (`ponytail`-style).
- [Simulator camera] Simulators may lack camera → Mitigation: album path remains primary; toast on camera failure.
- [OHOS / fork image_picker] App uses a git-forked picker → Mitigation: stick to APIs already used by toolkit; smoke on primary iOS/Android targets.
- [OpenSpec allowedEditRoots] Apply agents scoped to `my_go_study` may not edit Flutter → Mitigation: tasks say implementer must work in `my_ai_project` (or dual-root session).

## Migration Plan

1. Ship Flutter-only; no DB/API migration.
2. Rollback: revert publish media UI to chips; API unchanged.

## Open Questions

- Whether to show the disclaimer that published media is a default sample URL (product copy only; does not change requirements).
