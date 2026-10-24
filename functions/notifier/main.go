// Command notifier is a Lambda function behind a Function URL. It receives the
// movies app's signed "user.created" webhook and sends a mock registration
// confirmation email through SES (Floci in the lab).
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type sesMailer struct {
	client *sesv2.Client
	sender string
}

func (mailer sesMailer) Send(ctx context.Context, message Email) (string, error) {
	output, err := mailer.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(mailer.sender),
		Destination:      &types.Destination{ToAddresses: []string{message.To}},
		Content: &types.EmailContent{Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(message.Subject), Charset: aws.String("UTF-8")},
			Body:    &types.Body{Text: &types.Content{Data: aws.String(message.Text), Charset: aws.String("UTF-8")}},
		}},
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(output.MessageId), nil
}

func main() {
	secret := os.Getenv("WEBHOOK_SECRET")
	sender := os.Getenv("SENDER_EMAIL")
	if len(secret) < 32 || sender == "" {
		log.Fatal(errors.New("WEBHOOK_SECRET (>= 32 bytes) and SENDER_EMAIL must be set"))
	}
	// The default chain honours AWS_ENDPOINT_URL, which Floci injects into Lambda containers.
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}
	handler := &Handler{
		Secret: []byte(secret),
		Mailer: sesMailer{client: sesv2.NewFromConfig(cfg), sender: sender},
		MaxAge: 5 * time.Minute,
		Now:    time.Now,
	}
	lambda.Start(handler.Handle)
}
