# Contributing

Thank you for considering contributing to snips.sh! We welcome contributions from anyone, whether it's a bug report, a feature request, code improvements, or documentation updates.

## Issues and Questions

Please use [Issues](https://github.com/robherley/snips.sh/issues) to report bugs and feature enhancements. For any questions, discussions or general feedback, please use the [Discussions](https://github.com/robherley/snips.sh/discussions).

## Local Development

To get started, you'll need to have [Go installed](https://go.dev/doc/install).

In addition, the [libtensorflow](https://www.tensorflow.org/install/lang_c) shared objects for the C API need to be present on your system in order to use [Guesslang](https://github.com/robherley/guesslang-go). Otherwise, you'll see a bunch of "cannot open shared object file" errors. There's a utility script (`script/install-libtensorflow`) that will install it via `brew` for macOS or download from source for Linux.

Once those dependencies are installed, you just need to:

```
go run main.go
```

To run it locally. There are some nice defaults for local development. To see all the available configuration options, run:

```
go run main.go -usage
```

Taking a look at the [`database.md`](/docs/database.md) and [`self-hosting.md`](/docs/self-hosting.md) documents may be useful too.

If you are working on the web UI, I recommend installing [air](https://github.com/cosmtrek/air) so the application recompiles when the files change. Otherwise, the assets won't update while the binary is running.

### Scripts

This repository follows [scripts-to-rule-them-all](https://github.com/github/scripts-to-rule-them-all), here's a brief description of each:

`script/atlas`: locally installs [atlas](https://atlasgo.io/) CLI

`script/install-libtensorflow`: installs [libtensorflow](https://www.tensorflow.org/install/lang_c) shared objects for the C API (required for Guesslang)

`script/lint`: locally installs [golangci-lint](https://github.com/golangci/golangci-lint) and runs the linter

`script/record-tape`: runs [vhs](https://github.com/charmbracelet/vhs) on `docs/tapes/` to generate gifs for readme

`script/schema-diff`: using [atlas](https://atlasgo.io/), prints the difference of local schema with latest `main` schema

`script/ssh-tmp`: helper to run ssh with a new (temporary) public key, useful for testing new user access

`script/test`: runs go tests with [gotestsum](https://github.com/gotestyourself/gotestsum)
