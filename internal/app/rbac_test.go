package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"homework/internal/model"
)

const rbacValidMovieBody = `{"title":"Test Movie","director":"Test Director","country":"Test Country","genre":"Drama"}`

type rbacMatrixCase struct {
	name   string
	method string
	path   string
	body   string
	want   [4]int
}

func TestRBAC_AccessMatrix(t *testing.T) {
	tests := []rbacMatrixCase{
		{name: "GET /movies", method: http.MethodGet, path: "/api/v1/movies", want: [4]int{200, 200, 200, 200}},
		{name: "GET /movies/:id", method: http.MethodGet, path: "/api/v1/movies/1", want: [4]int{200, 200, 200, 200}},
		{name: "GET /me", method: http.MethodGet, path: "/api/v1/me", want: [4]int{401, 200, 200, 200}},
		{name: "POST /movies", method: http.MethodPost, path: "/api/v1/movies", body: rbacValidMovieBody, want: [4]int{401, 403, 201, 201}},
		{name: "PUT /movies/:id", method: http.MethodPut, path: "/api/v1/movies/1", body: rbacValidMovieBody, want: [4]int{401, 403, 200, 200}},
		{name: "DELETE /movies/:id", method: http.MethodDelete, path: "/api/v1/movies/1", want: [4]int{401, 403, 403, 204}},
		{name: "GET /users", method: http.MethodGet, path: "/api/v1/users", want: [4]int{401, 403, 403, 200}},
		{name: "PUT /users/:id/role", method: http.MethodPut, path: "/api/v1/users/2/role", body: `{"role":"editor"}`, want: [4]int{401, 403, 403, 200}},
	}
	roles := []struct {
		name   string
		column int
	}{
		{name: "anonymous", column: 0},
		{name: "viewer", column: 1},
		{name: "editor", column: 2},
		{name: "admin", column: 3},
	}

	for _, test := range tests {
		for _, role := range roles {
			t.Run(test.name+"/"+role.name, func(t *testing.T) {
				env := newAuthTestEnv(t)
				target := env.createUser(t, "target@example.com", authTestPassword, model.RoleViewer)
				token := ""
				if role.name != "anonymous" {
					user := env.createUser(t, role.name+"@example.com", authTestPassword, role.name)
					token = env.login(t, user.Email, authTestPassword)
				}
				movieCreator := env.createUser(t, "movie-creator@example.com", authTestPassword, model.RoleEditor)
				movieCreatorToken := env.login(t, movieCreator.Email, authTestPassword)
				createdMovie := env.do(http.MethodPost, "/api/v1/movies", rbacValidMovieBody, movieCreatorToken)
				if createdMovie.Code != http.StatusCreated {
					t.Fatalf("seed movie status = %d, want 201: %s", createdMovie.Code, createdMovie.Body)
				}

				path := test.path
				if strings.Contains(path, "/users/2/") {
					path = fmt.Sprintf("/api/v1/users/%d/role", target.ID)
				}
				response := env.do(test.method, path, test.body, token)
				want := test.want[role.column]
				if response.Code != want {
					t.Errorf("%s as %s status = %d, want %d: %s", test.name, role.name, response.Code, want, response.Body)
				}
			})
		}
	}
}

func TestRBAC_AuthenticationAndValidationOrder(t *testing.T) {
	tests := []struct {
		name   string
		role   string
		path   string
		body   string
		status int
	}{
		{name: "anonymous malformed ID", path: "/api/v1/movies/abc", body: rbacValidMovieBody, status: http.StatusUnauthorized},
		{name: "viewer malformed ID", role: model.RoleViewer, path: "/api/v1/movies/abc", body: rbacValidMovieBody, status: http.StatusForbidden},
		{name: "editor malformed ID", role: model.RoleEditor, path: "/api/v1/movies/abc", body: rbacValidMovieBody, status: http.StatusBadRequest},
		{name: "editor invalid body", role: model.RoleEditor, path: "/api/v1/movies/999", body: `{"title":`, status: http.StatusUnprocessableEntity},
		{name: "editor valid body missing movie", role: model.RoleEditor, path: "/api/v1/movies/999", body: rbacValidMovieBody, status: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := newAuthTestEnv(t)
			token := ""
			if test.role != "" {
				user := env.createUser(t, test.role+"@example.com", authTestPassword, test.role)
				token = env.login(t, user.Email, authTestPassword)
			}
			response := env.do(http.MethodPut, test.path, test.body, token)
			if response.Code != test.status {
				t.Fatalf("PUT %s status = %d, want %d: %s", test.path, response.Code, test.status, response.Body)
			}
			if test.status == http.StatusUnauthorized || test.status == http.StatusForbidden {
				assertRBACError(t, response, map[int]string{401: "unauthorized", 403: "forbidden"}[test.status])
			}
		})
	}
}

