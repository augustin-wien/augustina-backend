# Augustin Backend

## Development

Start the application locally:

```bash
go run app/app.go
```

Start the application with Docker:

```bash
docker compose build
docker compose up -d
```

Go to http://localhost:3000 or http://localhost:3000/api/v1/helloworld

Run tests from within the Docker container:

```bash
docker compose exec container_name sh
go test ./... -v -cover
```


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