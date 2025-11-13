# Development

This document collects the common local development steps and tips.

Prerequisites

- Docker & Docker Compose
- Go toolchain (matches `go.mod` in `app/`)
- (optional) ngrok for webhook tunnelling

Initial repository setup

1. Initialize submodules (wordpress, wpcli, parser):

```bash
git submodule update --init --recursive
```

2. Copy environment templates:

```bash
cp .env.example .env
cp docker/.env.parser.example docker/.env.parser
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
docker compose up -d augustin augustin-db augustin-db-test keycloak
```

Accessing the API

Open: http://localhost:3000/api/hello/ and the Swagger UI at http://localhost:3000/swagger/

Tests and linters

Run tests inside the `augustin` container (recommended to match the environment):

```bash
docker compose exec augustin sh
golint ./...
go test ./... -p 1 -v -cover
```

To run a single test locally (outside container):

```bash
go test ./... -p 1 -v -run TestName
```

Test coverage

Create a coverage profile and open an HTML report locally:

```bash
go test ./... -p 1 -coverprofile=cover.out
go tool cover -html cover.out -o app/cover.html
# open app/cover.html in your browser
```

Swagger regeneration

Integration tests

Some tests require external services (Postgres, Keycloak). These are integration tests and will fail if the services are not available locally. We recommend running them in CI, or locally via the provided `ci-local` Make target which uses a local runner to reproduce the CI environment.

Swagger regeneration

See `docs/overview.md` for the commands to regenerate the OpenAPI docs.

VS Code

We recommend the official Go extension for VS Code. To lint on save, add the following to your VS Code `settings.json`:

```json
"go.lintOnSave": "package",
"go.lintTool": "golint"
```

Working with Keycloak (developer notes)

The local Keycloak instance is reachable at http://localhost:8080. When developing, the repository includes a set of example users and roles used by tests and local runs. See `docker/keycloak/export` after running the containers for exported configs.

Keycloak example users and client (developer convenience)

The development environment ships with a test realm containing example users. For convenience the common local accounts are:

| Username       | Password | Role                            |
| -------------- | -------- | ------------------------------- |
| test_nouser    | Test123! | -                               |
| test_user      | Test123! | magazin-1                       |
| test_user_all  | Test123! | magazin-1, magazin-2, magazin-3 |
| test_superuser | Test123! | admin                           |
| test_vendor    | Test123! | vendor                          |

The default OpenID client used in the local setup is `wordpress`. Example client secret (development only):

```
Client ID: wordpress
Client Secret: 84uZmW6FlEPgvUd201QUsWRmHzUIamZB
```

Generate a Keycloak token (example):

```bash
curl --location --request POST 'http://localhost:8080/realms/augustin/protocol/openid-connect/token' \
	--header 'Content-Type: application/x-www-form-urlencoded' \
	--data-urlencode 'client_id=frontend' \
	--data-urlencode 'grant_type=password' \
	--data-urlencode 'username=user001' \
	--data-urlencode 'password=Test123!' \
	--data-urlencode 'scope=openid'
```
### PDF-Parser

The pdf parser needs a WordPress app password in order to autopublish the articles to Wordpress. This password is stored in the `./docker/.env.parser` file and is used by the parser to authenticate itself against WordPress. Either create a new app password by copying `./docker/.env.parser.example` to `./docker/.env.parser` or run the wpcli container to set up a new one. Note: the wpcli container resets the WordPress installation on every start. This will delete all data in the database and install a fresh WordPress installation.

After setting an app password, redeploy the PDF-Parser container:

```bash
docker compose up -d parser
```

Troubleshooting: If the PDF-Parser does not work, restart the `wpcli` container or re-run the parser deployment:

```bash
docker compose restart wpcli
docker compose up -d parser
```

### ngrok (webhook tunnelling)

For tunnelling endpoints from the internet to your localhost port, we use ngrok. Example (development/demo):

```bash
ngrok http --domain=workable-credible-mole.ngrok-free.app 3000
```