func TestRBAC_UpdateUserRoles(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		newRole  string
		wantCode int
	}{
		{name: "set editor", target: "viewer", newRole: model.RoleEditor, wantCode: http.StatusOK},
		{name: "invalid role", target: "viewer", newRole: "root", wantCode: http.StatusUnprocessableEntity},
		{name: "unknown user", target: "unknown", newRole: model.RoleEditor, wantCode: http.StatusNotFound},
		{name: "demote last admin", target: "admin", newRole: model.RoleViewer, wantCode: http.StatusUnprocessableEntity},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := newAuthTestEnv(t)
			admin := env.createUser(t, "admin@example.com", authTestPassword, model.RoleAdmin)
			token := env.login(t, admin.Email, authTestPassword)

			var targetID uint64
			switch test.target {
			case "viewer":
				target := env.createUser(t, "target@example.com", authTestPassword, model.RoleViewer)
				targetID = target.ID
			case "admin":
				targetID = admin.ID
			case "unknown":
				targetID = 9999
			default:
				t.Fatalf("unknown test target %q", test.target)
			}
			body, err := json.Marshal(map[string]string{"role": test.newRole})
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			response := env.do(http.MethodPut, fmt.Sprintf("/api/v1/users/%d/role", targetID), string(body), token)
			if response.Code != test.wantCode {
				t.Fatalf("role update status = %d, want %d: %s", response.Code, test.wantCode, response.Body)
			}
			if test.wantCode == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(response.Body, &result); err != nil {
					t.Fatalf("decode role update response: %v (body: %s)", err, response.Body)
				}
				if result["role"] != test.newRole {
					t.Errorf("updated role = %v, want %q", result["role"], test.newRole)
				}
			} else {
				wantCode := map[int]string{
					http.StatusUnprocessableEntity: "validation_error",
					http.StatusNotFound:            "not_found",
				}[test.wantCode]
				assertRBACError(t, response, wantCode)
			}
		})
	}
}

func TestRBAC_PromotedUsersReceiveNewRoleOnLogin(t *testing.T) {
	env := newAuthTestEnv(t)
	admin := env.createUser(t, "admin@example.com", authTestPassword, model.RoleAdmin)
	target := env.createUser(t, "target@example.com", authTestPassword, model.RoleViewer)
	adminToken := env.login(t, admin.Email, authTestPassword)

	promotion := env.do(http.MethodPut, fmt.Sprintf("/api/v1/users/%d/role", target.ID),
		`{"role":"editor"}`, adminToken)
	if promotion.Code != http.StatusOK {
		t.Fatalf("promotion status = %d, want 200: %s", promotion.Code, promotion.Body)
	}
	newToken := env.login(t, target.Email, authTestPassword)
	claims, err := env.tokens.Parse(newToken)
	if err != nil {
		t.Fatalf("parse new login token: %v", err)
	}
	if claims.Role != model.RoleEditor {
		t.Fatalf("new token role = %q, want editor", claims.Role)
	}
}

