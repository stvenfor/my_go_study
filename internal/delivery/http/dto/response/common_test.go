package response

import (
	"math"
	"testing"
)

func TestNewPagination(t *testing.T) {
	p := NewPagination(1, 20, 100)
	if p.TotalPages != 5 {
		t.Fatalf("expected totalPages=5, got %d", p.TotalPages)
	}

	p2 := NewPagination(2, 20, 55)
	if p2.TotalPages != 3 {
		t.Fatalf("expected totalPages=3, got %d", p2.TotalPages)
	}

	p3 := NewPagination(1, 20, 0)
	if p3.TotalPages != 0 {
		t.Fatalf("expected totalPages=0 for empty total, got %d", p3.TotalPages)
	}
}

func TestPageQueryOffset(t *testing.T) {
	q := PageQuery{Page: 3, Size: 10}
	if q.Offset() != 20 {
		t.Fatalf("expected offset 20, got %d", q.Offset())
	}
}

func TestPaginationTotalPagesMath(t *testing.T) {
	total := int64(55)
	size := 20
	want := int(math.Ceil(float64(total) / float64(size)))
	if NewPagination(1, size, total).TotalPages != want {
		t.Fatal("pagination math mismatch")
	}
}
