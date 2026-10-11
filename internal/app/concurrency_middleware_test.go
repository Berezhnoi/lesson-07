package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"homework/internal/middleware"
)

func TestConcurrentCreatesHaveSequentialIDs(t *testing.T) {
	e := newEnv(t)
	const count = 100
	ids := make(chan uint32, count)
	var waitGroup sync.WaitGroup

	for index := 0; index < count; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			body, err := json.Marshal(e.payload(fmt.Sprintf("parallel-%d", index), "Drama"))
			if err != nil {
				t.Errorf("encode request: %v", err)
				return
			}
			response, panicked := e.try(http.MethodPost, e.base(), string(body))
			if panicked != nil || response.Code != http.StatusCreated {
				t.Errorf("POST: status=%d panic=%v body=%s", response.Code, panicked, response.Body)
				return
			}
			var created struct {
				ID uint32 `json:"id"`
			}
			if err := json.Unmarshal(response.Body, &created); err != nil {
				t.Errorf("decode response: %v", err)
				return
			}
			ids <- created.ID
		}(index)
	}
	waitGroup.Wait()
	close(ids)

	got := make([]int, 0, count)
	for id := range ids {
		got = append(got, int(id))
	}
	sort.Ints(got)
	if len(got) != count {
		t.Fatalf("got %d successful creates, want %d", len(got), count)
	}
	for index, id := range got {
		if id != index+1 {
			t.Fatalf("created ids must be sequential without gaps: got %v", got)
		}
	}
}

func TestMixedConcurrentOperations(t *testing.T) {
	e := newEnv(t)
	const workers = 8
	const iterations = 12
	var waitGroup sync.WaitGroup

	for worker := 0; worker < workers; worker++ {
		waitGroup.Add(1)
		go func(worker int) {
			defer waitGroup.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				tag := fmt.Sprintf("worker-%d-item-%d", worker, iteration)
				body, err := json.Marshal(e.payload(tag, "Drama"))
				if err != nil {
					t.Errorf("encode create: %v", err)
					return
				}
				created, panicked := e.try(http.MethodPost, e.base(), string(body))
				if panicked != nil || created.Code != http.StatusCreated {
					t.Errorf("POST: status=%d panic=%v", created.Code, panicked)
					return
				}
				var movie struct {
					ID uint32 `json:"id"`
				}
				if err := json.Unmarshal(created.Body, &movie); err != nil {
					t.Errorf("decode create: %v", err)
					return
				}

				getResponse, panicked := e.try(http.MethodGet, e.item(movie.ID), "")
				if panicked != nil || getResponse.Code != http.StatusOK {
					t.Errorf("GET: status=%d panic=%v", getResponse.Code, panicked)
					return
				}
				updatedBody, err := json.Marshal(e.payload(tag+"-updated", "Drama"))
				if err != nil {
					t.Errorf("encode update: %v", err)
					return
				}
				putResponse, panicked := e.try(http.MethodPut, e.item(movie.ID), string(updatedBody))
				if panicked != nil || putResponse.Code != http.StatusOK {
					t.Errorf("PUT: status=%d panic=%v", putResponse.Code, panicked)
					return
				}
				listResponse, panicked := e.try(http.MethodGet, e.base()+"?limit=100", "")
				if panicked != nil || listResponse.Code != http.StatusOK {
					t.Errorf("list: status=%d panic=%v", listResponse.Code, panicked)
					return
				}
				deleteResponse, panicked := e.try(http.MethodDelete, e.item(movie.ID), "")
				if panicked != nil || deleteResponse.Code != http.StatusNoContent || len(deleteResponse.Body) != 0 {
					t.Errorf("DELETE: status=%d panic=%v body=%s", deleteResponse.Code, panicked, deleteResponse.Body)
					return
				}
			}
		}(worker)
	}
	waitGroup.Wait()
}

func TestRecoveryResponseIsGenericAndHasCORS(t *testing.T) {
	t.Setenv("ALLOWED_ORIGIN", "https://allowed.example")
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.CORS(), middleware.Recovery(), middleware.RequestLogger())
	router.GET("/panic", func(*gin.Context) {
		panic("sensitive panic detail")
	})

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("panic response status = %d, want 500", response.Code)
	}
	if strings.Contains(response.Body.String(), "sensitive panic detail") {
		t.Errorf("panic response leaked panic text: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"internal_server_error"`) || !strings.Contains(response.Body.String(), `"message":"internal server error"`) {
		t.Errorf("unexpected panic response: %s", response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://allowed.example" {
		t.Errorf("Access-Control-Allow-Origin = %q, want configured origin", got)
	}
}

func TestPreflightWorksForUnknownAndItemPaths(t *testing.T) {
	e := newEnv(t)
	for _, path := range []string{"/unknown", e.base() + "/123"} {
		request := httptest.NewRequest(http.MethodOptions, path, nil)
		response, panicked := e.tryReq(request)
		if panicked != nil {
			t.Fatalf("OPTIONS %s panicked: %v", path, panicked)
		}
		if response.Code != http.StatusNoContent || len(response.Body) != 0 {
			t.Errorf("OPTIONS %s: status=%d body=%q", path, response.Code, response.Body)
		}
		if response.Header.Get("Access-Control-Allow-Origin") == "" {
			t.Errorf("OPTIONS %s missing allow-origin", path)
		}
		if got := response.Header.Get("Access-Control-Allow-Methods"); got != "GET,POST,PUT,DELETE,OPTIONS" {
			t.Errorf("OPTIONS %s methods = %q", path, got)
		}
		if got := response.Header.Get("Access-Control-Allow-Headers"); got != "Content-Type,Authorization" {
			t.Errorf("OPTIONS %s headers = %q", path, got)
		}
	}
}

func TestRoutersHaveIndependentStorage(t *testing.T) {
	first := newEnv(t)
	second := newEnv(t)
	first.create(first.valid())

	response := second.do(http.MethodGet, second.item(1), "")
	second.wantError(response, http.StatusNotFound, "not_found")
}

func TestCORSHeaderOnErrorResponses(t *testing.T) {
	e := newEnv(t)
	responses := []resp{
		e.do(http.MethodGet, "/unknown", ""),
		e.do(http.MethodPost, e.base(), `{}`),
	}
	for _, response := range responses {
		if response.Header.Get("Access-Control-Allow-Origin") == "" {
			t.Errorf("status %d response is missing Access-Control-Allow-Origin", response.Code)
		}
	}
}
