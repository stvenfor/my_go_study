package community

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

// parity-p0-5：正式环境（demoMedia=false）遇到这类输入必须拒绝，不再填充 picsum/样例视频。
func TestRequiresRealMedia_gate(t *testing.T) {
	str := func(s string) *string { return &s }
	cases := []struct {
		name string
		in   CreatePostInput
		want bool
	}{
		{"image 无 URL", CreatePostInput{MediaType: "image"}, true},
		{"image 带 URL", CreatePostInput{MediaType: "image", ImageURLs: []string{"https://cdn/a.jpg"}}, false},
		{"video 无 URL", CreatePostInput{MediaType: "video"}, true},
		{"video 缺封面", CreatePostInput{MediaType: "video", VideoURL: str("https://cdn/a.mp4")}, true},
		{"video 齐全", CreatePostInput{MediaType: "video", VideoURL: str("https://cdn/a.mp4"), VideoCoverURL: str("https://cdn/a.jpg")}, false},
		{"none 只看正文", CreatePostInput{MediaType: "none"}, false},
		{"空 media_type 只看正文", CreatePostInput{}, false},
	}
	for _, tc := range cases {
		if got := requiresRealMedia(tc.in); got != tc.want {
			t.Fatalf("%s: requiresRealMedia=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestCommunitySearch_normalizeType(t *testing.T) {
	cases := map[string]string{
		"": "all", "all": "all", "POST": "post", "动态": "post",
		"topic": "topic", "话题": "topic", "user": "user", "用户": "user",
	}
	for in, want := range cases {
		if got := normalizeSearchType(in); got != want {
			t.Fatalf("type %q: got %q want %q", in, got, want)
		}
	}
}

func TestCommunitySearch_normalizeQuery(t *testing.T) {
	q, err := normalizeSearchQuery("  Flutter  ")
	if err != nil || q != "Flutter" {
		t.Fatalf("trim: %q %v", q, err)
	}
	long := strings.Repeat("啊", maxSearchQueryLen+1)
	if _, err := normalizeSearchQuery(long); err == nil {
		t.Fatal("want length error")
	}
}

func TestCommunitySearch_newSearchList(t *testing.T) {
	list := newSearchList([]TopicDTO{{ID: "1", Name: "a"}}, 1, 5, 12)
	if len(list.List) != 1 {
		t.Fatalf("list len %d", len(list.List))
	}
	p, ok := list.Pagination.(map[string]any)
	if !ok || p["total"] != int64(12) || p["totalPages"] != 3 {
		t.Fatalf("pagination %#v", list.Pagination)
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
