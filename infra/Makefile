# Local cloud lab, run inside the VM: Floci (AWS emulator) + Terraform + K3s.
# Never points at real AWS: dummy credentials, Floci endpoint only.
SHELL := /bin/bash
ROOT := $(abspath ..)
SECRETS := $(CURDIR)/secrets.env
FLOCI := http://localhost:4566
KUBECTL := KUBECONFIG=$(HOME)/.kube/config kubectl
NS := movies
NODE_IP = $(shell $(KUBECTL) get node -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
AWS := AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 AWS_ENDPOINT_URL=$(FLOCI) aws

.PHONY: all secrets floci-up floci-down lambda-build lambda-test tf-apply tf-destroy image \
	k8s-secrets k8s-webhook k8s-dns k8s-apply e2e mailbox clean-mailbox

all: secrets floci-up lambda-build tf-apply image k8s-apply

secrets: ## Generate infra/secrets.env (gitignored, mode 600) once; values are never printed.
	@if [ -f $(SECRETS) ]; then echo "secrets.env exists"; else \
	  umask 077; { \
	    echo "WEBHOOK_SECRET=$$(openssl rand -base64 48 | tr -d '\n/+=')"; \
	    echo "JWT_SECRET=$$(openssl rand -base64 48 | tr -d '\n/+=')"; \
	    echo "DB_PASSWORD=$$(openssl rand -hex 16)"; \
	    echo "ADMIN_PASSWORD=$$(openssl rand -hex 16)"; \
	  } > $(SECRETS); echo "generated secrets.env"; fi

floci-up:
	docker compose -f floci/docker-compose.yml up -d --wait

floci-down:
	docker compose -f floci/docker-compose.yml down

lambda-test:
	cd $(ROOT)/functions/notifier && gofmt -l . | (! grep .) && go vet ./... && go test -count=1 ./...

lambda-build: lambda-test
	cd $(ROOT)/functions/notifier && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	  go build -tags lambda.norpc -trimpath -ldflags "-s -w" -o build/bootstrap .

tf-apply: lambda-build
	cd terraform && terraform fmt -check && terraform init -input=false >/dev/null && terraform validate && \
	  TF_VAR_webhook_secret="$$(grep '^WEBHOOK_SECRET=' $(SECRETS) | cut -d= -f2-)" terraform apply -input=false -auto-approve

tf-destroy:
	cd terraform && TF_VAR_webhook_secret="$$(grep '^WEBHOOK_SECRET=' $(SECRETS) | cut -d= -f2-)" terraform destroy -input=false -auto-approve

image: ## Build the lab image and import it into K3s containerd (no registry).
	docker build -f k8s/Dockerfile -t movies-api:lab $(ROOT)
	docker save movies-api:lab | sudo k3s ctr images import -

k8s-secrets:
	$(KUBECTL) apply -f k8s/namespace.yaml
	@$(KUBECTL) -n $(NS) create secret generic movies-secrets --from-env-file=$(SECRETS) \
	  --dry-run=client -o yaml | $(KUBECTL) apply -f - >/dev/null && echo "secret/movies-secrets applied"

k8s-webhook:
	$(KUBECTL) -n $(NS) create configmap movies-webhook \
	  --from-literal=WEBHOOK_URL="$$(cd terraform && terraform output -raw function_url)" \
	  --dry-run=client -o yaml | $(KUBECTL) apply -f -

k8s-dns:
	NODE_IP=$(NODE_IP) envsubst '$${NODE_IP}' < k8s/coredns-custom.yaml.tmpl | $(KUBECTL) apply -f -
	$(KUBECTL) -n kube-system rollout restart deployment/coredns
	$(KUBECTL) -n kube-system rollout status deployment/coredns --timeout=120s

k8s-apply: k8s-secrets k8s-webhook k8s-dns
	$(KUBECTL) apply -f k8s/config.yaml -f k8s/postgres.yaml
	$(KUBECTL) -n $(NS) create configmap movies-migrations --from-file=$(ROOT)/migrations \
	  --dry-run=client -o yaml | $(KUBECTL) apply -f -
	$(KUBECTL) -n $(NS) rollout status statefulset/postgres --timeout=180s
	$(KUBECTL) -n $(NS) delete job movies-migrate movies-seed-admin --ignore-not-found
	$(KUBECTL) apply -f k8s/jobs.yaml
	$(KUBECTL) -n $(NS) wait --for=condition=complete job/movies-migrate --timeout=180s
	$(KUBECTL) -n $(NS) wait --for=condition=complete job/movies-seed-admin --timeout=180s
	$(KUBECTL) apply -f k8s/app.yaml
	$(KUBECTL) -n $(NS) rollout restart deployment/movies-api
	$(KUBECTL) -n $(NS) rollout status deployment/movies-api --timeout=180s

e2e: ## Admin creates a user through the web UI on K3s, then the email must appear in Floci SES.
	ADMIN_PASSWORD="$$(grep '^ADMIN_PASSWORD=' $(SECRETS) | cut -d= -f2-)" ./e2e.sh http://localhost:30080 $(FLOCI)

mailbox:
	@curl -s $(FLOCI)/_aws/ses | python3 -m json.tool

clean-mailbox:
	curl -s -X DELETE $(FLOCI)/_aws/ses -o /dev/null -w "%{http_code}\n"
