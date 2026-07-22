# Development guide

## Requirements

- Go 1.25 or newer (CI resolves the current compatible toolchain)
- GNU Make or compatible `make`
- Docker with Compose for container testing

## Commands

```sh
make dev          # run the development server
make build        # build bin/turnino
make check        # formatting, vet and race-enabled tests
make docker-build # build the local container
docker compose up --build
```

Configuration is documented in `.env.example`. Go does not load `.env` itself; export the variables in your shell or let an editor/task runner load them.

## Module path

The Go module is `github.com/fvlgnn/turnino`.

## Pull-request checks

CI verifies formatting, `go vet`, race-enabled tests, compilation, and a complete container build. Dependabot checks Go modules, GitHub Actions, and Docker base images monthly.

## Release process

1. Ensure `main` is green and update release notes/documentation.
2. Create and push a semantic version tag such as `v0.1.0`.
3. GitHub Actions builds archives and creates a GitHub release.
4. The same workflow publishes `ghcr.io/fvlgnn/turnino` for AMD64 and ARM64.
5. Make the GHCR package public after its first publication if the repository settings do not do so automatically.

Do not create a public tag until the public tree has been checked for private data and credentials.
