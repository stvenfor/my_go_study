// analytics_record.go 数据分析宽表领域实体（时间均为 Unix 秒）。
package entity

const AnalyticsRecordsTable = "analytics_records"

// AnalyticsRecord 数据分析记录（≥30 字段，供列表摘要 + 详情全量展示）。
type AnalyticsRecord struct {
	ID                   int64   `json:"id" gorm:"primaryKey"`
	Code                 string  `json:"code" gorm:"size:64;not null;index"`
	Title                string  `json:"title" gorm:"size:256;not null"`
	Subtitle             string  `json:"subtitle" gorm:"size:256"`
	Category             string  `json:"category" gorm:"size:64;index"`
	SubCategory          string  `json:"sub_category" gorm:"size:64"`
	Status               string  `json:"status" gorm:"size:32;index"`
	Priority             int32   `json:"priority" gorm:"not null;default:0"`
	Region               string  `json:"region" gorm:"size:64"`
	Channel              string  `json:"channel" gorm:"size:64"`
	OwnerName            string  `json:"owner_name" gorm:"size:128"`
	OwnerTeam            string  `json:"owner_team" gorm:"size:128"`
	SourceSystem         string  `json:"source_system" gorm:"size:64"`
	MetricPV             int64   `json:"metric_pv" gorm:"not null;default:0"`
	MetricUV             int64   `json:"metric_uv" gorm:"not null;default:0"`
	MetricClick          int64   `json:"metric_click" gorm:"not null;default:0"`
	MetricConvert        int64   `json:"metric_convert" gorm:"not null;default:0"`
	MetricRevenue        float64 `json:"metric_revenue" gorm:"not null;default:0"`
	MetricCost           float64 `json:"metric_cost" gorm:"not null;default:0"`
	MetricROI            float64 `json:"metric_roi" gorm:"not null;default:0"`
	MetricBounceRate     float64 `json:"metric_bounce_rate" gorm:"not null;default:0"`
	MetricAvgDurationSec int32   `json:"metric_avg_duration_sec" gorm:"not null;default:0"`
	ScoreQuality         float64 `json:"score_quality" gorm:"not null;default:0"`
	ScoreRisk            float64 `json:"score_risk" gorm:"not null;default:0"`
	TagPrimary           string  `json:"tag_primary" gorm:"size:64"`
	TagSecondary         string  `json:"tag_secondary" gorm:"size:64"`
	FlagFeatured         bool    `json:"flag_featured" gorm:"not null;default:false"`
	FlagAnomaly          bool    `json:"flag_anomaly" gorm:"not null;default:false"`
	Notes                string  `json:"notes" gorm:"type:text"`
	ObservedAt           int64   `json:"observed_at" gorm:"not null;index"`
	WindowStart          int64   `json:"window_start" gorm:"not null"`
	WindowEnd            int64   `json:"window_end" gorm:"not null"`
	PublishedAt          int64   `json:"published_at" gorm:"not null"`
	CreatedAt            int64   `json:"created_at" gorm:"not null"`
	UpdatedAt            int64   `json:"updated_at" gorm:"not null"`
}

func (AnalyticsRecord) TableName() string { return AnalyticsRecordsTable }

// AnalyticsListFilter 分页列表过滤。
type AnalyticsListFilter struct {
	Page     int
	PageSize int
}
