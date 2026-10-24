# Lab runbook (from scratch)

Windows host: VirtualBox + Vagrant. Everything else runs inside the VM.

## 1. VM (Windows)

```powershell
cd infra\vm
vagrant up        # Ubuntu 24.04 + Docker, Go, Terraform, aws cli, K3s
vagrant ssh
```

## 2. Code (VM)

```sh
git clone https://github.com/Berezhnoi/movies.git ~/movies
cd ~/movies
```

## 3. Deploy everything (VM)

```sh
make -C infra secrets   # random lab secrets -> infra/secrets.env (gitignored)
make -C infra all       # Floci -> Lambda build+test -> terraform apply -> image -> K3s deploy
```

## 4. Check (VM)

```sh
make -C infra k3s-status   # node Ready, pods Running
make -C infra e2e          # admin creates a user -> email found in Floci SES
make -C infra mailbox      # show captured emails
```

## 5. Use the app (Windows)

```powershell
vagrant ssh -- -L 30080:localhost:30080   # keep open
```

Open `http://localhost:30080/login` and log in as `admin@movies.local`. The password is in the VM:
`grep ^ADMIN_PASSWORD= ~/movies/infra/secrets.env | cut -d= -f2-`

## Teardown

```sh
make -C infra tf-destroy && kubectl delete ns movies && make -C infra floci-down   # VM
vagrant destroy -f                                                                 # Windows
```

More detail: `docs/floci-notes.md`, `docs/k3s-notes.md`.
