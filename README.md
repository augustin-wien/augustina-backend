# Augustin Backend

## Development

Check out the git submodules to load the wordpress, wpcli and parser git checkout mainrepositories:

```bash
git submodule update --init --recursive
```

Copy `.env.example` to `.env` via
```bash
cp .env.example /.env
```
Then set `KEYCLOAK_HOST=http://keycloak:8080/` in your `.env`

Copy `docker/.env.parser.example` to `docker/.env.parser` via
```bash
cp docker/.env.parser.example docker/.env.parser
```

Start the application with Docker:

```bash
docker compose build
docker compose up -d
```

Go to http://localhost:3000/api/hello/

**Notes**

1. For developing in frontend or backend only, run this command
  ```bash
  docker compose up -d augustin augustin-db augustin-db-test keycloak
  ```
2. To make the PDF-Parser run correctly check out description below.
3. Temporary fix to remove wpcli file: `sudo rm docker/wpcli/.env.parser`


### Ports
Backend: `localhost:3000`

Wordpress: `localhost:8090`

Keycloak login mask: `localhost:8080`

PDF Parser: `localhost:8070`

Frontend / main webshop: `localhost:8060`

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

Run tests within the augustin shell (sed is used to colorize the output, -p 1 is used to prevent parallel computing which causes problems with resetting the database for each test):

```bash
go test ./... -p 1 -v | sed ''/PASS/s//$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$(printf "\033[31mFAIL\033[0m")/''  | sed ''/ERROR/s//$(printf "\033[31mERROR\033[0m")/''
```

To run a specific Test Case:

```bash
go test ./... -p 1 -v -cover -run NameOfTestCase
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

### Setup for developing with VivaWallet Webhooks
For tunneling endpoints from the internet to your locaĺhost port, we use [ngrok](https://ngrok.com/).
If you keep using our demo account and stick with our code basis this command should do it in your terminal:
```bash
ngrok http --domain=workable-credible-mole.ngrok-free.app 3000
```

### VS Code Settings
VS Code is our code editor of choice.
Therefore, to develop in Go, we use the main [VS Code Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go). This enables a lot of features like formatting on save.

Further, we also lint on save by adding these two lines in our `settings.json` file for VS Code:
```json
  "go.lintOnSave": "package",
  "go.lintTool": "golint"
```

### Environment variables
When variable `CREATE_DEMO_DATA=true`, demo data will be created during container creation. This data can be used for development purposes.

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

## Wordpress
The wpcli container resets the wordpress installation on every start. This will delete all data in the database and install a fresh wordpress installation.

### PDF-Parser

The pdf parser needs a wp app password in order to autopublish the articles to the wordpress. This password is stored in the `./docker/.env.parser` file and is used by the parser to authenticate itself against the wordpress. Either you can create a new app password by copying `./docker/.env.parser.example` to `./docker/.env.parser` or run the wpcli container to set up a new one. Note: the wpcli container resets the wordpress installation on every start. This will delete all data in the database and install a fresh wordpress installation.

After setting an app password, the PDF-Parser container has to be deployed again with `docker compose up -d parser`.

#### Explanation
This is due to the reason that the docker container 'wpcli' sets new environment variables, which have to be set again for the PDF-Parser.

#### Trouble shooting
In case your PDF-Parser does not work, make sure everything ran fine in yout wpcli container or might restart it via `docker compose restart wpcli`and then run `docker compose up -d parser`

## Troubleshooting

```"invalid character '}' looking for beginning of object key string```
-> You might have a false commY at the end of your json

## VivaWallet

### Credentials
All the following credentials are needed in your `.env` file.
VIVA_WALLET_SOURCE_CODE="6343"
VIVA_WALLET_SMART_CHECKOUT_CLIENT_ID="e76rpevturffktne7n18v0oxyj3m6s532r1q4y4k4xx13.apps.vivapayments.com"
VIVA_WALLET_SMART_CHECKOUT_CLIENT_KEY="qh08FkU0dF8vMwH76jGAuBmWib9WsP"
VIVA_WALLET_VERIFICATION_KEY="94FA5D3BA6DBC79DC56E6BC7E2F8A3F25A566EAE"
VIVA_WALLET_API_URL="https://demo-api.vivapayments.com"
VIVA_WALLET_ACCOUNTS_URL="https://demo-accounts.vivapayments.com"

#### Source Code
To get your Source code, follow the instructions here: https://developer.vivawallet.com/getting-started/create-a-payment-source/payment-source-for-online-payments/
In case this did not help, try this link: https://help.vivawallet.com/en/articles/5119253-where-can-i-find-the-source-code

#### Smart Checkout Credentials
To get your Smart Checkout Client ID and Client Key, follow the instructions here: https://developer.vivawallet.com/getting-started/find-your-account-credentials/client-smart-checkout-credentials/

#### Verification key
To create a new verification key, follow the instructions here: https://developer.vivawallet.com/webhooks-for-payments/#generate-a-webhook-verification-key

#### TransactionTypeID
To have an overview which transaction type id means what, follow this link: https://developer.vivawallet.com/integration-reference/response-codes/#transactiontypeid-parameter

### Checkout Process

#### 1. Create payment order
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
- If successful, response in demo instance is the checkout URL: `{"SmartCheckoutURL":"https://demo.vivapayments.com/web/checkout?ref=8958019584072636"}`
- Extract link and paste it to your browser or if possible click on it to move forward to next step

#### 2. Test cards
- After being redirected to the VivaWallet checkout URL, you need to use a [VivaWallet Test Card
](https://developer.vivawallet.com/integration-reference/test-cards-and-environments/) to have a successful process.
- NOTE: First option with Visa card did not work for me but third card option for Mastercard did.

#### 3. Redirection page
- After a successful transaction the user is being redirected to a success page, which is right now `https://local.com/success` and will be changed towards production.
- NOTE: The whole sample URL, looks something like this `https://local.com/success?t=d87ea0e6-91da-4312-abdf-67ebb84ee981&s=5857961245421135&lang=en-GB&eventId=0&eci=1`
- Here the frontend (or developer for testing purposes) has to extract the transactionID, which is **t** or in this sample URL above `d87ea0e6-91da-4312-abdf-67ebb84ee981`

#### 4. Create verification call
- It is the frontends task to extract the transactionID from the URL to verify the transaction via new endpoint
- URL: http://localhost:3000/api/verification/
- POST Request
- Sample cURL call
  ```bash
  curl --header "Content-Type: application/json" \
  --request POST \
  --data '{"transactionid":"0a384178-d329-4d54-9474-75c4adff51c0"}' \
  http://localhost:3000/api/verification/
  ```
- If successful, response is: `{"Verification":true}` -> checkout process is successfully finished
- If unsuccessfil, response is: `{"Verification":false}` -> this step is still unclear, in which circumstances a user can be redirected to the success page but her transaction cannot be verified
