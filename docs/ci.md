# CI and tests

GitHub Actions runs two workflows:

- `.github/workflows/test.yml` (job `Gotest`): starts Postgres, Keycloak and the other services with Docker Compose and runs `go test ./... -p 1 -v -cover` in `app/`.
- `.github/workflows/linter.yml` (job `Golint`): runs `golangci-lint` on `app/`.

Tests with the `integration` build tag need a real Keycloak or SMTP server and are not run in CI. Run them locally with `go test ./... -tags integration -p 1 -v`.

Local reproduction

The `Gotest` job can be run locally with `act`:

- `make install-act` downloads `act` into `./tools/`.
- `make ci-local` runs the job locally (requires Docker and enough disk space for the images).

See `docs/ci/act.md` for details.
