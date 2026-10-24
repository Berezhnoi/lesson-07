# K3s notes (single node inside the lab VM)

## Install

Version: **v1.36.5+k3s1**. This is what the K3s `stable` channel resolved to on 2026-10-24; it was released
2026-09-30. Bundled components are kept: the **Traefik** ingress (default IngressClass `traefik`) and the
**local-path** storage class (default, `WaitForFirstConsumer`).

```sh
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION='v1.36.5+k3s1' \
  INSTALL_K3S_EXEC='server --write-kubeconfig-mode 640 --write-kubeconfig-group vagrant' sh -
```

`infra/vm/Vagrantfile` runs the same command during provisioning. Running the installer again with different
`INSTALL_K3S_EXEC` flags changes the server flags and keeps the cluster data. That is how Traefik was
re-enabled after the earlier `--disable traefik` install.

## kubectl for the non-root user

- `/etc/rancher/k3s/k3s.yaml` is mode 640, group `vagrant` (`--write-kubeconfig-group`), so `kubectl` works in
  every shell (login, non-interactive, scripts) without `KUBECONFIG`. Otherwise the `kubectl` symlink to `k3s`
  reads the root-only file and fails with "Unable to read /etc/rancher/k3s/k3s.yaml … permission denied".
  The vagrant user already has passwordless sudo, so this grants nothing new.
- A copy also lives at `~/.kube/config` (mode 600); `infra/Makefile` uses it explicitly.
- Completion: `source <(kubectl completion bash)`, plus `alias k=kubectl` with `complete -o default -F __start_kubectl k`.
  This is in `~/.bashrc` and, for new VMs, in `/etc/profile.d/kubeconfig.sh`.
- The kubeconfig holds cluster-admin client credentials. Never commit or print it; `infra/.gitignore` ignores `kubeconfig*`.

## Coexistence with Docker

- K3s uses its **own containerd** (`/run/k3s/containerd/containerd.sock`), separate from Docker's
  (`/var/run/docker.sock`). Images built with Docker must be imported with `docker save … | sudo k3s ctr images import -`.
- Floci runs in Docker and publishes port 4566. After K3s added its iptables rules (KUBE-*, CNI and flannel
  chains next to Docker's DOCKER chains), the port still works from the VM, from pods (via the node IP
  10.0.2.15) and from the host. `make -C infra k3s-smoke` re-checks this.
- Ports in use: Traefik (klipper ServiceLB) takes 80 and 443 on the node, the Movies app NodePort is 30080,
  and Floci (Docker) is 4566.

## Port forwards (VirtualBox, `127.0.0.1` on the host only)

| Host | Guest | Purpose |
|---|---|---|
| 8080 | 80 | Traefik ingress |
| 4566 | 4566 | Floci |

## Checks

- `make -C infra k3s-status` shows nodes, pods, storage class, ingress class, `kubectl version` and `systemctl status k3s`.
- `make -C infra k3s-smoke` runs in namespace `k3s-smoke` and deletes it afterwards. It checks:
  - a 2-replica nginx Deployment, Service, and Ingress on `/smoke`;
  - that `curl http://localhost/smoke` gets 200 through Traefik and both pods answer;
  - that a file written to a local-path PVC survives deleting and recreating the pod;
  - Docker and Floci coexistence.
- Traefik answers `no available server` for a few seconds after new endpoints appear. The smoke script waits for that.
- Evidence: `docs/evidence/03-k3s.txt`.

## Uninstall

```sh
/usr/local/bin/k3s-uninstall.sh   # stops K3s, removes the binary, data (/var/lib/rancher/k3s) and its iptables rules
rm -f ~/.kube/config
```

Docker and Floci are not affected. Local-path volumes under `/var/lib/rancher/k3s/storage` are deleted.
