#!/usr/bin/env bash
# Repeatable K3s smoke test: Traefik ingress, 2 replicas, local-path persistence,
# and Docker (Floci) coexistence. Deletes the k3s-smoke namespace at the end.
set -euo pipefail
cd "$(dirname "$0")"
export KUBECONFIG=${KUBECONFIG:-$HOME/.kube/config}
NS=k3s-smoke
trap 'kubectl delete namespace "$NS" --ignore-not-found --wait=true >/dev/null' EXIT

section() { printf '\n== %s\n' "$*"; }

section "apply workload (Deployment x2, Service, Ingress /smoke, PVC)"
kubectl apply -f k8s/smoke/smoke.yaml
kubectl -n "$NS" rollout status deploy/web --timeout=180s
kubectl -n "$NS" get deploy,pods,svc,ingress -o wide

section "curl http://localhost/smoke through Traefik (6 requests)"
# Traefik needs a few seconds to pick up new endpoints ("no available server" until then).
distinct() { for _ in $(seq 1 20); do curl -s http://localhost/smoke; done | grep 'k3s-smoke ok' | sort -u | wc -l; }
for _ in $(seq 1 60); do [ "$(distinct)" -ge 2 ] && break; sleep 1; done
for _ in $(seq 1 6); do curl -s -w '  HTTP %{http_code}\n' http://localhost/smoke; done
echo "responses per pod over 20 requests:"
for _ in $(seq 1 20); do curl -s http://localhost/smoke; done | sort | uniq -c
served=$(distinct)
echo "distinct pods that answered: $served (want 2)"
[ "$served" -eq 2 ]

section "local-path PVC persists across pod recreation"
kubectl apply -f k8s/smoke/pvc-pod.yaml
kubectl -n "$NS" wait --for=condition=Ready pod/pvc-check --timeout=180s
stamp="written at $(date -u +%FT%TZ)"
kubectl -n "$NS" exec pvc-check -- sh -c "echo '$stamp' > /data/proof.txt && cat /data/proof.txt"
kubectl -n "$NS" delete pod pvc-check --wait=true
kubectl apply -f k8s/smoke/pvc-pod.yaml
kubectl -n "$NS" wait --for=condition=Ready pod/pvc-check --timeout=180s
got=$(kubectl -n "$NS" exec pvc-check -- cat /data/proof.txt)
echo "after recreate: $got"
[ "$got" = "$stamp" ]
kubectl -n "$NS" get pvc data
kubectl get pv -o custom-columns=NAME:.metadata.name,CLAIM:.spec.claimRef.name,CLASS:.spec.storageClassName,PATH:.spec.local.path,HOSTPATH:.spec.hostPath.path

section "Docker coexistence (Floci in Docker, K3s on its own containerd)"
docker ps --format '{{.Names}}  {{.Image}}  {{.Ports}}' | grep -i floci || echo "Floci container not running"
echo "Docker published port 4566 from the VM: HTTP $(curl -s -o /dev/null -w '%{http_code}' http://localhost:4566/_aws/ses)"
echo "iptables chains: DOCKER=$(sudo iptables -S | grep -c -- '-N DOCKER') KUBE=$(sudo iptables -S | grep -c -- '-N KUBE') CNI=$(sudo iptables -t nat -S | grep -c -- 'CNI-\|FLANNEL')"
echo "K3s containerd socket: $(sudo test -S /run/k3s/containerd/containerd.sock && echo present); Docker socket: $(test -S /var/run/docker.sock && echo present)"

section "cleanup"
echo "deleting namespace $NS"
