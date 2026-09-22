# 03 — 新建 + 客户选择（API + Flutter 提交）

**What to build:** 销售能选当前店客户、填必填车况与金额、选类型后提交；新单默认待审核并出现在列表。

**Blocked by:** 02 — 详情（API + Flutter 详情）

**Status:** done

- [x] `GET /api/v1/used-car-orders/customers`（当前店分页，可选 q）
- [x] `POST /api/v1/used-car-orders`（校验必填；默认 status=待审核；可选 image_url）
- [x] Usecase 单测：缺字段/非法 kind/他店客户拒绝
- [x] Flutter 新建页（类型、客户、车况、金额、可选本地图）可提交成功并返回列表
