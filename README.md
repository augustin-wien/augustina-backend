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

Open the augustin shell in the container:

```bash
docker compose exec augustin bash
```

Run tests within the augustin shell:

```bash
go test ./... -v -cover
```

Run migrations within the augustin shell (see [tern](https://github.com/jackc/tern)):

```bash
cd migrations
```

Create a new migration:

```bash
tern create <migration_name>
```

Apply all pending migrations:

```bash
tern migrate
```

Revert last migration:

```bash
tern migrate --destination -1
```
