package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

// userCreatedEvent mirrors internal/webhook.UserCreatedEvent in the app module.
type userCreatedEvent struct {
	Event string `json:"event"`
	User  struct {
		ID    uint64 `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	} `json:"user"`
}

type Email struct {
	To, Subject, Text string
}

type Mailer interface {
	Send(ctx context.Context, message Email) (string, error)
}

type Handler struct {
	Secret []byte
	Mailer Mailer
	MaxAge time.Duration
	Now    func() time.Time
}

func (handler *Handler) Handle(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	if method := request.RequestContext.HTTP.Method; method != "" && method != http.MethodPost {
		return reply(http.StatusMethodNotAllowed, "method not allowed"), nil
	}
	body := []byte(request.Body)
	if request.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			return reply(http.StatusBadRequest, "invalid body encoding"), nil
		}
		body = decoded
	}
	if !handler.verify(header(request.Headers, "X-Webhook-Timestamp"), header(request.Headers, "X-Webhook-Signature"), body) {
		return reply(http.StatusUnauthorized, "invalid signature"), nil
	}

	var event userCreatedEvent
	if err := json.Unmarshal(body, &event); err != nil || event.Event != "user.created" {
		return reply(http.StatusBadRequest, "unsupported event"), nil
	}
	if _, err := mail.ParseAddress(event.User.Email); err != nil || event.User.ID == 0 {
		return reply(http.StatusBadRequest, "invalid user"), nil
	}

	messageID, err := handler.Mailer.Send(ctx, confirmationEmail(event))
	if err != nil {
		log.Printf("send confirmation for user %d failed: %v", event.User.ID, err)
		return reply(http.StatusBadGateway, "email not sent"), nil
	}
	log.Printf("confirmation for user %d sent, message id %s", event.User.ID, messageID)
	return reply(http.StatusAccepted, "email sent"), nil
}

// verify checks "sha256=<hex HMAC-SHA256 of timestamp.body>" and rejects stale timestamps (replay window).
func (handler *Handler) verify(timestamp, signature string, body []byte) bool {
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	age := handler.Now().Sub(time.Unix(seconds, 0))
	if age > handler.MaxAge || age < -handler.MaxAge {
		return false
	}
	got, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil || !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, handler.Secret)
	mac.Write([]byte(timestamp + "."))
	mac.Write(body)
	return hmac.Equal(got, mac.Sum(nil))
}

func confirmationEmail(event userCreatedEvent) Email {
	name := event.User.Name
	if name == "" {
		name = event.User.Email
	}
	return Email{
		To:      event.User.Email,
		Subject: "Your Movies account is ready",
		Text: fmt.Sprintf("Hello %s,\n\nAn administrator created a Movies account for %s with the %s role.\n"+
			"You can now sign in.\n\nThis is a mock registration confirmation from the local lab.\n",
			name, event.User.Email, event.User.Role),
	}
}

// header looks a name up case-insensitively; Function URLs lowercase header names.
func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

func reply(status int, message string) events.LambdaFunctionURLResponse {
	body, _ := json.Marshal(map[string]string{"message": message})
	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}
}
