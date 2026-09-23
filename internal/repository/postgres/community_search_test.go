package postgres

import "testing"

func TestEscapeILIKE(t *testing.T) {
	if got := escapeILIKE(`a%b_c\d`); got != `a\%b\_c\\d` {
		t.Fatalf("got %q", got)
	}
	if got := ilikeContains("x"); got != `%x%` {
		t.Fatalf("contains %q", got)
	}
}
