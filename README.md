# Augustin Backend

## Development

Start the application with Docker:

```bash
docker compose build
docker compose up -d
```

Go to http://localhost:3000/api/hello

### Tests

Open the augustin shell in the container:

```bash
docker compose exec augustin bash
```

Run linter within the augustin shell:

```bash
golint ./...
```

Run tests within the augustin shell:

```bash
go test -v -cover
```

### Migrations

Run migrations within the augustin shell (see [tern](https://github.com/jackc/tern)):

```bash
cd migrations
```

Create a new migration:

```bash
tern new <migration_name>
```

Apply all pending migrations:

```bash
tern migrate
```

Revert last migration:

```bash
tern migrate --destination -1
```

### Swagger

Go to http://localhost:3000/api/swagger

To add API calls to Swagger, download [Swaggo](https://github.com/swaggo/swag) via `go install github.com/swaggo/swag/cmd/swag@latest`

To update the swagger documentation for a changed api call run the following command `swag init -g handlers/handlers.go`

## Keycloak

The keycloak server is available at http://localhost:8080 and the admin console at http://localhost:8080/auth/admin

The admin user for the `master` realm is `admin` and the password is `admin`
Additionaly there are the following users for the `augustin` realm:

| Username | Password | Role |
| -------- | -------- | ---- |
| test_nouser     | Test123!     | - |
| test_user     | Test123!     | magazin-1 |
| test_user_all     | Test123!     | magazin-1, magazin-2, magazin-3 |
| test_superuser     | Test123!     | admin |

The default openid configuration is available at http://localhost:8080/auth/realms/augustin/.well-known/openid-configuration

The default openid client is `wordpress` and the client secret is `84uZmW6FlEPgvUd201QUsWRmHzUIamZB`

### Keycloak Wordpress Setup

Install the [`OpenID Connect Generic`](https://wordpress.org/plugins/daggerhart-openid-connect-generic/) plugin and configure it as follows:

| Setting | Value |
| -------- | -------- |
| Login Type | Auto Login - SSO |
| Client ID     | wordpress     |
| Client Secret Key | 84uZmW6FlEPgvUd201QUsWRmHzUIamZB |
| OpenID Scope | email profile openid offline_access roles |
| Login Endpoint URL | http://localhost:8080/realms/augustin/protocol/openid-connect/auth |
| Userinfo Endpoint URL | http://localhost:8080/realms/augustin/protocol/openid-connect/userinfo |
| Token Validation Endpoint URL | http://localhost:8080/realms/augustin/protocol/openid-connect/token |
| End Session Endpoint URL | http://localhost:8080/realms/augustin/protocol/openid-connect/logout |
| Enforce Privacy | Yes |

#### Optional Setup to have a role -> capability mapping
Install the plugin [`Groups`](https://wordpress.org/plugins/groups/) and the plugin [`Augustin`](https://github.com/augustin-wien/augustin-wp-papers) and it should work automatically.

### Generate token with curl

```bash
curl --location --request POST 'http://localhost:8080/realms/augustin/protocol/openid-connect/token' --header 'Content-Type: application/x-www-form-urlencoded' --data-urlencode 'client_id=frontend' --data-urlencode 'grant_type=password' --data-urlencode 'username=user001' --data-urlencode 'password=Test123!' --data-urlencode 'scope=openid'
```
