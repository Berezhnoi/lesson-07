package web

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"homework/internal/model"
	webui "homework/web"
)

func TestRendererParsesEmbeddedTemplatesOutsideRepository(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	renderer, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	catalog := CatalogPage{
		BasePage: BasePage{Title: "Catalog - Movies"},
		Movies: []MovieRowView{MapMovie(model.Movie{
			ID: 1, Title: "Sample", Director: "Director", Country: "Country", Genre: "Drama",
		}, now)},
		Genres: []GenreOption{{Value: "Drama", Label: "Drama", Selected: true}},
		Q:      "sample", Genre: "Drama", Page: 1, Limit: 10,
	}
	for name, data := range map[string]any{
		"catalog": catalog,
		"error":   ErrorPage{StatusText: "Bad Request", Message: "Please try again."},
	} {
		t.Run(name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest("GET", "/"+name, nil)
			renderer.Page(context, 200, name, data)
			if context.Writer.Status() != 200 {
				t.Fatalf("status = %d, want 200", context.Writer.Status())
			}
		})
	}
	if _, err := webui.TemplateFS.Open(filepath.Join("templates", "layout.html")); err != nil {
		t.Fatalf("embedded template unavailable from empty working directory: %v", err)
	}
}

func TestRendererEscapesMovieTitle(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	title := "<script>alert(1)</script>"
	page := CatalogPage{
		BasePage: BasePage{Title: "Catalog - Movies"},
		Movies:   []MovieRowView{{Title: title}},
		Page:     1, Limit: 10,
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("GET", "/catalog", nil)
	renderer.Page(context, 200, "catalog", page)
	body := recorder.Body.String()
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatalf("escaped title missing from rendered output: %s", body)
	}
	if strings.Contains(body, title) {
		t.Fatalf("raw script title present in rendered output")
	}
}

func TestReleaseStatusBoundariesAndMovieMapping(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		year int
		want string
	}{
		{year: 2027, want: "Upcoming"},
		{year: 2026, want: "New"},
		{year: 2024, want: "New"},
		{year: 2023, want: "Classic"},
	}
	for _, test := range tests {
		label, _ := ReleaseStatus(test.year, now)
		if label != test.want {
			t.Errorf("ReleaseStatus(%d) = %q, want %q", test.year, label, test.want)
		}
	}
	row := MapMovie(model.Movie{Title: "Unknown"}, now)
	if row.YearText != "-" || row.StatusLabel != "" || row.StatusClass != "" {
		t.Errorf("unknown release mapping = %#v", row)
	}
}
