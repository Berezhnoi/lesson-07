package web_test

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"homework/internal/app"
	"homework/internal/auth"
	"homework/internal/model"
	"homework/internal/repository"
)

const (
	catalogTestSecret   = "web-catalog-integration-test-secret-at-least-32-bytes"
	catalogTestPassword = "web-test-password"
)

type catalogSession struct {
	accessCookie *http.Cookie
	csrfCookie   *http.Cookie
	csrfToken    string
	loginBody    string
	catalogBody  string
}

type catalogEnvironment struct {
	router   http.Handler
	movies   repository.MovieRepository
	users    *repository.InMemoryUserRepository
	tokens   *auth.TokenManager
	sessions map[string]catalogSession
	movieIDs map[string]uint32
}

func newCatalogEnvironment(t *testing.T) *catalogEnvironment {
	t.Helper()

	movies := repository.NewInMemoryMovieRepositoryAdapter()
	movieIDs := make(map[string]uint32)
	for _, movie := range []model.Movie{
		{Title: "Star Search", Director: "A. Director", Country: "US", Genre: "Drama"},
		{Title: "The Other Film", Director: "B. Director", Country: "UK", Genre: "Comedy"},
		{
			Title: `<script>alert(1)</script>`, Director: `"><img src=x onerror=alert(1)>`,
			Country: "US", Genre: "Thriller",
		},
	} {
		created, err := movies.Create(context.Background(), movie)
		if err != nil {
			t.Fatalf("seed movie %q: %v", movie.Title, err)
		}
		movieIDs[movie.Title] = created.ID
	}

	users := repository.NewInMemoryUserRepository()
	for _, role := range []string{model.RoleViewer, model.RoleEditor, model.RoleAdmin} {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(catalogTestPassword), bcrypt.MinCost)
		if err != nil {
			t.Fatalf("hash password for %s: %v", role, err)
		}
		user := model.User{
			Email: role + "@example.test", Name: strings.Title(role),
			PasswordHash: string(passwordHash), Role: role,
		}
		if err := users.Create(context.Background(), &user); err != nil {
			t.Fatalf("create %s user: %v", role, err)
		}
	}
	tokens, err := auth.NewTokenManager(catalogTestSecret, 24*time.Hour)
	if err != nil {
		t.Fatalf("create test token manager: %v", err)
	}
	router := app.NewRouterWithDeps(app.Deps{Movies: movies, Users: users, Tokens: tokens})
	environment := &catalogEnvironment{
		router: router, movies: movies, users: users, tokens: tokens,
		sessions: make(map[string]catalogSession), movieIDs: movieIDs,
	}
	for _, role := range []string{model.RoleViewer, model.RoleEditor, model.RoleAdmin} {
		environment.sessions[role] = environment.login(t, role)
	}
	return environment
}

func (environment *catalogEnvironment) login(t *testing.T, role string) catalogSession {
	t.Helper()

	loginPage := serveCatalogRequest(environment.router, http.MethodGet, "/login", nil)
	if loginPage.Code != http.StatusOK {
		t.Fatalf("GET /login for %s = %d: %s", role, loginPage.Code, loginPage.Body)
	}
	csrfCookie := responseCookie(t, loginPage, "csrf_token")
	loginCSRF := csrfTokenFromHTML(t, loginPage.Body.String())

	form := url.Values{
		"email":      {role + "@example.test"},
		"password":   {catalogTestPassword},
		"csrf_token": {loginCSRF},
	}
	loginResponse := serveCatalogRequest(environment.router, http.MethodPost, "/login", strings.NewReader(form.Encode()),
		&http.Cookie{Name: csrfCookie.Name, Value: csrfCookie.Value, Path: csrfCookie.Path},
	)
	if loginResponse.Code != http.StatusSeeOther {
		t.Fatalf("POST /login for %s = %d: %s", role, loginResponse.Code, loginResponse.Body)
	}
	accessCookie := responseCookie(t, loginResponse, "access_token")

	catalogResponse := serveCatalogRequest(environment.router, http.MethodGet, "/catalog", nil,
		accessCookie, csrfCookie)
	if catalogResponse.Code != http.StatusOK {
		t.Fatalf("GET /catalog after %s login = %d: %s", role, catalogResponse.Code, catalogResponse.Body)
	}
	token := csrfTokenFromHTML(t, catalogResponse.Body.String())
	if token != csrfCookie.Value {
		t.Fatalf("catalog CSRF token = %q, cookie token = %q", token, csrfCookie.Value)
	}
	return catalogSession{
		accessCookie: accessCookie, csrfCookie: csrfCookie, csrfToken: token,
		loginBody: loginResponse.Body.String(), catalogBody: catalogResponse.Body.String(),
	}
}

