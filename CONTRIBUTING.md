# Contributing

Thanks for improving Agent Fleet.

## Develop

```bash
git clone https://github.com/feralxr/agent-fleet.git
cd agent-fleet
go test ./...
go build -o fleet ./cmd/fleet
./fleet
```

Useful targets:

```bash
make test
make build
make release   # cross-compile into dist/
```

## Guidelines

- Keep the TUI **responsive**: never shell out to `git` (or other slow I/O) from `View()`.
- Prefer small, focused commits.
- Run `gofmt` and `go test ./...` before opening a PR.
- Do not commit binaries, tokens, or local `fleet.toml` secrets.

## Pull requests

1. Fork and branch from `main`
2. Describe the user-visible change
3. Link any related issue

## Releases

Maintainers tag `v*` on `main`. CI builds multi-arch binaries and publishes a GitHub Release.
