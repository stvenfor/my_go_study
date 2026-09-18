package usecase

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
)

// analyticsSeedMarker 写入 notes，用于识别「图表友好」种子版本；缺失则刷新 seed。
const analyticsSeedMarker = "seed_chart_v1"

// AnalyticsUsecase 数据分析业务。
type AnalyticsUsecase struct {
	repo domainrepo.AnalyticsRepository
}

// NewAnalyticsUsecase 创建用例。
func NewAnalyticsUsecase(repo domainrepo.AnalyticsRepository) *AnalyticsUsecase {
	return &AnalyticsUsecase{repo: repo}
}

// ListPage 分页列表（page 从 1 起）。
func (u *AnalyticsUsecase) ListPage(ctx context.Context, page, pageSize int) ([]entity.AnalyticsRecord, int64, error) {
	return u.repo.ListPage(ctx, entity.AnalyticsListFilter{Page: page, PageSize: pageSize})
}

// Get 详情。
func (u *AnalyticsUsecase) Get(ctx context.Context, id int64) (*entity.AnalyticsRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的记录 ID")
	}
	return u.repo.GetByID(ctx, id)
}

// EnsureSeedData 若尚无 chart-v1 种子则清空 source_system=seed 并写入丰富演示数据。
func (u *AnalyticsUsecase) EnsureSeedData(ctx context.Context) error {
	marked, err := u.repo.CountNotesContaining(ctx, analyticsSeedMarker)
	if err != nil {
		return err
	}
	if marked > 0 {
		return nil
	}
	if err := u.repo.DeleteBySourceSystem(ctx, "seed"); err != nil {
		return err
	}
	return u.repo.CreateBatch(ctx, buildAnalyticsSeedRecords(time.Now().UTC().Unix()))
}

type seedScenario struct {
	title    string
	subtitle string
	category string
	subCat   string
	status   string
	region   string
	channel  string
	// Funnel ratios relative to PV (must be decreasing).
	uvRatio      float64
	clickRatio   float64 // of UV
	convertRatio float64 // of click
	revMul       float64
	costMul      float64
	bounce       float64 // 0–1
	quality      float64 // 0–100
	risk         float64 // 0–100
	featured     bool
	anomaly      bool
	priority     int32
	tagA         string
	tagB         string
	notesExtra   string
}

