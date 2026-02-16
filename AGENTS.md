# snips.sh

A pastebin/snippet sharing service with SSH, HTTP, and web interfaces.

## CGO / ONNX Runtime Dependency

This project uses CGO with the ONNX runtime (`third_party/onnxruntime`). All `go` commands that compile code (build, test, run, vet) need specific CGO and linker flags to work. **Never run `go build`, `go test`, `go run`, or `go vet` directly.** Always use the entrypoint scripts in `script/` which source `script/env` to configure the environment.

If the ONNX runtime is not vendored yet, run `script/vendor-onnxruntime` first.

## Scripts

All entrypoint scripts live in `script/`. The Go-related ones source `script/env` to set CGO flags.

| Script | Purpose | Accepts args? |
|---|---|---|
| `script/env` | Sourced (not executed) by other scripts to set `CGO_ENABLED`, `CGO_CFLAGS`, `CGO_LDFLAGS`, etc. | N/A |
| `script/build` | `go build` with correct ldflags. Output defaults to `bin/snips.sh`. | Yes (`$@` passed to `go build`) |
| `script/run` | `go run` with correct ldflags. | Yes (`$@` passed to `go run`) |
| `script/test` | Runs tests via `gotestsum`. Defaults to `./...` if no args given. | Yes (`$@` replaces `./...` package pattern) |
| `script/vet` | `go vet` with correct ldflags. Defaults to `./...` if no args given. | Yes (`$@` replaces `./...` package pattern) |
| `script/lint` | Runs both `script/lint-go` and `script/lint-web`. | No |
| `script/lint-go` | Runs `golangci-lint` (auto-installs to `bin/`). | No |
| `script/lint-web` | Runs `biome check` (auto-installs to `bin/`). | No |
| `script/vendor-onnxruntime` | Downloads and vendors the ONNX runtime into `third_party/onnxruntime`. | No |
| `script/mocks` | Runs `mockery` for generating mocks (auto-installs). | Yes (`$@` passed to `mockery`) |
| `script/migrator` | Runs `goose` for database migrations (auto-installs). | Yes (`$@` passed to `goose`) |

## Common Commands

```sh
script/vendor-onnxruntime  # first-time setup: vendor ONNX runtime
script/build               # build the binary
script/run                 # run the server
script/test                # run all tests
script/test ./internal/... # run tests for a specific package
script/vet                 # vet all packages
script/lint                # lint everything (go + web)
```
