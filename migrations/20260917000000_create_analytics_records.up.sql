-- analytics_records: 数据分析宽表（时间均为 Unix 秒 bigint）
CREATE TABLE IF NOT EXISTS analytics_records (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    title VARCHAR(256) NOT NULL,
    subtitle VARCHAR(256) NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT '',
    sub_category VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT '',
    priority INTEGER NOT NULL DEFAULT 0,
    region VARCHAR(64) NOT NULL DEFAULT '',
    channel VARCHAR(64) NOT NULL DEFAULT '',
    owner_name VARCHAR(128) NOT NULL DEFAULT '',
    owner_team VARCHAR(128) NOT NULL DEFAULT '',
    source_system VARCHAR(64) NOT NULL DEFAULT '',
    metric_pv BIGINT NOT NULL DEFAULT 0,
    metric_uv BIGINT NOT NULL DEFAULT 0,
    metric_click BIGINT NOT NULL DEFAULT 0,
    metric_convert BIGINT NOT NULL DEFAULT 0,
    metric_revenue DOUBLE PRECISION NOT NULL DEFAULT 0,
    metric_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    metric_roi DOUBLE PRECISION NOT NULL DEFAULT 0,
    metric_bounce_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    metric_avg_duration_sec INTEGER NOT NULL DEFAULT 0,
    score_quality DOUBLE PRECISION NOT NULL DEFAULT 0,
    score_risk DOUBLE PRECISION NOT NULL DEFAULT 0,
    tag_primary VARCHAR(64) NOT NULL DEFAULT '',
    tag_secondary VARCHAR(64) NOT NULL DEFAULT '',
    flag_featured BOOLEAN NOT NULL DEFAULT FALSE,
    flag_anomaly BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT NOT NULL DEFAULT '',
    observed_at BIGINT NOT NULL DEFAULT 0,
    window_start BIGINT NOT NULL DEFAULT 0,
    window_end BIGINT NOT NULL DEFAULT 0,
    published_at BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_analytics_records_code ON analytics_records (code);
CREATE INDEX IF NOT EXISTS idx_analytics_records_category ON analytics_records (category);
CREATE INDEX IF NOT EXISTS idx_analytics_records_status ON analytics_records (status);
CREATE INDEX IF NOT EXISTS idx_analytics_records_observed_at ON analytics_records (observed_at DESC);
