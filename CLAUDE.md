# Augustin Backend — AI Context

## Repository layout

The Go module root is `app/`, not the repository root. Always run Go commands from there:

```sh
cd app
go build ./...
go test ./...
```

The `go.mod` file is at `app/go.mod`. Running `go build` from the repository root will fail.

## Running tests

All `go test` commands must be run from `app/`:

```sh
cd app

# Run all tests (requires a running Postgres — see docker-compose.yml)
go test ./... -p 1 -v

# Run a single package
go test ./database/ -v

# Run a specific test by name
go test ./database/ -run TestSyncAbonementLicensesToKeycloak -v

# Run integration-tagged tests (need DB + Keycloak)
go test ./... -tags integration -p 1 -v
```

Tests that don't carry `//go:build integration` run in CI automatically. Integration-tagged tests require a real Keycloak instance and are not run in CI by default.

The `TestMain` in `database/queries_test.go` calls `Db.InitEmptyTestDb()` for every test run in that package, so a local Postgres must be reachable. Start one with:

```sh
docker compose up -d postgres
```

## LicenseGroup

`LicenseGroup` is a free-text string identifier on `Item` (stored in the `licensegroup` column). It is **not** a foreign key — it is the name of a Keycloak group. When a customer is granted access to a digital item (e.g. an online issue), the backend adds the item's `LicenseGroup` value to the customer's `licensegroups` list and syncs that to Keycloak. Keycloak then uses group membership to gate access to the protected digital content.

Example values: `"analog_edition"`, `"digital_edition"`.

### How it flows

1. An `issue` or `online_issue` item is created with a `LicenseGroup` string (e.g. `"digital_edition"`).
2. A `license_item` linked to that issue also carries the same `LicenseGroup` string.
3. When a customer purchases an abonement, `ProcessAbonementLicenseGroupsForDate` reads the item's `LicenseGroup` and calls `AddLicenseGroupToCustomer`.
4. `Customer.LicenseGroups` is a `[]string` in Go, serialized as a comma-separated string in the `licensegroups` TEXT column. The API exposes it as a JSON array.
5. `SyncAbonementLicensesToKeycloak` propagates the groups to Keycloak by calling `AssignDigitalLicenseGroup` for each group.

### Do not change LicenseGroup to a foreign key or dropdown

The field must remain a free-text string because it is the Keycloak group name. The value must match exactly what is configured in Keycloak. Administrators type it manually; it is not derived from any other database record.

## Item types

| Type | Description |
|---|---|
| `normal_item` | Regular shop item |
| `issue` | A physical newspaper issue |
| `online_issue` | A digital issue, access gated via Keycloak LicenseGroup |
| `license_item` | Grants access to a LicenseGroup; linked to an issue via the `LicenseItem` edge |
| `abonement` | Subscription item |
| `donation` | Donation item |
| `transaction_costs` | Transaction cost item |
