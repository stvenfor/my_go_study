-- 手动刷新 analytics_records 图表友好种子（seed_chart_v1）
-- 用法：在本地 Postgres 执行本脚本；或重启 Go API（EnsureSeedData 会自动替换旧 seed）。
-- 注意：仅删除 source_system = 'seed' 的行，手工写入的非 seed 数据保留。

BEGIN;

DELETE FROM analytics_records WHERE source_system = 'seed';

-- 最小示例 8 条（完整集由 Go EnsureSeedData 写入约 48 条）。
-- 漏斗约束：metric_pv >= metric_uv >= metric_click >= metric_convert
-- bounce 0–1；score_* 0–100

INSERT INTO analytics_records (
  code, title, subtitle, category, sub_category, status, priority,
  region, channel, owner_name, owner_team, source_system,
  metric_pv, metric_uv, metric_click, metric_convert,
  metric_revenue, metric_cost, metric_roi, metric_bounce_rate, metric_avg_duration_sec,
  score_quality, score_risk, tag_primary, tag_secondary,
  flag_featured, flag_anomaly, notes,
  observed_at, window_start, window_end, published_at, created_at, updated_at
) VALUES
(
  'AN-0001', '双十一主会场漏斗', '获客 · App 大促峰值', '获客', '大促', 'active', 5,
  '华东', 'App', '分析师01', '增长组-1', 'seed',
  12000, 5040, 1915, 230,
  12240.00, 2112.00, 5.80, 0.28, 90,
  88.0, 22.0, '大促', '高转化',
  TRUE, FALSE, 'seed_chart_v1 | 漏斗健康正向样例 | SQL 手工种子',
  EXTRACT(EPOCH FROM NOW())::bigint - 3600,
  EXTRACT(EPOCH FROM NOW())::bigint - 7*86400,
  EXTRACT(EPOCH FROM NOW())::bigint - 3600,
  EXTRACT(EPOCH FROM NOW())::bigint - 7200,
  EXTRACT(EPOCH FROM NOW())::bigint,
  EXTRACT(EPOCH FROM NOW())::bigint
),
(
  'AN-0002', '异常：投放点击虚高', '投放 · 合作方流量注水嫌疑', '投放', '效果广告', 'paused', 5,
  '华南', '合作方', '分析师02', '增长组-2', 'seed',
  13800, 7590, 5465, 82,
  993.60, 5299.20, 0.19, 0.71, 40,
  41.0, 86.0, '异常', '低转化',
  FALSE, TRUE, 'seed_chart_v1 | 异常红条 + 低转化环 | SQL 手工种子',
  EXTRACT(EPOCH FROM NOW())::bigint - 7200,
  EXTRACT(EPOCH FROM NOW())::bigint - 7*86400,
  EXTRACT(EPOCH FROM NOW())::bigint - 7200,
  EXTRACT(EPOCH FROM NOW())::bigint - 10800,
  EXTRACT(EPOCH FROM NOW())::bigint,
  EXTRACT(EPOCH FROM NOW())::bigint
),
(
  'AN-0003', '精选：小程序裂变', '转化 · 小程序分享链路', '转化', '裂变', 'active', 4,
  '华北', '小程序', '分析师03', '增长组-3', 'seed',
  15600, 7488, 3370, 741,
  11606.40, 1747.20, 6.64, 0.19, 110,
  92.0, 14.0, '精选', '裂变',
  TRUE, FALSE, 'seed_chart_v1 | 精选琥珀条 | SQL 手工种子',
  EXTRACT(EPOCH FROM NOW())::bigint - 10800,
  EXTRACT(EPOCH FROM NOW())::bigint - 7*86400,
  EXTRACT(EPOCH FROM NOW())::bigint - 10800,
  EXTRACT(EPOCH FROM NOW())::bigint - 14400,
  EXTRACT(EPOCH FROM NOW())::bigint,
  EXTRACT(EPOCH FROM NOW())::bigint
),
(
  'AN-0004', '零点击冷启动页', '获客 · H5 新落地页尚无互动', '获客', '落地页', 'draft', 2,
  '西南', 'H5', '分析师04', '增长组-1', 'seed',
  17400, 5394, 0, 0,
  0.00, 1113.60, 0.00, 0.62, 28,
  55.0, 35.0, '冷启动', '无点击',
  FALSE, FALSE, 'seed_chart_v1 | 转化环应显示 — | SQL 手工种子',
  EXTRACT(EPOCH FROM NOW())::bigint - 14400,
  EXTRACT(EPOCH FROM NOW())::bigint - 7*86400,
  EXTRACT(EPOCH FROM NOW())::bigint - 14400,
  EXTRACT(EPOCH FROM NOW())::bigint - 18000,
  EXTRACT(EPOCH FROM NOW())::bigint,
  EXTRACT(EPOCH FROM NOW())::bigint
);

COMMIT;
