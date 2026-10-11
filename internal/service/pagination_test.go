package service

import "testing"

func TestPaginationValuesAndOffset(t *testing.T) {
	tests := []struct {
		name       string
		page       string
		limit      string
		hasPage    bool
		hasLimit   bool
		total      int
		wantOffset int
		wantLimit  int
		wantEmpty  bool
		wantError  bool
	}{
		{name: "defaults", total: 25, wantLimit: 10},
		{name: "empty collection", total: 0, wantLimit: 10, wantEmpty: true},
		{name: "caps limit", page: "1", limit: "150", hasPage: true, hasLimit: true, total: 250, wantLimit: 100},
		{name: "caps limit beyond uint64", page: "1", limit: "999999999999999999999999999999999999999", hasPage: true, hasLimit: true, total: 250, wantLimit: 100},
		{name: "page beyond end", page: "4", limit: "2", hasPage: true, hasLimit: true, total: 5, wantOffset: 5, wantLimit: 2, wantEmpty: true},
		{name: "very large page", page: "9223372036854775807", limit: "10", hasPage: true, hasLimit: true, total: 25, wantOffset: 25, wantLimit: 10, wantEmpty: true},
		{name: "page beyond uint64", page: "999999999999999999999999999999999999999", limit: "10", hasPage: true, hasLimit: true, total: 25, wantOffset: 25, wantLimit: 10, wantEmpty: true},
		{name: "negative page", page: "-1", hasPage: true, total: 25, wantError: true},
		{name: "zero page", page: "0", hasPage: true, total: 25, wantError: true},
		{name: "empty page", page: "", hasPage: true, total: 25, wantError: true},
		{name: "nonnumeric page", page: "abc", hasPage: true, total: 25, wantError: true},
		{name: "fractional page", page: "1.5", hasPage: true, total: 25, wantError: true},
		{name: "negative limit", limit: "-1", hasLimit: true, total: 25, wantError: true},
		{name: "zero limit", limit: "0", hasLimit: true, total: 25, wantError: true},
		{name: "empty limit", limit: "", hasLimit: true, total: 25, wantError: true},
		{name: "nonnumeric limit", limit: "abc", hasLimit: true, total: 25, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, limit, err := parsePaginationValues(test.page, test.limit, test.hasPage, test.hasLimit)
			if test.wantError {
				if _, ok := err.(InvalidPaginationError); !ok {
					t.Fatalf("expected InvalidPaginationError, got %T (%v)", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected pagination error: %v", err)
			}
			offset, empty := paginationOffset(page, limit, test.total)
			if offset != test.wantOffset {
				t.Errorf("offset = %d, want %d", offset, test.wantOffset)
			}
			if limit != test.wantLimit {
				t.Errorf("limit = %d, want %d", limit, test.wantLimit)
			}
			if empty != test.wantEmpty {
				t.Errorf("empty = %t, want %t", empty, test.wantEmpty)
			}
		})
	}
}
