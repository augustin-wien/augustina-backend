# Running GitHub Actions locally with `act`

This project includes a Makefile target `ci-local` which uses `act` to run the `Gotest` job from `.github/workflows/test.yml` locally.

Two supported ways to run it:

1) Install `act` into the repo tooling directory and run via Makefile (recommended for no-sudo installs):

```bash
# download act into ./tools and make it executable
make install-act

# run the CI job locally (passes .env if present)
make ci-local
```

2) Install `act` system-wide and run the Makefile target directly:

macOS (brew):

```bash
brew install act
make ci-local
```

Linux (example using the project's installer script):

```bash
bash scripts/install_act.sh
make ci-local
```

Notes and caveats:

- `act` runs workflow jobs inside Docker. Ensure Docker is installed and running.
- The runner image used by Makefile is `ghcr.io/catthehacker/ubuntu:full-22.04` which provides common tools; change the image in the Makefile if you prefer a different runner.
- The workflow may require secrets (Keycloak admin credentials, Docker Hub token, etc.). You can pass secrets to `act` with `-s NAME=value` or via a secrets file. See https://github.com/nektos/act for more details.
- Some steps (like `docker compose up`) behave slightly differently in `act`; if you hit issues, run the equivalent steps locally (docker compose up, wait, then run migrations and `go test`).

If you want, I can add a small `.actrc` example or a `scripts/act-secrets` helper to simplify secret passing.
