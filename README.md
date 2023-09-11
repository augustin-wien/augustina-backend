# Augustin Backend

## Development

Check out the git submodules to load the wordpress, wpcli and parser git checkout mainrepositories:

```bash
git submodule update --init --recursive
```

Start the application with Docker:

```bash
docker compose build
docker compose up -d
```

Go to http://localhost:3000/api/hello/

Note: To make the PDF-Parser run correctly check out description below.

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
