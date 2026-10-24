// Package webhook notifies an external endpoint when an admin creates a user.
//
// It is wired from cmd/api only: an http.Handler wrapper marks requests to the
// admin-only web registration route, and a UserRepository decorator sends a
// "user.created" event after a marked Create succeeds. Existing packages are
// not modified, and the notification never changes the HTTP response.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"homework/internal/model"
	"homework/internal/repository"
)

const (
	EventUserCreated = "user.created"

	SignatureHeader = "X-Webhook-Signature"
	TimestampHeader = "X-Webhook-Timestamp"

	defaultTimeout = 5 * time.Second
	minSecretBytes = 32
)

// Config holds the webhook settings. An empty URL disables notifications.
type Config struct {
	URL     string
	Secret  string
	Timeout time.Duration
}

// ConfigFromEnv reads WEBHOOK_URL, WEBHOOK_SECRET and WEBHOOK_TIMEOUT.
// Call it after config.Load so values from .env are already in the environment.
func ConfigFromEnv() (Config, error) {
	cfg := Config{URL: os.Getenv("WEBHOOK_URL"), Secret: os.Getenv("WEBHOOK_SECRET"), Timeout: defaultTimeout}
	if cfg.URL == "" {
		return cfg, nil
	}
	parsed, err := url.Parse(cfg.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return Config{}, errors.New("WEBHOOK_URL must be an absolute http(s) URL")
	}
	if len(cfg.Secret) < minSecretBytes {
		return Config{}, fmt.Errorf("WEBHOOK_SECRET must be at least %d bytes when WEBHOOK_URL is set", minSecretBytes)
	}
	if raw := os.Getenv("WEBHOOK_TIMEOUT"); raw != "" {
		cfg.Timeout, err = time.ParseDuration(raw)
		if err != nil || cfg.Timeout <= 0 {
			return Config{}, errors.New("WEBHOOK_TIMEOUT must be a positive duration")
		}
	}
	return cfg, nil
}

// UserCreatedEvent is the JSON body sent to the webhook. It never contains credentials.
type UserCreatedEvent struct {
	Event      string             `json:"event"`
	OccurredAt time.Time          `json:"occurred_at"`
	User       model.UserResponse `json:"user"`
}

// Client posts signed events to the configured URL.
type Client struct {
	url    string
	secret []byte
	http   *http.Client
	now    func() time.Time
}

func NewClient(cfg Config) *Client {
	return &Client{
		url:    cfg.URL,
		secret: []byte(cfg.Secret),
		http:   &http.Client{Timeout: cfg.Timeout},
		now:    time.Now,
	}
}

// Send posts one event. The signature is hex HMAC-SHA256 over "<timestamp>.<body>".
func (client *Client) Send(ctx context.Context, event UserCreatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode webhook event: %w", err)
	}
	timestamp := strconv.FormatInt(client.now().Unix(), 10)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(TimestampHeader, timestamp)
	request.Header.Set(SignatureHeader, "sha256="+Sign(client.secret, timestamp, body))

	response, err := client.http.Do(request)
	if err != nil {
		return fmt.Errorf("send webhook request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("webhook responded with status %d", response.StatusCode)
	}
	return nil
}

// Sign returns the hex HMAC-SHA256 of "<timestamp>.<body>"; receivers recompute it to verify.
func Sign(secret []byte, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

type triggerKey struct{}

// MarkAdminRegistration marks POST /register requests. That web route is
// registered behind CSRF and RequireRole("admin"), so a user created while
// handling a marked request was created by an admin. Public API registration
// (/api/v1/auth/register) is not marked and sends no event.
func MarkAdminRegistration(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/register" {
			r = r.WithContext(context.WithValue(r.Context(), triggerKey{}, true))
		}
		next.ServeHTTP(w, r)
	})
}

func isMarked(ctx context.Context) bool {
	marked, _ := ctx.Value(triggerKey{}).(bool)
	return marked
}

type sender interface {
	Send(ctx context.Context, event UserCreatedEvent) error
}

// NotifyingUsers decorates a UserRepository and sends a user.created event
// asynchronously after a successful Create on a marked request.
type NotifyingUsers struct {
	repository.UserRepository
	sender  sender
	timeout time.Duration
	now     func() time.Time
	// done is called after each delivery attempt; tests use it to wait.
	done func()
}

func NewNotifyingUsers(users repository.UserRepository, client *Client, timeout time.Duration) *NotifyingUsers {
	return &NotifyingUsers{UserRepository: users, sender: client, timeout: timeout, now: time.Now}
}

func (users *NotifyingUsers) Create(ctx context.Context, user *model.User) error {
	if err := users.UserRepository.Create(ctx, user); err != nil {
		return err
	}
	if !isMarked(ctx) {
		return nil
	}
	event := UserCreatedEvent{
		Event:      EventUserCreated,
		OccurredAt: users.now().UTC(),
		User: model.UserResponse{
			ID: user.ID, Email: user.Email, Name: user.Name, Role: user.Role, CreatedAt: user.CreatedAt,
		},
	}
	// Detach from the request so the redirect is not delayed or cancelled by delivery.
	go func(ctx context.Context) {
		if users.done != nil {
			defer users.done()
		}
		ctx, cancel := context.WithTimeout(ctx, users.timeout)
		defer cancel()
		if err := users.sender.Send(ctx, event); err != nil {
			log.Printf("webhook %s for user %d failed: %v", EventUserCreated, event.User.ID, err)
			return
		}
		log.Printf("webhook %s for user %d delivered", EventUserCreated, event.User.ID)
	}(context.WithoutCancel(ctx))
	return nil
}
