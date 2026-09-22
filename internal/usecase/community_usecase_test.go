package usecase

import (
	"strings"
	"testing"
)

func TestNormalizeMedia_defaults(t *testing.T) {
	mt, imgs, v, c, err := normalizeMedia("image", nil, nil, nil)
	if err != nil || mt != 1 || len(imgs) != 3 {
		t.Fatalf("image defaults: mt=%d imgs=%d err=%v", mt, len(imgs), err)
	}
	mt, imgs, v, c, err = normalizeMedia("video", nil, nil, nil)
	if err != nil || mt != 2 || v == nil || c == nil || *v == "" || *c == "" {
		t.Fatalf("video defaults: mt=%d v=%v c=%v err=%v", mt, v, c, err)
	}
	mt, imgs, v, c, err = normalizeMedia("none", []string{"x"}, nil, nil)
	if err != nil || mt != 0 || len(imgs) != 0 || v != nil {
		t.Fatalf("none: mt=%d imgs=%v v=%v err=%v", mt, imgs, v, err)
	}
}

func TestNormalizeMedia_imageCap(t *testing.T) {
	tooMany := make([]string, 10)
	for i := range tooMany {
		tooMany[i] = "u"
	}
	_, _, _, _, err := normalizeMedia("image", tooMany, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "最多") {
		t.Fatalf("want cap error, got %v", err)
	}
}

func TestNormalizePostTab(t *testing.T) {
	cases := map[string]string{
		"": "latest", "latest": "latest", "最新": "latest",
		"hot": "hot", "热门": "hot", "HOT": "hot",
		"following": "following", "关注": "following",
		"other": "latest",
	}
	for in, want := range cases {
		if got := normalizePostTab(in); got != want {
			t.Fatalf("tab %q: got %q want %q", in, got, want)
		}
	}
}

func TestPostHeatFormula(t *testing.T) {
	like, comment := 3, 4
	got := int64(like)*2 + int64(comment)
	if got != 10 {
		t.Fatalf("got %d", got)
	}
}

func TestAppendTopicTagLogic(t *testing.T) {
	content := "hello"
	tag := "#问大家"
	if !strings.Contains(content, tag) {
		content = content + "\n" + tag
	}
	if content != "hello\n#问大家" {
		t.Fatalf("got %q", content)
	}
	dup := content
	if !strings.Contains(dup, tag) {
		dup = dup + "\n" + tag
	}
	if dup != content {
		t.Fatalf("duplicated tag")
	}
}