func TestRBAC_DeniedRequestsHaveNoMovieSideEffectsOrInternals(t *testing.T) {
	tests := []struct {
		name   string
		role   string
		status int
		code   string
	}{
		{name: "anonymous", status: http.StatusUnauthorized, code: "unauthorized"},
		{name: "viewer", role: model.RoleViewer, status: http.StatusForbidden, code: "forbidden"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := newAuthTestEnv(t)
			token := ""
			if test.role != "" {
				user := env.createUser(t, "viewer@example.com", authTestPassword, test.role)
				token = env.login(t, user.Email, authTestPassword)
			}
			response := env.do(http.MethodPost, "/api/v1/movies", rbacValidMovieBody, token)
			if response.Code != test.status {
				t.Fatalf("POST /movies status = %d, want %d: %s", response.Code, test.status, response.Body)
			}
			assertRBACError(t, response, test.code)
			for _, internal := range []string{"panic", "stack", "repository", "database", "sql", "goroutine"} {
				if strings.Contains(strings.ToLower(string(response.Body)), internal) {
					t.Errorf("denial response exposes %q: %s", internal, response.Body)
				}
			}

			list := env.do(http.MethodGet, "/api/v1/movies?limit=100", "", "")
			if list.Code != http.StatusOK {
				t.Fatalf("GET /movies status = %d, want 200: %s", list.Code, list.Body)
			}
			var movies []json.RawMessage
			if err := json.Unmarshal(list.Body, &movies); err != nil {
				t.Fatalf("decode movie list: %v (body: %s)", err, list.Body)
			}
			if len(movies) != 0 {
				t.Errorf("denied POST created %d movie(s): %s", len(movies), list.Body)
			}
		})
	}
}

func TestRBAC_ConcurrentRequestsAcrossRoles(t *testing.T) {
	env := newAuthTestEnv(t)
	roles := []struct {
		name     string
		status   int
		token    string
		requests int
	}{
		{name: "anonymous", status: http.StatusUnauthorized, requests: 8},
		{name: model.RoleViewer, status: http.StatusForbidden, requests: 8},
		{name: model.RoleEditor, status: http.StatusCreated, requests: 8},
		{name: model.RoleAdmin, status: http.StatusCreated, requests: 8},
	}
	for index := range roles {
		if roles[index].name == "anonymous" {
			continue
		}
		user := env.createUser(t, roles[index].name+"@example.com", authTestPassword, roles[index].name)
		roles[index].token = env.login(t, user.Email, authTestPassword)
	}

	type result struct {
		role   string
		status int
		body   []byte
	}
	totalRequests := 0
	for _, role := range roles {
		totalRequests += role.requests
	}
	results := make(chan result, totalRequests)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for _, role := range roles {
		role := role
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			for index := 0; index < role.requests; index++ {
				body := fmt.Sprintf(`{"title":"%s-%d","director":"Director","country":"Country","genre":"Drama"}`, role.name, index)
				response := env.do(http.MethodPost, "/api/v1/movies", body, role.token)
				results <- result{role: role.name, status: response.Code, body: response.Body}
			}
		}()
	}
	close(start)
	workers.Wait()
	close(results)

	created := 0
	for result := range results {
		want := http.StatusUnauthorized
		switch result.role {
		case model.RoleViewer:
			want = http.StatusForbidden
		case model.RoleEditor, model.RoleAdmin:
			want = http.StatusCreated
		}
		if result.status != want {
			t.Errorf("%s concurrent POST status = %d, want %d: %s", result.role, result.status, want, result.body)
		}
		if result.status == http.StatusCreated {
			created++
		}
	}
	if created != 16 {
		t.Errorf("created movie responses = %d, want 16", created)
	}
	list := env.do(http.MethodGet, "/api/v1/movies?limit=100", "", "")
	var movies []json.RawMessage
	if err := json.Unmarshal(list.Body, &movies); err != nil {
		t.Fatalf("decode concurrent movie list: %v (body: %s)", err, list.Body)
	}
	if len(movies) != created {
		t.Errorf("movie list contains %d movies, want %d", len(movies), created)
	}
}

func assertRBACError(t *testing.T, response authTestResponse, code string) {
	t.Helper()

	var result struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body, &result); err != nil {
		t.Fatalf("decode error response: %v (body: %s)", err, response.Body)
	}
	if result.Error.Code != code {
		t.Errorf("error code = %q, want %q: %s", result.Error.Code, code, response.Body)
	}
	if response.Code == http.StatusUnauthorized && response.Header.Get("WWW-Authenticate") != "Bearer" {
		t.Errorf("WWW-Authenticate = %q, want Bearer", response.Header.Get("WWW-Authenticate"))
	}
}
