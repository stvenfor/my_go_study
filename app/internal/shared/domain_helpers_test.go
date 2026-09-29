package shared

import (
	"errors"
	"testing"
	"time"
)

func TestPageOffset(t *testing.T) {
	cases := []struct {
		name                        string
		page, size                  int
		defSize, maxSize            int
		wantPage, wantSize, wantOff int
	}{
		{"defaults", 0, 0, 10, 50, 1, 10, 0},
		{"clamp max", 2, 100, 10, 50, 2, 50, 50},
		{"normal", 3, 20, 10, 50, 3, 20, 40},
		{"neg page", -1, 5, 10, 50, 1, 5, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, s, off := PageOffset(tc.page, tc.size, tc.defSize, tc.maxSize)
			if p != tc.wantPage || s != tc.wantSize || off != tc.wantOff {
				t.Fatalf("got page=%d size=%d off=%d want %d %d %d", p, s, off, tc.wantPage, tc.wantSize, tc.wantOff)
			}
		})
	}
}

func TestValidReviewStatusFilter(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"", true},
		{"all", true},
		{"pending_review", true},
		{"approved", true},
		{"rejected", true},
		{"  approved ", true},
		{"bogus", false},
	}
	for _, tc := range cases {
		if got := ValidReviewStatusFilter(tc.raw); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.raw, got, tc.want)
		}
	}
}

func TestNormalizeImageURL(t *testing.T) {
	empty := "  "
	if NormalizeImageURL(&empty) != nil {
		t.Fatal("blank should become nil")
	}
	ok := " https://x "
	got := NormalizeImageURL(&ok)
	if got == nil || *got != "https://x" {
		t.Fatalf("got %#v", got)
	}
	if NormalizeImageURL(nil) != nil {
		t.Fatal("nil in → nil out")
	}
}

func TestParsePositiveInt64(t *testing.T) {
	sentinel := errors.New("nf")
	cases := []struct {
		raw     string
		wantID  int64
		wantErr bool
	}{
		{"42", 42, false},
		{" 7 ", 7, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"x", 0, true},
		{"", 0, true},
	}
	for _, tc := range cases {
		id, err := ParsePositiveInt64(tc.raw, sentinel)
		if tc.wantErr {
			if !errors.Is(err, sentinel) || id != 0 {
				t.Fatalf("%q: got id=%d err=%v", tc.raw, id, err)
			}
			continue
		}
		if err != nil || id != tc.wantID {
			t.Fatalf("%q: got id=%d err=%v want %d", tc.raw, id, err, tc.wantID)
		}
	}
}

func TestShanghaiDate(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	// UTC 2026-09-29 20:30 == 上海 2026-09-30 04:30 → 当天 0 点应为 09-30。
	in := time.Date(2026, 9, 29, 20, 30, 0, 0, time.UTC)
	got := ShanghaiDate(in)
	if got.Year() != 2026 || got.Month() != time.September || got.Day() != 30 {
		t.Fatalf("got %s want 2026-09-30", got.Format("2006-01-02"))
	}
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
		t.Fatalf("not midnight: %s", got)
	}
	if got.Location().String() != loc.String() {
		t.Fatalf("location %s want %s", got.Location(), loc)
	}
}
