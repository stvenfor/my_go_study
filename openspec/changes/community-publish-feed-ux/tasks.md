## 1. Convention dialog UI

- [x] 1.1 Rebuild `CommunityConventionDialog` to reference layout (title, body rules, link affordance, primary「我知道了」button, dim barrier)
- [x] 1.2 Keep once-per-calendar-day SharedPreferences behavior; wire link to stub (toast or placeholder page)

## 2. Feed image grid fill

- [x] 2.1 Fix nine-grid cell so each tile is square and image uses cover fill with no large empty bands
- [x] 2.2 Confirm video posts still show a single video card and never an image grid

## 3. Topic association page UI

- [x] 3.1 Rebuild search bar chrome to match「搜索话题」reference
- [x] 3.2 Add highlighted「问大家」card (from `is_ask_everyone`) with description copy
- [x] 3.3 Restyle topic rows: `#name` + right-aligned heat; keep select → return topic

## 4. Publish multi-select and switch clear

- [x] 4.1 Add multi-image pick (prefer `pickMultiImage`) into draft list; cap at 9 with user feedback
- [x] 4.2 Support continue-add: reopen picker and append until 9
- [x] 4.3 On image↔video switch, clear the other side’s paths, previews, and video cover

## 5. Video cover and previews

- [x] 5.1 Allow setting a local video cover after video pick; show cover on publish draft
- [x] 5.2 Tap video draft → preview playback; tap image thumbnails → enlarge preview (`ImagePreviewPage`)

## 6. Verification

- [ ] 6.1 Visual check: convention + topic page vs reference screenshots
- [ ] 6.2 Visual check: feed image cells fully filled; video-only posts
- [ ] 6.3 Flow: select 4 images → add more → ≤9; switch to video clears images and vice versa
- [x] 6.4 `dart analyze` on `module_community` (and touched utils) clean of new errors
