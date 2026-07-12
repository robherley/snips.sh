# Contributing

Thank you for considering contributing to snips.sh! We welcome contributions from anyone, whether it's a bug report, a feature request, code improvements, or documentation updates.

## Issues and Questions

Please use [Issues](https://github.com/robherley/snips.sh/issues) to report bugs and feature enhancements. For any questions, discussions or general feedback, please use the [Discussions](https://github.com/robherley/snips.sh/discussions).

## Local Development

To get started, you'll need [Go](https://go.dev/doc/install) and [just](https://github.com/casey/just) installed.

AI-powered file type detection is provided by [magika-go](https://github.com/robherley/magika-go), which embeds the model and ONNX runtime at build time. No additional setup is required.

To run locally:

```bash
just run
```

There are some nice defaults for local development. To see all the available configuration options, run:

```
just run -usage
```

Taking a look at the [`database.md`](/docs/database.md) and [`self-hosting.md`](/docs/self-hosting.md) documents may be useful too.

If you are working on the web UI, I recommend installing [air](https://github.com/cosmtrek/air) so the application recompiles when the files change. Otherwise, the assets won't update while the binary is running.

### Recipes

Run `just` to list the available development recipes. The main recipes are:

`just build`: builds the snips.sh binary; set `OUTPUT` to change its destination

`just run`: runs the application locally with the required ONNX runtime configuration

`just lint`: locally installs and runs the Go and web linters

`just migrate`: wrapper for [goose](https://github.com/pressly/goose) to manage database migrations (see [`database.md`](/docs/database.md))

`just mocks`: generates mock interfaces using [mockery](https://github.com/vektra/mockery) for testing

`just record-tape`: runs [vhs](https://github.com/charmbracelet/vhs) on `docs/tapes/` to generate gifs for the README

`just ssh-tmp`: runs SSH with a new temporary public key, useful for testing new user access

`just test`: locally installs [gotestsum](https://github.com/gotestyourself/gotestsum) and runs the tests

`just vendor-onnxruntime`: downloads and installs the [ONNX runtime](https://github.com/microsoft/onnxruntime) for the current platform
