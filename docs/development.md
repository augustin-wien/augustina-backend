# Development

This document collects the common local development steps and tips.

Prerequisites

- Docker & Docker Compose
- Go toolchain (matches `go.mod` in `app/`)
- (optional) ngrok for webhook tunnelling

Initial repository setup

1. Initialize submodules (frontend, augustin-plugin, WordPress plugin and themes):

```bash
git submodule update --init --recursive
```

2. Copy the environment template:

```bash
cp .env.example .env
```

Set `KEYCLOAK_HOST` in `.env` to `http://keycloak:8080/` or your environment-specific host.

Running the system locally (docker compose)

To start everything:

```bash
docker compose build
docker compose up -d
```

For backend-only development you can start a subset of services:

```bash
docker compose up -d augustin augustin-db augustin-db-test keycloak mailpit
```

The `augustin` container runs the `tern` migrations for the dev and test database on startup.

Useful Make targets:

- `make restart` restarts the backend container and follows its logs
- `make log-webhooks` follows the logs of the webhook receiver
- `make update-db-schema` regenerates the Ent code after schema changes

Additional local services

- `mailpit` catches all e-mails sent by the backend and Keycloak. The defaults in `.env.example` already point SMTP to it. Web UI: http://localhost:8025
- `webhook-receiver` accepts outgoing webhooks (e.g. set `FLOUR_WEBHOOK_URL` or `ODOO_WEBHOOK_URL` to `http://webhook-receiver:8080/`) and writes them to `./webhook-logs`.
- `matomo` is only needed to test the optional analytics integration. `matomo-init` installs it unattended; tracking starts once Matomo URL and site ID are set under Backoffice > Settings.

Accessing the API

Open: http://localhost:3000/api/hello/ and the Swagger UI at http://localhost:3000/swagger/

Tests and linters

Tests use the separate test database (`augustin-db-test`, `DB_PORT_TEST=5433` in `.env.example`). Every test run empties the database it connects to, so make sure `DB_PORT_TEST` is set and does not point at the dev database.

Run the tests inside the `augustin` container:

```bash
make test
# or
docker compose exec augustin sh
go test ./... -p 1 -v -cover
```

To run a single test locally (from `app/`):

```bash
go test ./database/ -v -run TestName
```

Tests that need a real Keycloak or SMTP server carry the `integration` build tag and are not run by default:

```bash
go test ./... -tags integration -p 1 -v
```

CI lints with `golangci-lint` (see `.github/workflows/linter.yml`). Additional checks: `make gosec`, `make staticcheck`, `make govulncheck` or all of them with `make security`.

Test coverage

Create a coverage profile and open an HTML report locally:

```bash
go test ./... -p 1 -coverprofile=cover.out
go tool cover -html cover.out -o app/cover.html
# open app/cover.html in your browser
```

Swagger regeneration

See `docs/overview.md` for the commands to regenerate the OpenAPI docs.

Working with Keycloak

The local Keycloak instance is reachable at http://localhost:8080. The realm is imported from `docker/keycloak/import` on startup.

The test realm contains these users:

| Username       | Password | Role                            |
| -------------- | -------- | ------------------------------- |
| test_nouser    | Test123! | -                               |
| test_user      | Test123! | magazin-1                       |
| test_user_all  | Test123! | magazin-1, magazin-2, magazin-3 |
| test_superuser | Test123! | admin                           |
| test_vendor    | Test123! | vendor                          |

Generate a token with the `frontend` client:

```bash
curl --location --request POST 'http://localhost:8080/realms/augustin/protocol/openid-connect/token' \
	--header 'Content-Type: application/x-www-form-urlencoded' \
	--data-urlencode 'client_id=frontend' \
	--data-urlencode 'grant_type=password' \
	--data-urlencode 'username=test_superuser' \
	--data-urlencode 'password=Test123!' \
	--data-urlencode 'scope=openid'
```

### ngrok (webhook tunnelling)

To receive VivaWallet webhooks locally, tunnel port 3000 with ngrok, for example:

```bash
ngrok http --domain=workable-credible-mole.ngrok-free.app 3000
```
