# Floci notes (local AWS emulator)

Version used: `floci/floci:2.2.0` (released 2026-10-06; pinned in `infra/floci/docker-compose.yml`).
Sources: <https://github.com/floci-io/floci>, <https://floci.io/floci/services/lambda/>,
<https://floci.io/floci/services/ses/>, `docs/getting-started/terraform.md` in the Floci repo.
"Verified" below means checked against the running emulator in the lab VM on 2026-10-24.

## Running

| Fact | Source | Status |
|---|---|---|
| All services on one endpoint, `http://localhost:4566` | README | Verified |
| Any non-empty credentials work (`test`/`test`) unless strict auth is enabled | README | Verified |
| Lambda runs real Docker containers, so Floci needs `/var/run/docker.sock` (and runs as root) | README, Lambda docs | Verified: a `floci-aws-<function>-*` container appears on the first invoke |
| With `FLOCI_HOSTNAME`, Lambda containers join the Compose network and receive `AWS_ENDPOINT_URL` pointing at Floci | Lambda docs | Verified indirectly: the function's SES call reaches Floci with no endpoint set in our code |

## Lambda

- Any runtime with an official AWS Lambda image works, including `provided.al2023`. A ZIP-based `provided.*`
  function runs `/var/task/bootstrap`. Go is built as `bootstrap` (linux/amd64, `-tags lambda.norpc`). Verified.
- **Cold start:** the first invoke pulls `public.ecr.aws/lambda/provided:al2023` and took about 23 seconds.
  The app's default `WEBHOOK_TIMEOUT` of 5 seconds would drop that first event, so K3s sets `60s`.
- **Function URLs:** Terraform's `aws_lambda_function_url` returns
  `http://<id>.lambda-url.us-east-1.floci:4566/`. That hostname is not in DNS. The docs only say Function URLs
  are "reachable directly on `/{proxy:.*}` under the Lambda URL controller", without a hostname format.
  Verified: Floci routes the request by the `Host` header, so `curl -H "Host: <id>.lambda-url.us-east-1.floci" http://localhost:4566/`
  invokes the function, and the request arrives as a Function URL event (lowercased headers, HTTP method set).
  - **Workaround (smallest):** a K3s CoreDNS custom zone (`infra/k8s/coredns-custom.yaml.tmpl`) resolves
    `*.floci` to the node IP, where port 4566 is published. The app uses the Terraform output URL unchanged.
    No app code change is needed.
- Not implemented in Floci (from the docs): `InvokeWithResponseStream` and layer permissions. Neither is used here.
- IAM: role and policy are stored. Policies are enforced only with `FLOCI_SERVICES_IAM_ENFORCEMENT_ENABLED=true`,
  which is off here, so least-privilege is declared but not proven locally.

## SES

- v1 and v2 APIs share identities and sent messages. Floci has no sandbox and stores every email locally.
- Inspect the mailbox: `GET /_aws/ses` (optional `?id=` or `?email=<sender>`). `DELETE /_aws/ses` clears it.
  Verified (`make -C infra mailbox`).
- **Difference from the docs:** the docs say sends check that the sender is verified. In 2.2.0, a v1 `SendEmail`
  from an unverified address was accepted. Terraform still creates `aws_ses_email_identity`, as AWS requires.
- No SMTP relay is configured (`FLOCI_SERVICES_SES_SMTP_HOST` unset), so mail is stored only and nothing leaves the VM.

## Terraform

- The standard `hashicorp/aws` provider works with `skip_credentials_validation`, `skip_metadata_api_check`,
  `skip_requesting_account_id`, and **an endpoint for every service used**. Here that is iam, lambda, ses,
  sesv2 and sts. A service without an endpoint entry would go to real AWS, so `floci_endpoint` is validated
  to be local.
- State is local (`infra/terraform/terraform.tfstate`, gitignored). It contains the webhook secret, because
  the secret is a Lambda environment variable. Never commit state.
