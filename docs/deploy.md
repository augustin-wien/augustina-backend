# Deploy & operations

This document contains operational notes: building images, migrations, VivaWallet integration, e-mail and notifications.

Building and pushing images

The production images (`dkia/augustin-backend`, `dkia/augustin-frontend`) are built and pushed with `docker-compose.production.yml`:

```bash
make build   # pulls base images, updates the frontend submodule, bumps the version, builds both images
make push    # pushes both images and writes the local changelog
```

`make build` increments the patch version in `app/config/config.go` and appends the current commit hash (`scripts/update_version.py`).

`make push` runs `make changelog` at the end (`scripts/changelog.sh`). It prepends a section to `CHANGELOG.local.md` listing the backend and frontend commits since the last push, plus any uncommitted changes that went into the images. The commits of the last push are stored in `.last-push`. Both files are local and gitignored. Without `.last-push` the last 30 commits are listed. `make changelog` can also be run on its own.

The Keycloak image is built and pushed separately with `make build-keycloak` and `make push-keycloak`.

Database

To open a SQL shell on the test database (defaults from `.env.example`):

```bash
docker exec -it augustin-db-test psql -U user -p 5433 product_api
# password: password (default in example)
```

Migrations

We use `tern` for SQL migrations. They live in `app/migrations` and are applied automatically when the container starts (`docker/entrypoint.sh` in production, the `augustin` service command in development).

Manual commands, from `app/`:

```bash
tern new --migrations ./migrations <migration_name>
tern migrate --migrations ./migrations --config ./migrations/tern.conf                    # apply pending migrations
tern migrate --migrations ./migrations --config ./migrations/tern.conf --destination -1   # revert last migration
```

`tern` treats migration files as Go templates. A literal `{{.Field}}` in a migration (e.g. in mail template bodies) must be escaped, otherwise it is silently replaced with `<no value>`.

VivaWallet

Set the required environment variables in your `.env` file:

- VIVA_WALLET_SOURCE_CODE
- VIVA_WALLET_SMART_CHECKOUT_CLIENT_ID
- VIVA_WALLET_SMART_CHECKOUT_CLIENT_KEY
- VIVA_WALLET_SMART_CHECKOUT_URL
- VIVA_WALLET_VERIFICATION_KEY
- VIVA_WALLET_API_URL
- VIVA_WALLET_ACCOUNTS_URL
- VIVA_WALLET_TRANSACTION_TYPE_ID_PAYPAL

Webhook endpoints used by VivaWallet:

- Transaction Failed: `/api/webhooks/vivawallet/failure/`
- Transaction Price Calculated: `/api/webhooks/vivawallet/price/`
- Transaction Payment Created: `/api/webhooks/vivawallet/success/`

Demo credentials (do NOT use in production):

```bash
VIVA_WALLET_SOURCE_CODE="6343"
VIVA_WALLET_SMART_CHECKOUT_CLIENT_ID="e76rpevturffktne7n18v0oxyj3m6s532r1q4y4k4xx13.apps.vivapayments.com"
VIVA_WALLET_SMART_CHECKOUT_CLIENT_KEY="qh08FkU0dF8vMwH76jGAuBmWib9WsP"
VIVA_WALLET_VERIFICATION_KEY="94FA5D3BA6DBC79DC56E6BC7E2F8A3F25A566EAE"
VIVA_WALLET_API_URL="https://demo-api.vivapayments.com"
VIVA_WALLET_ACCOUNTS_URL="https://demo-accounts.vivapayments.com"
```

E-mail templates

Mail templates are stored in the database (`mail_templates` table). The defaults are seeded by migrations and can be edited in the backoffice or via the `/api/mail-templates/` endpoints.

SMTP configuration

To send e-mails, set these environment variables in your `.env`. In development they point to `mailpit`:

```bash
SMTP_SERVER=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=user
SMTP_PASSWORD=password
SMTP_SENDER_ADDRESS=user@example.com
SMTP_SENDER_NAME=Augustin
SMTP_SSL=false
SMTP_INSECURE_SKIP_VERIFY=false   # skip TLS certificate verification (dev only)
```

Sentry (error reporting)

To enable Sentry error logging, set a Sentry DSN in your `.env`:

```bash
SENTRY_DSN=            # Backend sentry (example: https://<key>@sentry.io/<id>)
VITE_SENTRY_DSN=       # Frontend Sentry DSN (optional)
```

Notifications

Optional e-mail notifications about backend errors and events:

```bash
NOTIFICATIONS_EMAIL_ENABLED=true
NOTIFICATIONS_EMAIL_SERVER=
NOTIFICATIONS_EMAIL_PORT=587
NOTIFICATIONS_EMAIL_SENDER=user@example.com
NOTIFICATIONS_EMAIL_USER=user
NOTIFICATIONS_EMAIL_PASSWORD=password
NOTIFICATIONS_EMAIL_RECEIVER=user@example.com
NOTIFICATIONS_EMAIL_PREFIX=augustin   # subject prefix for notifications
NOTIFICATIONS_PREFIX=augustin         # subject prefix for error notifications, also the Sentry server name
```

Troubleshooting

If you see JSON parsing errors such as `invalid character '}' looking for beginning of object key string`, check any JSON in env files or templates for stray commas or invalid syntax.
