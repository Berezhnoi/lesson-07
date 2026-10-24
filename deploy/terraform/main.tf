data "aws_iam_policy_document" "assume_lambda" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "send_email" {
  statement {
    actions   = ["ses:SendEmail"]
    resources = ["*"]
  }
}

resource "aws_iam_role" "notifier" {
  name               = "${var.function_name}-role"
  assume_role_policy = data.aws_iam_policy_document.assume_lambda.json
}

# Least privilege: the function may only send email.
resource "aws_iam_role_policy" "send_email" {
  name   = "send-email"
  role   = aws_iam_role.notifier.id
  policy = data.aws_iam_policy_document.send_email.json
}

resource "aws_ses_email_identity" "sender" {
  email = var.sender_email
}

data "archive_file" "notifier" {
  type        = "zip"
  source_file = var.bootstrap_path
  output_path = "${path.module}/build/notifier.zip"
}

resource "aws_lambda_function" "notifier" {
  function_name    = var.function_name
  role             = aws_iam_role.notifier.arn
  runtime          = "provided.al2023"
  architectures    = ["x86_64"]
  handler          = "bootstrap"
  filename         = data.archive_file.notifier.output_path
  source_code_hash = data.archive_file.notifier.output_base64sha256
  timeout          = 10
  memory_size      = 128

  environment {
    variables = {
      SENDER_EMAIL   = aws_ses_email_identity.sender.email
      WEBHOOK_SECRET = var.webhook_secret
    }
  }
}

# Public URL; callers are authenticated by the HMAC signature, not IAM.
resource "aws_lambda_function_url" "notifier" {
  function_name      = aws_lambda_function.notifier.function_name
  authorization_type = "NONE"
}
