# ADR 0016 — 新车跟进：店管全店可见

## Status
Accepted (2026-09-23)

## Context
C1 仅本人可见。业务需要店管看当前店全部跟进档案，且勿用门店职务当权限。

## Decision
- 判定：`AccessUsecase.can(..., PermRoleAssignStore, currentStore)`，与首页待办店管一致
- 同接口按角色扩口径：店管 `owner_user_id` 过滤为空；销售仍锁本人
- 店管可 PATCH / 写流水本店任意档案；不做转交字段
- 响应带 `owner_user_id` + `owner_display_name`

## Consequences
- Flutter 店管列表可标销售名；非店管无感
- 转交 / 跨店另 Slice
