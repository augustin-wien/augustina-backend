# Augustin Backend

## Development

Start the application with Docker:

```bash
docker compose build
docker compose up -d
```

Go to http://localhost:3000/api/hello/

### Swagger

Visit [http://localhost:3000/swagger/](http://localhost:3000/swagger/)

To update swagger, install swagger

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Rebuild swagger

```bash
cd app
swag init -g handlers/swagger.go --parseDependency --parseInternal --parseDepth 1
```

Note: If the update does not show in your browser, reset cache.

### Tests

Open the augustin shell in the container:

```bash
docker compose exec augustin bash
```

Run linter within the augustin shell:

```bash
golint ./...
```

Run tests within the augustin shell (sed is used to colorize the output):

```bash
go test ./... -v | sed ''/PASS/s//$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$(printf "\033[31mFAIL\033[0m")/''  | sed ''/ERROR/s//$(printf "\033[31mERROR\033[0m")/''
```

Open SQL shell in the container (assuming the variables from `.env.example` are used):

```bash
docker exec -it augustin-db-test  psql -U user -W product_api
```

And then enter `password` as password.

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

### Export Keycloak configs
After running the application with docker compose, the keycloak configs can be exported with the following command:
```bash
docker compose exec keycloak  /opt/keycloak/bin/kc.sh export --dir /tmp/export
```
the exported configs are available in the `docker/keycloak/export` folder.

### Generate keycloak token with curl

```bash
curl --location --request POST 'http://localhost:8080/realms/augustin/protocol/openid-connect/token' --header 'Content-Type: application/x-www-form-urlencoded' --data-urlencode 'client_id=frontend' --data-urlencode 'grant_type=password' --data-urlencode 'username=user001' --data-urlencode 'password=Test123!' --data-urlencode 'scope=openid'
```

## Troubleshooting

```"invalid character '}' looking for beginning of object key string```
-> You might have a false commY at the end of your json

### VivaWallet Checkout Process

This PR covers the MVP (minimal viable product) of the VIvaWallet checkout process. Each process step is described below:

1. Create payment order via new endpoint
  - URL: `http://localhost:3000/api/transaction/`
  - POST Request
  - Sample cURL call
    ```bash
    curl --header "Content-Type: application/json" \
    --request POST \
    --data '{"amount":2500}' \
    http://localhost:3000/api/transaction/
    ```
  - Here the amount is in cents, so this call requests to charge 2500 cents which is 25€
  - If successful, response is: `{"SmartCheckoutURL":"https://demo.vivapayments.com/web/checkout?ref=8958019584072636"}`
  - VivaWallet checkout URL (Sample link in demo version: https://demo.vivapayments.com/web/checkout?ref=9699361263129530) like

5. After being redirected to the VivaWallet checkout URL, you need [VivaWallet Test Cards
](https://developer.vivawallet.com/integration-reference/test-cards-and-environments/) to have a successful process.
  - NOTE: First option with Visa card did not work for me but third card option for Mastercard did.

6. After a successful transaction the user is being redirected to a success page, which is right now `https://local.com/success` and will be changed towards production.
  - NOTE: The whole sample URL, looks something like this `https://local.com/success?t=d87ea0e6-91da-4312-abdf-67ebb84ee981&s=5857961245421135&lang=en-GB&eventId=0&eci=1`
  - Here the frontend (or developer for testing purposes) has to extract the transactionID, which is **t** or in this sample URL above `d87ea0e6-91da-4312-abdf-67ebb84ee981`

7. It is the frontends task to extract the transactionID from the URL to verify the transaction via new endpoint
  - URL: http://localhost:3000/api/verification/
  - POST Request
  - Sample cURL call
    ```bash
    curl --header "Content-Type: application/json" \
    --request POST \
    --data '{"transactionid":"0a384178-d329-4d54-9474-75c4adff51c0"}' \
    http://localhost:3000/api/verification/
    ```
  - If successful, response is: `{"Verification":true}`
