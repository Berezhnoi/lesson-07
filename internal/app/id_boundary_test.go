package app_test

import (
	"net/http"
	"testing"
)

func TestMovieIDBoundaries(t *testing.T) {
	e := newEnv(t)
	tests := []struct {
		name   string
		path   string
		status int
		code   string
	}{
		{name: "empty", path: e.base() + "/", status: http.StatusBadRequest, code: "invalid_id"},
		{name: "zero is valid format", path: e.item(0), status: http.StatusNotFound, code: "not_found"},
		{name: "uint32 maximum is valid format", path: e.item("4294967295"), status: http.StatusNotFound, code: "not_found"},
		{name: "uint32 overflow", path: e.item("4294967296"), status: http.StatusBadRequest, code: "invalid_id"},
		{name: "very large overflow", path: e.item("99999999999999999999"), status: http.StatusBadRequest, code: "invalid_id"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := e.do(http.MethodGet, test.path, "")
			e.wantError(r, test.status, test.code)
		})
	}
}
