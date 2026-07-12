set shell := ["bash", "-euo", "pipefail", "-c"]

root := justfile_directory()
bin_dir := root / "bin"
onnx_dir := env_var_or_default("ONNX_DIR", root / "third_party/onnxruntime")
onnx_lib := onnx_dir / "lib"
output := env_var_or_default("OUTPUT", bin_dir / "snips.sh")
extldflags := if os() == "macos" { "-Wl,-rpath," + onnx_lib + " -Wl,-no_warn_duplicate_libraries" } else { "-Wl,-rpath," + onnx_lib }

onnx_version := "1.23.2"
golangci_lint_version := "v2.9.0"
biome_version := "2.3.15"
gotestsum_version := "v1.10.0"
goose_version := "v3.26.3"
mockery_version := "v3.6.3"

export CGO_ENABLED := "1"
export CGO_CFLAGS := "-I" + onnx_dir / "include"
export CGO_LDFLAGS := "-L" + onnx_lib + " -lonnxruntime"
export DYLD_LIBRARY_PATH := if os() == "macos" { onnx_lib + if env_var_or_default("DYLD_LIBRARY_PATH", "") == "" { "" } else { ":" + env_var("DYLD_LIBRARY_PATH") } } else { env_var_or_default("DYLD_LIBRARY_PATH", "") }
export LD_LIBRARY_PATH := if os() == "linux" { onnx_lib + if env_var_or_default("LD_LIBRARY_PATH", "") == "" { "" } else { ":" + env_var("LD_LIBRARY_PATH") } } else { env_var_or_default("LD_LIBRARY_PATH", "") }

default:
    @just --list

_check-onnx:
    @test -d "{{ onnx_dir }}" || { echo "ONNX runtime not found at {{ onnx_dir }}" >&2; echo "Run: just vendor-onnxruntime" >&2; exit 1; }

_tool binary package version:
    mkdir -p "{{ bin_dir }}"
    test -x "{{ bin_dir }}/{{ binary }}" || GOBIN="{{ bin_dir }}" go install "{{ package }}@{{ version }}"

# Run the application locally
run *args: _check-onnx
    go run -ldflags '-extldflags "{{ extldflags }}"' . {{ args }}

# Build bin/snips.sh (override with OUTPUT=/path)
build *args: _check-onnx
    mkdir -p "$(dirname "{{ output }}")"
    go build -ldflags '-extldflags "{{ extldflags }}"' -o "{{ output }}" . {{ args }}

# Run all tests
test: _check-onnx (_tool "gotestsum" "gotest.tools/gotestsum" gotestsum_version)
    "{{ bin_dir }}/gotestsum" --raw-command -- go test -json -ldflags '-extldflags "{{ extldflags }}"' ./...

# Run all linters
lint: lint-go lint-web

# Lint Go code
lint-go: _check-onnx (_tool "golangci-lint" "github.com/golangci/golangci-lint/v2/cmd/golangci-lint" golangci_lint_version)
    "{{ bin_dir }}/golangci-lint" run

# Lint web code
lint-web:
    #!/usr/bin/env bash
    mkdir -p "{{ bin_dir }}"
    if [[ ! -x "{{ bin_dir }}/biome" ]]; then case "$(uname -s)-$(uname -m)" in Darwin-arm64) target=biome-darwin-arm64;; Darwin-x86_64) target=biome-darwin-x64;; Linux-x86_64) target=biome-linux-x64;; Linux-aarch64) target=biome-linux-arm64;; *) echo "Unsupported platform: $(uname -s)-$(uname -m)"; exit 1;; esac; curl -sSfL "https://github.com/biomejs/biome/releases/download/@biomejs/biome@{{ biome_version }}/$target" -o "{{ bin_dir }}/biome"; chmod +x "{{ bin_dir }}/biome"; fi
    "{{ bin_dir }}/biome" check

# Run goose with the supplied arguments
migrate *args: (_tool "goose" "github.com/pressly/goose/v3/cmd/goose" goose_version)
    "{{ bin_dir }}/goose" {{ args }}

