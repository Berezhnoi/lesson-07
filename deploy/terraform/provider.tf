# Floci only: dummy credentials, no account lookups, every used service pinned to the emulator.
provider "aws" {
  region     = var.region
  access_key = "test"
  secret_key = "test"

  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  endpoints {
    iam    = var.floci_endpoint
    lambda = var.floci_endpoint
    ses    = var.floci_endpoint
    sesv2  = var.floci_endpoint
    sts    = var.floci_endpoint
  }
}
