# Deploy & operations

This document contains operational notes: migrations, VivaWallet integration, and webhooks.

Database

To open a SQL shell (defaults from `.env.example`):

```bash
docker exec -it augustin-db-test psql -U user -W product_api
# password: password (default in example)
```

Migrations

We use `tern` for managing SQL migrations. Common commands from inside the repository:

```bash
cd migrations
tern new <migration_name>
tern migrate     # apply pending migrations
tern migrate --destination -1  # revert last migration
```

VivaWallet

The repository includes integrations for VivaWallet. Set the required environment variables in your `.env` file (example keys are provided in the top-level README previously):

- VIVA_WALLET_SOURCE_CODE
- VIVA_WALLET_SMART_CHECKOUT_CLIENT_ID
- VIVA_WALLET_SMART_CHECKOUT_CLIENT_KEY
- VIVA_WALLET_VERIFICATION_KEY
- VIVA_WALLET_API_URL
- VIVA_WALLET_ACCOUNTS_URL

Webhook endpoints used by VivaWallet (examples):

- Transaction Failed: `/api/webhooks/vivawallet/failure/`
- Transaction Price Calculated: `/api/webhooks/vivawallet/price/`
- Transaction Payment Created: `/api/webhooks/vivawallet/success/`

Troubleshooting

If you see JSON parsing errors such as `invalid character '}' looking for beginning of object key string`, check any JSON in env files or templates for stray commas or invalid syntax.

E-mail templates

Templates live in `app/templates`, for example `PDFLicenceItemTemplate.html` and `digitalLicenceItemTemplate.html`.

SMTP configuration

To send e-mails, set these environment variables in your `.env` (example values shown):

```bash
SMTP_SERVER=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=user
SMTP_PASSWORD=password
SMTP_SENDER_ADDRESS=user@example.com
SMTP_SSL=false
```

Sentry (error reporting)

To enable Sentry error logging, set a Sentry DSN in your `.env`:

```bash
SENTRY_DSN=            # Backend sentry (example: https://<key>@sentry.io/<id>)
VITE_SENTRY_DSN=       # Frontend Sentry DSN (optional)
```

Notifications (email / Matrix)

Optional notification environment variables (example):

```bash
NOTIFICATIONS_EMAIL_ENABLED=true
NOTIFICATIONS_EMAIL_SERVER=
NOTIFICATIONS_EMAIL_PORT=587
NOTIFICATIONS_EMAIL_SENDER=user@example.com
NOTIFICATIONS_EMAIL_USER=user
NOTIFICATIONS_EMAIL_PASSWORD=password
NOTIFICATIONS_EMAIL_RECEIVER=user@example.com
NOTIFICATIONS_PREFIX=augustin

NOTIFICATIONS_MATRIX_ENABLED=true
NOTIFICATIONS_MATRIX_HOME_SERVER=matrix.org
NOTIFICATIONS_MATRIX_ACCESS_TOKEN=
NOTIFICATIONS_MATRIX_ROOM_ID=
NOTIFICATIONS_MATRIX_USER_ID=
```

VivaWallet credentials (development/demo values)

For convenience the demo config used in previous README versions (do NOT use in production):

```bash
VIVA_WALLET_SOURCE_CODE="6343"
VIVA_WALLET_SMART_CHECKOUT_CLIENT_ID="e76rpevturffktne7n18v0oxyj3m6s532r1q4y4k4xx13.apps.vivapayments.com"
VIVA_WALLET_SMART_CHECKOUT_CLIENT_KEY="qh08FkU0dF8vMwH76jGAuBmWib9WsP"
VIVA_WALLET_VERIFICATION_KEY="94FA5D3BA6DBC79DC56E6BC7E2F8A3F25A566EAE"
VIVA_WALLET_API_URL="https://demo-api.vivapayments.com"
VIVA_WALLET_ACCOUNTS_URL="https://demo-accounts.vivapayments.com"
```

Troubleshooting note (JSON parsing)

If you get errors like `"invalid character '}' looking for beginning of object key string"`, check JSON in templates and env-derived JSON for stray commas or trailing characters.