# Generate mocks with mockery
mocks *args: (_tool "mockery" "github.com/vektra/mockery/v3" mockery_version)
    "{{ bin_dir }}/mockery" {{ args }}

# Record one or more tapes from docs/tapes
record-tape *tapes:
    #!/usr/bin/env bash
    command -v vhs >/dev/null || { echo "vhs not found: https://github.com/charmbracelet/vhs#installation"; exit 1; }
    [[ -n "{{ tapes }}" ]] || { echo "specify a tape name to record"; exit 1; }
    for tape in {{ tapes }}; do vhs "{{ root }}/docs/tapes/$tape.tape" -o "{{ root }}/docs/tapes/$tape.gif"; done

# Connect with a fresh temporary SSH key
ssh-tmp *args:
    #!/usr/bin/env bash
    [[ -n "{{ args }}" ]] || { echo "usage: just ssh-tmp user@host"; exit 1; }
    tmpdir="$(mktemp -d)"; trap 'rm -rf "$tmpdir"' EXIT
    ssh-keygen -t ecdsa -f "$tmpdir/id_ecdsa" -q -N ""
    ssh -F /dev/null -i "$tmpdir/id_ecdsa" -o IdentitiesOnly=yes {{ args }}

# Download ONNX Runtime for the current or TARGETOS/TARGETARCH platform
vendor-onnxruntime:
    #!/usr/bin/env bash
    os_name="${TARGETOS:-$(uname -s)}"
    arch_name="${TARGETARCH:-$(uname -m)}"

    case "$os_name" in
      darwin|Darwin) os=osx ;;
      linux|Linux) os=linux ;;
      *) echo "Unsupported OS: $os_name" >&2; exit 1 ;;
    esac

    case "$arch_name" in
      amd64|x86_64)
        [[ "$os" == osx ]] && arch=x86_64 || arch=x64
        ;;
      arm64|aarch64)
        [[ "$os" == osx ]] && arch=arm64 || arch=aarch64
        ;;
      *) echo "Unsupported architecture: $arch_name" >&2; exit 1 ;;
    esac

    vendor_dir="${VENDOR_DIR:-{{ root }}/third_party}"
    platform="$os-$arch"
    filename="onnxruntime-$platform-{{ onnx_version }}.tgz"
    archive="$vendor_dir/$filename"
    release_url="https://github.com/microsoft/onnxruntime/releases/download/v{{ onnx_version }}/$filename"
    release_api="https://api.github.com/repos/microsoft/onnxruntime/releases/tags/v{{ onnx_version }}"

    echo "Fetching checksum for $filename..."
    expected_checksum="$(
      curl -fsSL "$release_api" |
        jq -r --arg filename "$filename" \
          '.assets[] | select(.name == $filename) | .digest' |
        sed 's/^sha256://'
    )"

    if [[ -z "$expected_checksum" || "$expected_checksum" == null ]]; then
      echo "Failed to fetch checksum for $filename" >&2
      exit 1
    fi

    mkdir -p "$vendor_dir"
    trap 'rm -f "$archive"' EXIT

    echo "Downloading ONNX Runtime {{ onnx_version }} for $platform..."
    curl -fSL "$release_url" -o "$archive"

    if command -v sha256sum >/dev/null; then
      actual_checksum="$(sha256sum "$archive" | awk '{print $1}')"
    else
      actual_checksum="$(shasum -a 256 "$archive" | awk '{print $1}')"
    fi

    if [[ "$actual_checksum" != "$expected_checksum" ]]; then
      echo "Checksum verification failed" >&2
      echo "Expected: $expected_checksum" >&2
      echo "Actual:   $actual_checksum" >&2
      exit 1
    fi

    extracted_dir="$vendor_dir/onnxruntime-$platform-{{ onnx_version }}"
    install_dir="$vendor_dir/onnxruntime"

    tar -xzf "$archive" -C "$vendor_dir"
    rm -rf "$install_dir"
    mv "$extracted_dir" "$install_dir"

    echo "ONNX Runtime installed to $install_dir"
