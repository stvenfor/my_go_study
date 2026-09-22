## Context

See `proposal.md` — Why. Related in-progress change `community-publish-media-picker` landed single-pick album/camera; feed already uses a 3×3 grid but cells may letterbox; convention uses a plain `AlertDialog`; topic page is a basic `ListTile` list. Publish still uses `image_picker` single pick via `ImagePickerUtils`. Backend publish remains `media_type` + default remote URLs (`docs/community-posts-api.md`).

## Goals / Non-Goals

**Goals:**

- Match reference UI for convention modal and topic association
- Feed image tiles fill square cells (`BoxFit.cover`, no wasted whitespace in-cell)
- Multi-image draft (≤9) with continue-add; hard clear on image↔video switch
- Local video cover + preview; image thumbnail → existing preview page

**Non-Goals:**

- Real media upload / CDN
- Changing Go community API contracts (unless a tiny optional `image_urls` count later)
- Reworking feed text expand/collapse (already 3-line) except if layout regression appears
- Pixel-perfect Alipay asset cloning beyond structure/spacing/colors reasonable for this app

## Decisions

1. **Convention UI**  
   Custom centered dialog (dim barrier, rounded card, blue primary button, link style text) — not stock `AlertDialog`. Keep SharedPreferences day key from existing dialog.  
   _Alt:_ full-screen page — heavier than reference modal.

2. **Topic page layout**  
   Rebuild `TopicSelectPage`: search bar, ask-everyone banner card from `is_ask_everyone` topic, list rows `#name` + `heatLabel`. Reuse API `fetchTopics` / `searchTopics`.  
   _Alt:_ keep ListTile — rejected by product feedback.

3. **Feed grid fill**  
   Ensure each cell is square + `CacheImageUtils.network(..., fit: BoxFit.cover)` with `SizedBox.expand` / no unbounded height letterboxing. Audit `CacheImageUtils` if it forces intrinsic aspect.  
   _Alt:_ change nine-grid padding algorithm — secondary to fit.

4. **Multi-select**  
   Prefer `ImagePicker.pickMultiImage` (+ merge into existing draft list, cap 9). Camera remains single-add per shot. Continue-add = reopen picker and append unique paths.  
   _Alt:_ `wechat_assets_picker` — better UX, heavier; only if `pickMultiImage` cannot meet continue-add on target platforms.

5. **Switch clear**  
   Centralize in `PublishViewModel`: entering video flow clears `imagePaths`; entering image flow clears `videoPath` + `coverPath`.  
   Already partially intended; make explicit and UI-bound.

6. **Video cover**  
   Optional second pick (gallery/camera image) stored as local cover path for preview only; publish still sends `media_type=video` without uploading cover. Preview uses existing `VideoPlayPage` / cover thumbnail tap.  
   Image enlarge: reuse `ImagePreviewPage` with draft path list.

## Risks / Trade-offs

- [pickMultiImage platform gaps] → Mitigation: feature-detect / fallback sequential single pick with append; document on OHOS if fork differs.
- [CacheImageUtils letterboxing root cause] → Mitigation: inspect util; wrap with `FittedBox`/`cover` at grid cell if needed.
- [Overlap with media-picker change] → Mitigation: finish or supersede remaining media-picker tasks inside this change’s tasks; avoid duplicate PRs.
- [Preview ≠ feed defaults] → Known MVP; keep disclaimer copy.

## Migration Plan

1. Flutter-only UI; no migration.
2. Rollback by reverting community publish/feed widgets.

## Open Questions

- Whether「完整社区公约」link opens a WebView placeholder URL or a local markdown page (default: toast or static in-app page stub).