func TestCatalogWebIntegration(t *testing.T) {
	environment := newCatalogEnvironment(t)

	t.Run("catalog renders the complete layout and seeded movies", func(t *testing.T) {
		response := serveCatalogRequest(environment.router, http.MethodGet, "/catalog", nil)
		if response.Code != http.StatusOK {
			t.Fatalf("GET /catalog = %d: %s", response.Code, response.Body)
		}
		if got := response.Result().Header.Get("Content-Type"); got != "text/html; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
		}
		body := response.Body.String()
		for _, expected := range []string{
			"<!DOCTYPE html>", `<nav aria-label="Main navigation">`, "<main",
			"<table>", "Star Search", "The Other Film",
			fmt.Sprintf(`<tr id="movie-%d">`, environment.movieIDs["Star Search"]),
			fmt.Sprintf(`<tr id="movie-%d">`, environment.movieIDs["The Other Film"]),
		} {
			if !strings.Contains(body, expected) {
				t.Errorf("catalog response is missing %q", expected)
			}
		}
	})

	t.Run("HTMX search returns only matching movie row fragments", func(t *testing.T) {
		tests := []struct {
			name        string
			query       string
			wantRows    []string
			notWantRows []string
			wantEmpty   bool
		}{
			{
				name: "matching rows", query: "Star",
				wantRows:    []string{fmt.Sprintf(`<tr id="movie-%d">`, environment.movieIDs["Star Search"])},
				notWantRows: []string{fmt.Sprintf(`<tr id="movie-%d">`, environment.movieIDs["The Other Film"])},
			},
			{name: "no matching rows", query: "not-in-the-catalog", wantEmpty: true},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				query := url.Values{"q": {test.query}}
				request := httptest.NewRequest(http.MethodGet, "/web/movies?"+query.Encode(), nil)
				request.Header.Set("HX-Request", "true")
				response := serveCatalogRequestWithRequest(environment.router, request)
				if response.Code != http.StatusOK {
					t.Fatalf("HTMX GET /web/movies = %d: %s", response.Code, response.Body)
				}
				body := response.Body.String()
				for _, forbidden := range []string{"<html", "<head", "<nav", "<!DOCTYPE"} {
					if strings.Contains(body, forbidden) {
						t.Errorf("fragment contains layout markup %q: %s", forbidden, body)
					}
				}
				for _, expected := range test.wantRows {
					if !strings.Contains(body, expected) {
						t.Errorf("fragment is missing %q: %s", expected, body)
					}
				}
				for _, unwanted := range test.notWantRows {
					if strings.Contains(body, unwanted) {
						t.Errorf("fragment contains non-matching row %q: %s", unwanted, body)
					}
				}
				if test.wantEmpty && !strings.Contains(body, `<tr class="empty">`) {
					t.Errorf("no-match fragment is missing the empty-state row: %s", body)
				}
			})
		}
	})

	t.Run("delete enforces role authorization and CSRF", func(t *testing.T) {
		tests := []struct {
			name       string
			role       string
			submitCSRF bool
			wantStatus int
			wantHX     string
			wantDelete bool
		}{
			{name: "anonymous", submitCSRF: true, wantStatus: http.StatusUnauthorized, wantHX: "/login"},
			{name: "viewer", role: model.RoleViewer, submitCSRF: true, wantStatus: http.StatusForbidden},
			{name: "editor", role: model.RoleEditor, submitCSRF: true, wantStatus: http.StatusForbidden},
			{name: "admin", role: model.RoleAdmin, submitCSRF: true, wantStatus: http.StatusOK, wantDelete: true},
			{name: "anonymous missing CSRF", submitCSRF: false, wantStatus: http.StatusForbidden},
			{name: "viewer missing CSRF", role: model.RoleViewer, submitCSRF: false, wantStatus: http.StatusForbidden},
			{name: "editor missing CSRF", role: model.RoleEditor, submitCSRF: false, wantStatus: http.StatusForbidden},
			{name: "admin missing CSRF", role: model.RoleAdmin, submitCSRF: false, wantStatus: http.StatusForbidden},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				movie, err := environment.movies.Create(context.Background(), model.Movie{
					Title: "Delete authorization fixture", Director: "Director", Country: "US", Genre: "Drama",
				})
				if err != nil {
					t.Fatalf("create delete fixture: %v", err)
				}
				session := environment.sessions[model.RoleViewer]
				if test.role != "" {
					session = environment.sessions[test.role]
				}
				request := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/web/movies/%d", movie.ID), nil)
				request.Header.Set("HX-Request", "true")
				request.AddCookie(session.csrfCookie)
				if test.role != "" {
					request.AddCookie(session.accessCookie)
				}
				if test.submitCSRF {
					request.Header.Set("X-CSRF-Token", session.csrfToken)
				}
				response := serveCatalogRequestWithRequest(environment.router, request)
				if response.Code != test.wantStatus {
					t.Fatalf("DELETE /web/movies/%d = %d, want %d: %s", movie.ID, response.Code, test.wantStatus, response.Body)
				}
				if got := response.Result().Header.Get("HX-Redirect"); got != test.wantHX {
					t.Errorf("HX-Redirect = %q, want %q", got, test.wantHX)
				}
				if test.wantDelete && response.Body.Len() != 0 {
					t.Errorf("successful HTMX delete body = %q, want empty", response.Body.String())
				}
				_, found := environment.movies.GetByID(context.Background(), movie.ID)
				if found == test.wantDelete {
					t.Errorf("movie existence after delete = %t, want %t", found, !test.wantDelete)
				}
			})
		}
	})

	t.Run("hostile title and director are escaped in pages and fragments", func(t *testing.T) {
		page := serveCatalogRequest(environment.router, http.MethodGet, "/catalog", nil)
		fragmentRequest := httptest.NewRequest(http.MethodGet, "/web/movies", nil)
		fragmentRequest.Header.Set("HX-Request", "true")
		fragment := serveCatalogRequestWithRequest(environment.router, fragmentRequest)
		for name, body := range map[string]string{
			"catalog page": page.Body.String(), "HTMX fragment": fragment.Body.String(),
		} {
			if page.Code != http.StatusOK || fragment.Code != http.StatusOK {
				t.Fatalf("escaping responses have statuses (%d, %d), want 200", page.Code, fragment.Code)
			}
			if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") ||
				!strings.Contains(body, `&#34;&gt;&lt;img src=x onerror=alert(1)&gt;`) {
				t.Errorf("%s is missing escaped hostile content: %s", name, body)
			}
			if strings.Contains(body, "<script>alert(1)</script>") || strings.Contains(body, "<img src=x") {
				t.Errorf("%s contains raw hostile markup: %s", name, body)
			}
		}
	})

	t.Run("cookies API authentication and web security headers", func(t *testing.T) {
		session := environment.sessions[model.RoleViewer]
		accessToken := session.accessCookie.Value
		if !session.accessCookie.HttpOnly || session.accessCookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("access_token cookie attributes = %+v, want HttpOnly and SameSite=Lax", session.accessCookie)
		}
		if strings.Contains(session.loginBody, accessToken) || strings.Contains(session.catalogBody, accessToken) {
			t.Fatal("access token appeared in an HTML response body")
		}
		if session.csrfCookie.Name != "csrf_token" || session.csrfToken == "" {
			t.Fatalf("CSRF was not issued: cookie=%+v token=%q", session.csrfCookie, session.csrfToken)
		}
		for _, body := range []string{session.catalogBody} {
			if strings.Contains(body, accessToken) {
				t.Fatal("access token appeared in an HTML body")
			}
		}

		for _, path := range []string{"/catalog", "/login"} {
			webPage := serveCatalogRequest(environment.router, http.MethodGet, path, nil)
			if webPage.Code != http.StatusOK {
				t.Errorf("GET %s = %d, want 200", path, webPage.Code)
			}
			if strings.Contains(webPage.Body.String(), accessToken) {
				t.Errorf("access token appeared in %s HTML body", path)
			}
			if webPage.Result().Header.Get("Content-Security-Policy") == "" ||
				webPage.Result().Header.Get("X-Content-Type-Options") != "nosniff" {
				t.Errorf("web security headers missing from %s: CSP=%q nosniff=%q", path,
					webPage.Result().Header.Get("Content-Security-Policy"), webPage.Result().Header.Get("X-Content-Type-Options"))
			}
		}
		cookieOnly := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		cookieOnly.AddCookie(session.accessCookie)
		cookieOnlyResponse := serveCatalogRequestWithRequest(environment.router, cookieOnly)
		if cookieOnlyResponse.Code != http.StatusUnauthorized {
			t.Errorf("API request with only access_token cookie = %d, want 401", cookieOnlyResponse.Code)
		}
		apiRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		apiRequest.Header.Set("Authorization", "Bearer "+accessToken)
		apiResponse := serveCatalogRequestWithRequest(environment.router, apiRequest)
		if apiResponse.Code != http.StatusOK {
			t.Errorf("API request with bearer token = %d, want 200: %s", apiResponse.Code, apiResponse.Body)
		}
		if apiResponse.Result().Header.Get("Content-Security-Policy") != "" ||
			apiResponse.Result().Header.Get("X-Content-Type-Options") != "" {
			t.Errorf("API response unexpectedly has web security headers: %+v", apiResponse.Result().Header)
		}
	})

	t.Run("web UI switch preserves JSON 404 and API behavior", func(t *testing.T) {
		disabled := false
		disabledRouter := app.NewRouterWithDeps(app.Deps{
			Movies: environment.movies, Users: environment.users, Tokens: environment.tokens,
			WebUIEnabled: &disabled,
		})
		for _, path := range []string{"/catalog", "/login", "/web/movies", "/static/css/style.css"} {
			response := serveCatalogRequest(disabledRouter, http.MethodGet, path, nil)
			if response.Code != http.StatusNotFound ||
				!strings.HasPrefix(response.Result().Header.Get("Content-Type"), "application/json") ||
				!strings.Contains(response.Body.String(), `"code":"not_found"`) {
				t.Errorf("disabled GET %s = %d (%q): %s; want JSON 404", path,
					response.Code, response.Result().Header.Get("Content-Type"), response.Body)
			}
		}

		for name, router := range map[string]http.Handler{"enabled": environment.router, "disabled": disabledRouter} {
			unauthenticated := serveCatalogRequest(router, http.MethodGet, "/api/v1/movies", nil)
			if unauthenticated.Code != http.StatusOK {
				t.Errorf("%s UI mode: public API GET /movies = %d, want 200", name, unauthenticated.Code)
			}
			apiRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
			apiRequest.Header.Set("Authorization", "Bearer "+environment.sessions[model.RoleViewer].accessCookie.Value)
			apiResponse := serveCatalogRequestWithRequest(router, apiRequest)
			if apiResponse.Code != http.StatusOK {
				t.Errorf("%s UI mode: API GET /me = %d, want 200", name, apiResponse.Code)
			}
			if apiResponse.Result().Header.Get("Content-Security-Policy") != "" ||
				apiResponse.Result().Header.Get("X-Content-Type-Options") != "" {
				t.Errorf("%s UI mode: API response has web headers", name)
			}
		}
	})
}

func serveCatalogRequest(router http.Handler, method, path string, body *strings.Reader, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var requestBody io.Reader
	if body != nil {
		requestBody = body
	}
	request := httptest.NewRequest(method, path, requestBody)
	if method == http.MethodPost {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	return serveCatalogRequestWithRequest(router, request)
}

func serveCatalogRequestWithRequest(router http.Handler, request *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func responseCookie(t *testing.T, response *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("response did not set %q cookie", name)
	return nil
}

var csrfMetaPattern = regexp.MustCompile(`name="csrf-token" content="([^"]*)"`)

func csrfTokenFromHTML(t *testing.T, body string) string {
	t.Helper()
	match := csrfMetaPattern.FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("HTML body contains no CSRF token meta element: %s", body)
	}
	return html.UnescapeString(match[1])
}
