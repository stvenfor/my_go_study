## 1. Dependencies and permissions

- [x] 1.1 Confirm `module_community` can use `image_picker` (direct dep or via `module_utils`); add dependency in `my_ai_project/features/community/pubspec.yaml` if missing
- [x] 1.2 Verify host app iOS Info.plist photo/camera usage strings and Android manifest permissions cover gallery + camera; add only if absent

## 2. Publish draft state

- [x] 2.1 Extend `PublishViewModel` with local preview file reference, clear-media action, and mutual exclusion when switching image↔video
- [x] 2.2 Keep `createPost` payload on `media_type` only (no multipart / local path in JSON)

## 3. Source chooser and picker

- [x] 3.1 On image control: show bottom sheet with album + camera; invoke `pickImage` with the chosen source
- [x] 3.2 On video control: show bottom sheet with album + camera; invoke `pickVideo` with the chosen source
- [x] 3.3 Handle cancel/null result and permission errors with toast; leave draft unchanged on cancel

## 4. Publish UI preview

- [x] 4.1 Show local image thumbnail when an image is selected; show video selected indicator/placeholder when a video is selected
- [x] 4.2 Provide clear/remove control to reset to no media
- [x] 4.3 Optional short copy that published feed will use default sample media (product-facing only)

## 5. Verification

- [ ] 5.1 Manual: album image → preview → publish → feed shows default remote image URLs
- [ ] 5.2 Manual: camera or album video → preview/indicator → publish → feed shows default remote video
- [ ] 5.3 Manual: cancel picker and permission-denied paths do not crash; clear media works
- [x] 5.4 `dart analyze` on `module_community` clean of new errors