func buildAnalyticsSeedRecords(now int64) []entity.AnalyticsRecord {
	scenarios := []seedScenario{
		{
			title: "双十一主会场漏斗", subtitle: "获客 · App 大促峰值", category: "获客", subCat: "大促",
			status: "active", region: "华东", channel: "App",
			uvRatio: 0.42, clickRatio: 0.38, convertRatio: 0.12,
			revMul: 8.5, costMul: 2.2, bounce: 0.28, quality: 88, risk: 22,
			featured: true, anomaly: false, priority: 5, tagA: "大促", tagB: "高转化",
			notesExtra: "漏斗健康，适合详情页正向样例。",
		},
		{
			title: "异常：投放点击虚高", subtitle: "投放 · 合作方流量注水嫌疑", category: "投放", subCat: "效果广告",
			status: "paused", region: "华南", channel: "合作方",
			uvRatio: 0.55, clickRatio: 0.72, convertRatio: 0.015,
			revMul: 0.6, costMul: 4.8, bounce: 0.71, quality: 41, risk: 86,
			featured: false, anomaly: true, priority: 5, tagA: "异常", tagB: "低转化",
			notesExtra: "点击远高于转化，ROI 差，列表应出红条。",
		},
		{
			title: "精选：小程序裂变", subtitle: "转化 · 小程序分享链路", category: "转化", subCat: "裂变",
			status: "active", region: "华北", channel: "小程序",
			uvRatio: 0.48, clickRatio: 0.45, convertRatio: 0.22,
			revMul: 6.2, costMul: 1.4, bounce: 0.19, quality: 92, risk: 14,
			featured: true, anomaly: false, priority: 4, tagA: "精选", tagB: "裂变",
			notesExtra: "高转化环 + 精选琥珀条。",
		},
		{
			title: "零点击冷启动页", subtitle: "获客 · H5 新落地页尚无互动", category: "获客", subCat: "落地页",
			status: "draft", region: "西南", channel: "H5",
			uvRatio: 0.31, clickRatio: 0, convertRatio: 0,
			revMul: 0, costMul: 0.8, bounce: 0.62, quality: 55, risk: 35,
			featured: false, anomaly: false, priority: 2, tagA: "冷启动", tagB: "无点击",
			notesExtra: "点击为 0，转化环应显示「—」。",
		},
		{
			title: "高客单线下门店", subtitle: "营收 · 线下体验成交", category: "营收", subCat: "门店",
			status: "active", region: "华东", channel: "线下",
			uvRatio: 0.22, clickRatio: 0.55, convertRatio: 0.35,
			revMul: 18.0, costMul: 5.5, bounce: 0.12, quality: 85, risk: 28,
			featured: true, anomaly: false, priority: 4, tagA: "高客单", tagB: "线下",
			notesExtra: "收支对比柱收入显著高于成本。",
		},
		{
			title: "成本倒挂活动", subtitle: "投放 · 补贴过重", category: "投放", subCat: "补贴",
			status: "paused", region: "西北", channel: "App",
			uvRatio: 0.40, clickRatio: 0.33, convertRatio: 0.08,
			revMul: 1.2, costMul: 7.5, bounce: 0.48, quality: 52, risk: 74,
			featured: false, anomaly: true, priority: 3, tagA: "亏损", tagB: "补贴",
			notesExtra: "成本高于收入，雷达风险偏高。",
		},
		{
			title: "留存召回短信", subtitle: "留存 · 沉默用户召回", category: "留存", subCat: "召回",
			status: "active", region: "华南", channel: "H5",
			uvRatio: 0.36, clickRatio: 0.28, convertRatio: 0.09,
			revMul: 3.1, costMul: 1.1, bounce: 0.41, quality: 73, risk: 31,
			featured: false, anomaly: false, priority: 3, tagA: "召回", tagB: "短信",
		},
		{
			title: "华北品牌搜索", subtitle: "获客 · 品牌词 SEM", category: "获客", subCat: "SEM",
			status: "active", region: "华北", channel: "H5",
			uvRatio: 0.50, clickRatio: 0.40, convertRatio: 0.11,
			revMul: 4.4, costMul: 2.0, bounce: 0.33, quality: 79, risk: 25,
			featured: false, anomaly: false, priority: 3, tagA: "SEM", tagB: "品牌",
		},
	}

	// Procedural enrichment for volume + chart variety.
	categories := []string{"获客", "转化", "留存", "营收", "投放"}
	regions := []string{"华东", "华南", "华北", "西南", "西北"}
	channels := []string{"App", "小程序", "H5", "线下", "合作方"}
	statuses := []string{"active", "paused", "draft", "archived"}
	titles := []string{
		"周末晚高峰转化", "新品首发预热", "会员日复购", "短视频引流落地",
		"老客专属券", "异地扩城试点", "内容种草合集", "客服转介绍",
	}

	const procedural = 40
	items := make([]entity.AnalyticsRecord, 0, len(scenarios)+procedural)

	for i, sc := range scenarios {
		pv := int64(12000 + i*1800)
		items = append(items, makeSeedRecord(now, i+1, pv, sc))
	}

	for j := 0; j < procedural; j++ {
		i := len(scenarios) + j + 1
		pv := int64(5000 + j*317)
		uvR := 0.28 + float64(j%6)*0.04
		clickR := 0.18 + float64(j%5)*0.05
		convR := 0.03 + float64(j%9)*0.025
		if j%13 == 0 {
			clickR = 0 // occasional zero-click
			convR = 0
		}
		bounce := round2(0.15 + float64(j%12)*0.04)
		quality := round1(48 + float64(j%50))
		risk := round1(8 + float64(j%70))
		revMul := 2.0 + float64(j%10)*0.7
		costMul := 1.0 + float64(j%8)*0.55
		sc := seedScenario{
			title:        titles[j%len(titles)] + fmt.Sprintf(" #%d", j+1),
			subtitle:     fmt.Sprintf("%s · %s 周报", categories[j%len(categories)], channels[j%len(channels)]),
			category:     categories[j%len(categories)],
			subCat:       fmt.Sprintf("子类-%d", (j%5)+1),
			status:       statuses[j%len(statuses)],
			region:       regions[j%len(regions)],
			channel:      channels[j%len(channels)],
			uvRatio:      uvR,
			clickRatio:   clickR,
			convertRatio: convR,
			revMul:       revMul,
			costMul:      costMul,
			bounce:       bounce,
			quality:      quality,
			risk:         risk,
			featured:     j%9 == 0,
			anomaly:      j%11 == 0,
			priority:     int32((j % 5) + 1),
			tagA:         fmt.Sprintf("tag-a-%d", j%4),
			tagB:         fmt.Sprintf("tag-b-%d", j%6),
			notesExtra:   "程序化种子，保证漏斗递减与评分 0–100。",
		}
		items = append(items, makeSeedRecord(now, i, pv, sc))
	}
	return items
}

