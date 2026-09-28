package metrics

import "testing"

func TestStatusClass(t *testing.T) {
	cases := map[int]string{100: "1xx", 200: "2xx", 301: "3xx", 404: "4xx", 500: "5xx", 503: "5xx"}
	for code, want := range cases {
		if got := StatusClass(code); got != want {
			t.Fatalf("StatusClass(%d)=%q want %q", code, got, want)
		}
	}
}
