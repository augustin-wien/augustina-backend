# CI and tests

This repository runs unit and integration tests in GitHub Actions using Docker Compose to start dependent services (Postgres, Keycloak, etc.).

Local reproduction

To reproduce the GitHub Actions job locally we include a small helper and an installer for the local runner tool in the repository. See `Makefile` targets `install-act` and `ci-local`.

- `make install-act` downloads the local runner binary into `./tools/`.
- `make ci-local` runs the CI job locally (requires Docker and sufficient disk for images).

Notes

- Integration tests require services to be up (Postgres, Keycloak). If you run `go test ./...` locally without the services, several tests will fail.
- Consider adding a build tag (for example `integration`) to heavy tests to make `go test ./...` run only unit tests by default. CI can then explicitly run `-tags=integration`.

Workflow details

We adjusted the GitHub Actions workflow to avoid service env interpolation issues and to copy the repository `.env` into the Docker compose directory when needed so the Compose services pick up the correct environment.

If you need to debug CI failures locally, run `make ci-local`, let the runner pull the images, and inspect the job logs. The job is identical to the CI job in `.github/workflows/test.yml`.

Further reading

See `docs/ci/act.md` for additional notes about using the local CI runner tool shipped in `./tools/`.