func makeSeedRecord(now int64, index int, pv int64, sc seedScenario) entity.AnalyticsRecord {
	uv := int64(math.Round(float64(pv) * clamp01(sc.uvRatio)))
	if uv > pv {
		uv = pv
	}
	click := int64(math.Round(float64(uv) * clamp01(sc.clickRatio)))
	if click > uv {
		click = uv
	}
	convert := int64(math.Round(float64(click) * clamp01(sc.convertRatio)))
	if convert > click {
		convert = click
	}

	unit := float64(pv) / 1000.0
	rev := round2(unit * sc.revMul * 120)
	cost := round2(unit * sc.costMul * 80)
	roi := 0.0
	if cost > 0 {
		roi = round2(rev / cost)
	}

	observed := now - int64(index*3600)
	notes := fmt.Sprintf("%s | %s | %s", analyticsSeedMarker, sc.notesExtra,
		"供 Flutter 列表转化环/迷你柱与详情漏斗·收支·雷达联调。")

	return entity.AnalyticsRecord{
		Code:                 fmt.Sprintf("AN-%04d", index),
		Title:                sc.title,
		Subtitle:             sc.subtitle,
		Category:             sc.category,
		SubCategory:          sc.subCat,
		Status:               sc.status,
		Priority:             sc.priority,
		Region:               sc.region,
		Channel:              sc.channel,
		OwnerName:            fmt.Sprintf("分析师%02d", (index%8)+1),
		OwnerTeam:            fmt.Sprintf("增长组-%d", (index%3)+1),
		SourceSystem:         "seed",
		MetricPV:             pv,
		MetricUV:             uv,
		MetricClick:          click,
		MetricConvert:        convert,
		MetricRevenue:        rev,
		MetricCost:           cost,
		MetricROI:            roi,
		MetricBounceRate:     clamp01(sc.bounce),
		MetricAvgDurationSec: int32(25 + (index % 120)),
		ScoreQuality:         clamp100(sc.quality),
		ScoreRisk:            clamp100(sc.risk),
		TagPrimary:           sc.tagA,
		TagSecondary:         sc.tagB,
		FlagFeatured:         sc.featured,
		FlagAnomaly:          sc.anomaly,
		Notes:                notes,
		ObservedAt:           observed,
		WindowStart:          observed - 7*24*3600,
		WindowEnd:            observed,
		PublishedAt:          observed - 3600,
		CreatedAt:            now - int64(index*60),
		UpdatedAt:            now - int64(index*30),
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clamp100(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return round1(v)
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round1(v float64) float64 { return math.Round(v*10) / 10 }
