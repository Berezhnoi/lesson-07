package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

const testSecret = "notifier-test-secret-with-more-than-32-bytes"

var fixedNow = time.Unix(1_800_000_000, 0)

type fakeMailer struct {
	sent []Email
	err  error
}

func (mailer *fakeMailer) Send(_ context.Context, message Email) (string, error) {
	if mailer.err != nil {
		return "", mailer.err
	}
	mailer.sent = append(mailer.sent, message)
	return "msg-1", nil
}

func sign(timestamp, body string) string {
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write([]byte(timestamp + "." + body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func signedRequest(body string, at time.Time) events.LambdaFunctionURLRequest {
	timestamp := strconv.FormatInt(at.Unix(), 10)
	request := events.LambdaFunctionURLRequest{
		Body:    body,
		Headers: map[string]string{"x-webhook-timestamp": timestamp, "x-webhook-signature": sign(timestamp, body)},
	}
	request.RequestContext.HTTP.Method = http.MethodPost
	return request
}

func newHandler(mailer Mailer) *Handler {
	return &Handler{Secret: []byte(testSecret), Mailer: mailer, MaxAge: 5 * time.Minute, Now: func() time.Time { return fixedNow }}
}

const validBody = `{"event":"user.created","occurred_at":"2027-01-15T08:00:00Z","user":{"id":7,"email":"new@example.com","name":"New User","role":"viewer","created_at":"2027-01-15T08:00:00Z"}}`

func TestValidEventSendsConfirmation(t *testing.T) {
	mailer := &fakeMailer{}
	response, err := newHandler(mailer).Handle(context.Background(), signedRequest(validBody, fixedNow))
	if err != nil || response.StatusCode != http.StatusAccepted {
		t.Fatalf("response = %d, %v; want 202", response.StatusCode, err)
	}
	if len(mailer.sent) != 1 || mailer.sent[0].To != "new@example.com" ||
		!strings.Contains(mailer.sent[0].Text, "Hello New User") || !strings.Contains(mailer.sent[0].Text, "viewer") {
		t.Fatalf("sent = %+v", mailer.sent)
	}
}

func TestBase64BodyIsDecoded(t *testing.T) {
	mailer := &fakeMailer{}
	request := signedRequest(validBody, fixedNow)
	request.Body = base64.StdEncoding.EncodeToString([]byte(validBody))
	request.IsBase64Encoded = true
	if response, _ := newHandler(mailer).Handle(context.Background(), request); response.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d; want 202", response.StatusCode)
	}
}

func TestRejectedRequestsSendNothing(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*events.LambdaFunctionURLRequest)
		want   int
	}{
		{"wrong method", func(r *events.LambdaFunctionURLRequest) { r.RequestContext.HTTP.Method = http.MethodGet }, http.StatusMethodNotAllowed},
		{"missing signature", func(r *events.LambdaFunctionURLRequest) { delete(r.Headers, "x-webhook-signature") }, http.StatusUnauthorized},
		{"tampered body", func(r *events.LambdaFunctionURLRequest) { r.Body = strings.Replace(r.Body, "new@", "evil@", 1) }, http.StatusUnauthorized},
		{"wrong secret", func(r *events.LambdaFunctionURLRequest) {
			r.Headers["x-webhook-signature"] = "sha256=" + strings.Repeat("0", 64)
		}, http.StatusUnauthorized},
		{"stale timestamp", func(r *events.LambdaFunctionURLRequest) {
			*r = signedRequest(validBody, fixedNow.Add(-10*time.Minute))
		}, http.StatusUnauthorized},
		{"other event", func(r *events.LambdaFunctionURLRequest) {
			*r = signedRequest(strings.Replace(validBody, "user.created", "user.deleted", 1), fixedNow)
		}, http.StatusBadRequest},
		{"bad email", func(r *events.LambdaFunctionURLRequest) {
			*r = signedRequest(strings.Replace(validBody, "new@example.com", "not-an-email", 1), fixedNow)
		}, http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mailer := &fakeMailer{}
			request := signedRequest(validBody, fixedNow)
			test.mutate(&request)
			response, err := newHandler(mailer).Handle(context.Background(), request)
			if err != nil || response.StatusCode != test.want {
				t.Fatalf("response = %d, %v; want %d", response.StatusCode, err, test.want)
			}
			if len(mailer.sent) != 0 {
				t.Fatal("email sent for a rejected request")
			}
		})
	}
}

func TestMailerFailureReturns502WithoutLeakingError(t *testing.T) {
	mailer := &fakeMailer{err: errors.New("ses internal detail")}
	response, _ := newHandler(mailer).Handle(context.Background(), signedRequest(validBody, fixedNow))
	if response.StatusCode != http.StatusBadGateway || strings.Contains(response.Body, "internal detail") {
		t.Fatalf("response = %d %q", response.StatusCode, response.Body)
	}
}
