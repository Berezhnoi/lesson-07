package service

import (
	"strings"
	"testing"
)

func TestValidateMovieJSON(t *testing.T) {
	valid := `{"title":"  Film title  ","director":"Director","country":"Country","genre":"Drama"}`
	tests := []struct {
		name      string
		body      string
		wantError bool
	}{
		{name: "valid object", body: valid},
		{name: "valid optional fields", body: `{"title":"Film","director":"Director","country":"Country","genre":"Drama","release_year":2015,"synopsis":""}`},
		{name: "nullable optional fields", body: `{"title":"Film","director":"Director","country":"Country","genre":"Drama","release_year":null,"synopsis":null}`},
		{name: "ignores extra and system fields", body: `{"title":"Film","director":"Director","country":"Country","genre":"Drama","extra":true,"id":99,"created_at":"bad","updated_at":"bad"}`},
		{name: "broken JSON", body: `{broken`, wantError: true},
		{name: "empty body", body: ``, wantError: true},
		{name: "array root", body: `[]`, wantError: true},
		{name: "string root", body: `"movie"`, wantError: true},
		{name: "number root", body: `42`, wantError: true},
		{name: "null root", body: `null`, wantError: true},
		{name: "trailing JSON", body: valid + ` {}`, wantError: true},
		{name: "trailing garbage", body: valid + ` garbage`, wantError: true},
		{name: "missing required field", body: `{"director":"Director","country":"Country","genre":"Drama"}`, wantError: true},
		{name: "null required field", body: `{"title":null,"director":"Director","country":"Country","genre":"Drama"}`, wantError: true},
		{name: "non-string required field", body: `{"title":1,"director":"Director","country":"Country","genre":"Drama"}`, wantError: true},
		{name: "empty required field", body: `{"title":"","director":"Director","country":"Country","genre":"Drama"}`, wantError: true},
		{name: "whitespace required field", body: `{"title":" \t ","director":"Director","country":"Country","genre":"Drama"}`, wantError: true},
		{name: "fractional release year", body: strings.Replace(valid, `"genre":"Drama"`, `"genre":"Drama","release_year":1.5`, 1), wantError: true},
		{name: "string release year", body: strings.Replace(valid, `"genre":"Drama"`, `"genre":"Drama","release_year":"2015"`, 1), wantError: true},
		{name: "boolean release year", body: strings.Replace(valid, `"genre":"Drama"`, `"genre":"Drama","release_year":true`, 1), wantError: true},
		{name: "overflowing release year", body: strings.Replace(valid, `"genre":"Drama"`, `"genre":"Drama","release_year":999999999999999999999999`, 1), wantError: true},
		{name: "non-string synopsis", body: strings.Replace(valid, `"genre":"Drama"`, `"genre":"Drama","synopsis":3`, 1), wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, err := ValidateMovieJSON([]byte(test.body))
			if test.wantError {
				if _, ok := err.(ValidationError); !ok {
					t.Fatalf("expected ValidationError, got %T (%v)", err, err)
				}
				if err.Error() == "" {
					t.Fatal("validation error must have a non-empty message")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
			if test.name == "valid object" && input.Title != "  Film title  " {
				t.Errorf("required field was modified: got %q", input.Title)
			}
			if test.name == "valid optional fields" {
				if input.ReleaseYear == nil || *input.ReleaseYear != 2015 {
					t.Errorf("release_year = %v, want 2015", input.ReleaseYear)
				}
				if input.Synopsis == nil || *input.Synopsis != "" {
					t.Errorf("synopsis = %v, want pointer to empty string", input.Synopsis)
				}
			}
			if test.name == "nullable optional fields" && (input.ReleaseYear != nil || input.Synopsis != nil) {
				t.Errorf("null optional fields should remain nil, got release_year=%v synopsis=%v", input.ReleaseYear, input.Synopsis)
			}
		})
	}
}
