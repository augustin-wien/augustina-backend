# Project overview

This repository contains the backend for the Augustin payment system.

What this service provides

- REST API for transactions, verification, vendors, locations and other domain features.
- Integration with Keycloak for authentication and roles.
- Persistence using PostgreSQL and Ent (entgo) models.
- Integrations for VivaWallet payments, e-mail notifications and outgoing webhooks.

Ports (default local)

- Backend: http://localhost:3000
- Keycloak admin: http://localhost:8080
- Frontend: http://localhost:8060
- Mailpit (captured e-mails): http://localhost:8025, SMTP on port 1025
- Webhook receiver: http://localhost:8081
- Matomo: http://localhost:8092

Swagger

After starting the backend you can view the OpenAPI docs at:

http://localhost:3000/swagger/

To regenerate the swagger files locally (if you change handlers):

1. Install the swagger generator: `go install github.com/swaggo/swag/cmd/swag@latest`
2. From the `app` directory run:

```bash
swag init -g handlers/swagger.go --parseDependency --parseInternal --parseDepth 1
```

Health & readiness

The service exposes `/healthz` and `/readyz`.
