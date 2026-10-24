variable "floci_endpoint" {
  description = "Floci endpoint inside the VM. Must stay a local address; real AWS is never used."
  type        = string
  default     = "http://localhost:4566"

  validation {
    condition     = can(regex("^http://(localhost|127\\.0\\.0\\.1|floci)(:[0-9]+)?$", var.floci_endpoint))
    error_message = "floci_endpoint must point at the local Floci emulator."
  }
}

variable "region" {
  type    = string
  default = "us-east-1"
}

variable "function_name" {
  type    = string
  default = "movies-user-created-notifier"
}

variable "sender_email" {
  description = "SES sender identity for the mock confirmation email."
  type        = string
  default     = "no-reply@movies.local"
}

variable "webhook_secret" {
  description = "Shared HMAC secret; must equal the app's WEBHOOK_SECRET. Pass via TF_VAR_webhook_secret or terraform.tfvars (gitignored)."
  type        = string
  sensitive   = true

  validation {
    condition     = length(var.webhook_secret) >= 32
    error_message = "webhook_secret must be at least 32 bytes."
  }
}

variable "bootstrap_path" {
  description = "Linux/amd64 bootstrap binary built by `make -C functions/notifier build`."
  type        = string
  default     = "../../functions/notifier/build/bootstrap"
}
