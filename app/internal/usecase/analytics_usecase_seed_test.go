package usecase

import (
	"strings"
	"testing"
	"time"
)

func TestBuildAnalyticsSeedRecords_FunnelAndScores(t *testing.T) {
	items := buildAnalyticsSeedRecords(time.Now().UTC().Unix())
	if len(items) < 40 {
		t.Fatalf("expected rich seed count, got %d", len(items))
	}

	var hasZeroClick, hasAnomaly, hasFeatured, hasMarker bool
	for _, it := range items {
		if it.MetricPV < it.MetricUV || it.MetricUV < it.MetricClick || it.MetricClick < it.MetricConvert {
			t.Fatalf("%s funnel not monotonic: pv=%d uv=%d click=%d convert=%d",
				it.Code, it.MetricPV, it.MetricUV, it.MetricClick, it.MetricConvert)
		}
		if it.MetricBounceRate < 0 || it.MetricBounceRate > 1 {
			t.Fatalf("%s bounce out of range: %v", it.Code, it.MetricBounceRate)
		}
		if it.ScoreQuality < 0 || it.ScoreQuality > 100 || it.ScoreRisk < 0 || it.ScoreRisk > 100 {
			t.Fatalf("%s scores out of 0–100: q=%v r=%v", it.Code, it.ScoreQuality, it.ScoreRisk)
		}
		if it.MetricClick == 0 {
			hasZeroClick = true
		}
		if it.FlagAnomaly {
			hasAnomaly = true
		}
		if it.FlagFeatured {
			hasFeatured = true
		}
		if strings.Contains(it.Notes, analyticsSeedMarker) {
			hasMarker = true
		}
		if it.SourceSystem != "seed" {
			t.Fatalf("source_system want seed, got %s", it.SourceSystem)
		}
	}
	if !hasZeroClick || !hasAnomaly || !hasFeatured || !hasMarker {
		t.Fatalf("missing scenario coverage zeroClick=%v anomaly=%v featured=%v marker=%v",
			hasZeroClick, hasAnomaly, hasFeatured, hasMarker)
	}
}
