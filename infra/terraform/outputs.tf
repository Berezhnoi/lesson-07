output "function_name" {
  value = aws_lambda_function.notifier.function_name
}

output "function_url" {
  description = "URL returned by Floci. See docs/floci-notes.md for how the app reaches it."
  value       = aws_lambda_function_url.notifier.function_url
}

output "sender_email" {
  value = aws_ses_email_identity.sender.email
}
