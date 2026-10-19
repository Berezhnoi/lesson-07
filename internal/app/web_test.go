package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"homework/internal/app"
	"homework/internal/model"
	"homework/internal/repository"
)

func TestCatalogPageSmokeAndEscaping(t *testing.T) {
	movies := repository.NewInMemoryMovieRepositoryAdapter()
	router := app.NewRouterWithDeps(app.Deps{Movies: movies})

	request := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET /catalog status = %d, body = %s", response.Code, response.Body)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	for _, text := range []string{"<!DOCTYPE html>", `<nav aria-label="Main navigation">`, "Catalog", "<table>", "No movies found"} {
		if !strings.Contains(response.Body.String(), text) {
			t.Errorf("catalog response missing %q", text)
		}
	}

	title := "<script>alert(1)</script>"
	if _, err := movies.Create(context.Background(), model.Movie{
		Title: title, Director: "Director", Country: "Country", Genre: "Drama",
	}); err != nil {
		t.Fatalf("create XSS fixture: %v", err)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/catalog", nil))
	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("GET /catalog with movie status = %d", response.Code)
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatal("escaped movie title not found")
	}
	if strings.Contains(body, title) {
		t.Fatal("movie title was rendered as a script tag")
	}
}

func TestWebRoutesCanBeDisabledWithoutChangingAPI(t *testing.T) {
	disabled := false
	router := app.NewRouterWithDeps(app.Deps{WebUIEnabled: &disabled})
	for _, path := range []string{"/catalog", "/login", "/register", "/static/css/style.css"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want 404", path, response.Code)
		}
		if !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
			t.Errorf("GET %s Content-Type = %q, want JSON", path, response.Header().Get("Content-Type"))
		}
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/movies", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("API GET /movies = %d (%q), want unchanged JSON 200", response.Code, response.Header().Get("Content-Type"))
	}
}

func TestWebRoutesAndStaticAssetsHaveSecurityHeaders(t *testing.T) {
	router := app.NewRouterWithDeps(app.Deps{})
	for _, requestPath := range []string{"/", "/catalog", "/static/css/style.css", "/static/js/htmx.min.js"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestPath, nil))
		if response.Code != http.StatusOK && requestPath != "/" {
			t.Errorf("GET %s status = %d, want 200", requestPath, response.Code)
		}
		registerResponse := httptest.NewRecorder()
		router.ServeHTTP(registerResponse, httptest.NewRequest(http.MethodGet, "/register", nil))
		if registerResponse.Code != http.StatusSeeOther || registerResponse.Header().Get("Location") != "/login?next=%2Fregister" {
			t.Errorf("anonymous GET /register = %d, Location %q; want redirect to login", registerResponse.Code, registerResponse.Header().Get("Location"))
		}
		if response.Header().Get("Content-Security-Policy") == "" {
			t.Errorf("GET %s missing Content-Security-Policy", requestPath)
		}
		if response.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("GET %s X-Content-Type-Options = %q", requestPath, response.Header().Get("X-Content-Type-Options"))
		}
		if response.Header().Get("Referrer-Policy") != "same-origin" {
			t.Errorf("GET %s Referrer-Policy = %q", requestPath, response.Header().Get("Referrer-Policy"))
		}
		if response.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Errorf("GET %s unexpectedly enables credentialed CORS", requestPath)
		}
	}
	rootResponse := httptest.NewRecorder()
	router.ServeHTTP(rootResponse, httptest.NewRequest(http.MethodGet, "/", nil))
	if rootResponse.Code != http.StatusFound || rootResponse.Header().Get("Location") != "/catalog" {
		t.Errorf("GET / = %d, Location %q; want redirect to /catalog", rootResponse.Code, rootResponse.Header().Get("Location"))
	}
	apiResponse := httptest.NewRecorder()
	router.ServeHTTP(apiResponse, httptest.NewRequest(http.MethodGet, "/api/v1/movies", nil))
	if apiResponse.Header().Get("Content-Security-Policy") != "" {
		t.Error("API response unexpectedly received web-only Content-Security-Policy")
	}
	css := httptest.NewRecorder()
	router.ServeHTTP(css, httptest.NewRequest(http.MethodGet, "/static/css/style.css", nil))
	if !strings.Contains(css.Body.String(), ".htmx-indicator") {
		t.Fatal("served stylesheet is missing the HTMX indicator rule")
	}
}

func TestCatalogInvalidPaginationRendersHTML400(t *testing.T) {
	router := app.NewRouterWithDeps(app.Deps{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/catalog?page=0", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("GET /catalog?page=0 status = %d, want 400", response.Code)
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "text/html") ||
		!strings.Contains(response.Body.String(), "<!DOCTYPE html>") {
		t.Fatalf("invalid pagination response is not an HTML error page: %s", response.Body)
	}
}
