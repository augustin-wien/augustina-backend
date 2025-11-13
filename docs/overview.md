# Project overview

This repository contains the backend for the Augustin payment system.

What this service provides

- REST API for transactions, verification, vendors, locations and other domain features.
- Integration with Keycloak for authentication and roles.
- Persistence using PostgreSQL and Ent (entgo) models.
- Background integrations for PDF parsing, VivaWallet payments and e-mail notifications.

Ports (default local)

- Backend: http://localhost:3000
- Keycloak admin: http://localhost:8080
- Wordpress: http://localhost:8090
- PDF Parser: http://localhost:8070
- Frontend / main webshop: http://localhost:8060

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

The service exposes health and readiness endpoints used by orchestration and CI. See the router and handlers for exact paths (usually `/healthz` and `/readyz`).
