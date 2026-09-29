package usecase

import (
	"context"
	"errors"
	"testing"
)

func TestPageOffset(t *testing.T) {
	cases := []struct {
		name            string
		page, size      int
		defSize, maxSize int
		wantPage, wantSize, wantOff int
	}{
		{"defaults", 0, 0, 10, 50, 1, 10, 0},
		{"clamp max", 2, 100, 10, 50, 2, 50, 50},
		{"normal", 3, 20, 10, 50, 3, 20, 40},
		{"neg page", -1, 5, 10, 50, 1, 5, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, s, off := pageOffset(tc.page, tc.size, tc.defSize, tc.maxSize)
			if p != tc.wantPage || s != tc.wantSize || off != tc.wantOff {
				t.Fatalf("got page=%d size=%d off=%d want %d %d %d", p, s, off, tc.wantPage, tc.wantSize, tc.wantOff)
			}
		})
	}
}

func TestRequireAccessCurrentStore_nilAccess(t *testing.T) {
	noStore := errors.New("no store")
	id, err := requireAccessCurrentStore(nil, context.Background(), "u1", noStore)
	if id != 0 || !errors.Is(err, noStore) {
		t.Fatalf("got id=%d err=%v", id, err)
	}
}

func TestAccessStoreLabels_nilAccess(t *testing.T) {
	name, label := accessStoreLabels(context.Background(), nil, "u1", 1)
	if name != "" || label != "" {
		t.Fatalf("got %q %q", name, label)
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
		if got := validReviewStatusFilter(tc.raw); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.raw, got, tc.want)
		}
	}
}

func TestNormalizeImageURL(t *testing.T) {
	empty := "  "
	if normalizeImageURL(&empty) != nil {
		t.Fatal("blank should become nil")
	}
	ok := " https://x "
	got := normalizeImageURL(&ok)
	if got == nil || *got != "https://x" {
		t.Fatalf("got %#v", got)
	}
	if normalizeImageURL(nil) != nil {
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
		id, err := parsePositiveInt64(tc.raw, sentinel)
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
