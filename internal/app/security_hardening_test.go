package app_test

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"homework/internal/app"
	"homework/internal/auth"
	"homework/internal/handler"
	"homework/internal/middleware"
	"homework/internal/model"
	"homework/internal/repository"
)

func TestRequestLoggingRedactsCredentialsAndBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	users := &hashCapturingUserRepository{UserRepository: repository.NewInMemoryUserRepository()}
	tokens, err := auth.NewTokenManager("logging-test-secret-with-at-least-32-bytes", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	router := app.NewRouterWithDeps(app.Deps{Users: users, Tokens: tokens})

	var captured bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&captured)
	defer log.SetOutput(previousOutput)

	const password = "logging-test-password"
	registerRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader(`{"email":"log-reader@example.com","password":"logging-test-password","name":"Log Reader"}`))
	registerRequest.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()
	router.ServeHTTP(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusAccepted {
		t.Fatalf("register status = %d, want 202: %s", registerResponse.Code, registerResponse.Body)
	}

	token, err := tokens.Issue(model.User{ID: 1, Role: model.RoleViewer})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	const querySecret = "query-secret-value"
	protectedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me?token="+querySecret, nil)
	protectedRequest.Header.Set("Authorization", "Bearer "+token)
	protectedResponse := httptest.NewRecorder()
	router.ServeHTTP(protectedResponse, protectedRequest)
	if protectedResponse.Code != http.StatusOK {
		t.Fatalf("protected status = %d, want 200: %s", protectedResponse.Code, protectedResponse.Body)
	}

	logOutput := captured.String()
	for _, secret := range []string{password, token, users.passwordHash, querySecret, "log-reader@example.com", "Log Reader"} {
		if secret == "" {
			t.Fatal("test did not capture the generated password hash")
		}
		if strings.Contains(logOutput, secret) {
			t.Errorf("request logs contain sensitive value %q: %q", secret, logOutput)
		}
	}
	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	if len(lines) != 2 {
		t.Fatalf("logged %d lines, want one per request: %q", len(lines), logOutput)
	}
	for index, expected := range [][]string{
		{"POST", "/api/v1/auth/register", "202"},
		{"GET", "/api/v1/me", "200"},
	} {
		fields := strings.Fields(lines[index])
		if len(fields) != 4 || fields[0] != expected[0] || fields[1] != expected[1] || fields[2] != expected[2] {
			t.Errorf("log line %q must contain only method, path, status, duration", lines[index])
			continue
		}
		if _, err := time.ParseDuration(fields[3]); err != nil {
			t.Errorf("log line %q has invalid duration: %v", lines[index], err)
		}
	}
}

func TestGlobalRequestBodyLimitReturnsStandardError(t *testing.T) {
	e := newEnv(t)
	body := strings.Repeat("x", 1<<20+1)
	response := e.do(http.MethodPost, "/api/v1/auth/register", body)
	if response.Code != http.StatusUnprocessableEntity && response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized request status = %d, want 413 or 422: %s", response.Code, response.Body)
	}
	if !isJSONContent(response) {
		t.Fatalf("oversized request Content-Type = %q, want JSON", response.Header.Get("Content-Type"))
	}
	assertNoInternalLeak(t, response)
	if strings.Contains(strings.ToLower(string(response.Body)), "maxbytes") {
		t.Fatalf("oversized request exposed internal body-limit details: %s", response.Body)
	}
}

func TestRecoveryDoesNotExposeAuthPanicText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.Recovery(), middleware.RequestLogger())
	router.POST("/auth/login", handler.NewAuthHandler(panickingAuthService{}).Login)

	var captured bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&captured)
	defer log.SetOutput(previousOutput)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{}`)))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d, want 500: %s", response.Code, response.Body)
	}
	if strings.Contains(response.Body.String(), "private auth panic detail") {
		t.Fatalf("auth panic response leaked internal text: %s", response.Body)
	}
	if !strings.Contains(response.Body.String(), `"code":"internal_server_error"`) {
		t.Fatalf("panic response is not the standard JSON error: %s", response.Body)
	}
	if strings.Contains(captured.String(), "private auth panic detail") {
		t.Fatalf("auth panic log leaked internal text: %s", captured.String())
	}
}

type hashCapturingUserRepository struct {
	repository.UserRepository
	passwordHash string
}

func (repo *hashCapturingUserRepository) Create(ctx context.Context, user *model.User) error {
	repo.passwordHash = user.PasswordHash
	return repo.UserRepository.Create(ctx, user)
}

type panickingAuthService struct{}

func (panickingAuthService) Register(context.Context, model.RegisterRequest) (model.UserResponse, error) {
	panic("private auth panic detail")
}

func (panickingAuthService) Login(context.Context, model.LoginRequest) (model.TokenResponse, error) {
	panic("private auth panic detail")
}
