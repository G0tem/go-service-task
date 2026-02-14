package tests

import (
	"testing"

	"github.com/G0tem/go-service-task/internal"
)

func TestPaginationTotalPages(t *testing.T) {
	tests := []struct {
		total    int64
		pageSize int
		want     int
	}{
		{0, 20, 0},
		{1, 20, 1},
		{20, 20, 1},
		{21, 20, 2},
		{40, 20, 2},
		{41, 20, 3},
		{100, 20, 5},
		{100, 10, 10},
		{5, 10, 1},
		{10, 10, 1},
		{15, 10, 2},
		{1, 1, 1},
		{0, 1, 0},
	}
	for _, tt := range tests {
		got := internal.PaginationTotalPages(tt.total, tt.pageSize)
		if got != tt.want {
			t.Errorf("PaginationTotalPages(%d, %d) = %d, want %d", tt.total, tt.pageSize, got, tt.want)
		}
	}
}

func TestPaginationTotalPages_InvalidPageSize(t *testing.T) {
	// pageSize <= 0 трактуем как 1
	if got := internal.PaginationTotalPages(5, 0); got != 5 {
		t.Errorf("PaginationTotalPages(5, 0) = %d, want 5", got)
	}
	if got := internal.PaginationTotalPages(5, -1); got != 5 {
		t.Errorf("PaginationTotalPages(5, -1) = %d, want 5", got)
	}
}

func TestParseInt(t *testing.T) {
	if got := internal.ParseInt("", 10); got != 10 {
		t.Errorf("ParseInt(\"\", 10) = %d, want 10", got)
	}
	if got := internal.ParseInt("42", 0); got != 42 {
		t.Errorf("ParseInt(\"42\", 0) = %d, want 42", got)
	}
	if got := internal.ParseInt("invalid", 7); got != 7 {
		t.Errorf("ParseInt(\"invalid\", 7) = %d, want 7", got)
	}
}
