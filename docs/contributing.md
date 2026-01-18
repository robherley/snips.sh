# Contributing

Thank you for considering contributing to snips.sh! We welcome contributions from anyone, whether it's a bug report, a feature request, code improvements, or documentation updates.

## Issues and Questions

Please use [Issues](https://github.com/robherley/snips.sh/issues) to report bugs and feature enhancements. For any questions, discussions or general feedback, please use the [Discussions](https://github.com/robherley/snips.sh/discussions).

## Local Development

To get started, you'll need to have [Go installed](https://go.dev/doc/install).

AI-powered file type detection is provided by [magika-go](https://github.com/robherley/magika-go), which embeds the model and ONNX runtime at build time. No additional setup is required.

To run locally:

```bash
script/run
```

There are some nice defaults for local development. To see all the available configuration options, run:

```
script/run -usage
```

Taking a look at the [`database.md`](/docs/database.md) and [`self-hosting.md`](/docs/self-hosting.md) documents may be useful too.

If you are working on the web UI, I recommend installing [air](https://github.com/cosmtrek/air) so the application recompiles when the files change. Otherwise, the assets won't update while the binary is running.

### Scripts

This repository follows [scripts-to-rule-them-all](https://github.com/github/scripts-to-rule-them-all), here's a brief description of each:

`script/build`: builds the snips.sh binary, supports cross-compilation via `CC` and `TARGETARCH` environment variables

`script/env`: sets CGO/linker flags for ONNX runtime, source this file before building (e.g., `source script/env`)

`script/lint`: locally installs [golangci-lint](https://github.com/golangci/golangci-lint) and runs the linter

`script/migrator`: wrapper for [goose](https://github.com/pressly/goose) to manage database migrations (see [`database.md`](/docs/database.md))

`script/mocks`: generates mock interfaces using [mockery](https://github.com/vektra/mockery) for testing

`script/record-tape`: runs [vhs](https://github.com/charmbracelet/vhs) on `docs/tapes/` to generate gifs for readme

`script/run`: runs the application locally with proper environment runtime configuration

`script/ssh-tmp`: helper to run ssh with a new (temporary) public key, useful for testing new user access

`script/test`: runs go tests with [gotestsum](https://github.com/gotestyourself/gotestsum)

`script/vendor-onnxruntime`: downloads and installs the [ONNX runtime](https://github.com/microsoft/onnxruntime) for the current platform
