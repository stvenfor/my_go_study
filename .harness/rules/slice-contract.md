# Rule — Slice contract

唯一执行单位：`plans/slices/<id>.md`（由 planner 落盘；未建 `plans/` 时先补目录与模板）

实现类必填：`本轮 ONLY` · `不做` · `验收` · `文件白名单` · `验证命令` · `证据`  
审计类（ONLY 含只读/audit/不改代码/差距表）：白名单/验证命令可放宽

子列表项必须缩进。模板：`plans/slices/_template.md`（可参考 `~/.cursor/skills/migration-os-harness/templates.md`）  
机检：`make agent-pre` → `check-brief`（若已落地）
