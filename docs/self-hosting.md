# Self Hosting

The architecture of snips.sh was designed with self-hosting in mind, it's very simple to get deployed. The recommended approach is to use the published container images on [GitHub Container Registry](https://github.com/robherley/snips.sh/pkgs/container/snips.sh).

- [Self Hosting](#self-hosting)
  - [Quick Start](#quick-start)
  - [Configuration](#configuration)
    - [Addresses/Ports](#addressesports)
    - [Database](#database)
    - [Host Keys](#host-keys)
    - [Limiting SSH Access](#limiting-ssh-access)
    - [Statsd Metrics](#statsd-metrics)
    - [Build without File Type Detection](#build-without-file-type-detection)
  - [Examples](#examples)
    - [Docker Compose](#docker-compose)


## Quick Start

Start the service with HTTP on `8080` and SSH on `2222`, and mount a `/data` directory.

```
docker run -p 2222:2222 -p 8080:8080 -v $PWD/data:/data ghcr.io/robherley/snips.sh
```

You should be able to reach the Web UI at `http://localhost:8080`

And the TUI/SSH server will be available at: `ssh localhost -p 2222`

> **Warning**
> This default configuration should not be used for a public instance, please be sure to look over the config and replace any secret configuration values (like the HMAC signing key).

## Configuration

All of snips' configuration is driven by environment variables. To view all the available options (and their default setting), you can run the following command:

```
docker run ghcr.io/robherley/snips.sh -usage
```

```
KEY                           TYPE              DEFAULT                DESCRIPTION
SNIPS_DEBUG                   True or False     False                  enable debug logging and pprof
SNIPS_ENABLEGUESSER           True or False     True                   enable AI model to detect file types
SNIPS_HMACKEY                 String            hmac-and-cheese        symmetric key used to sign URLs
SNIPS_FILECOMPRESSION         True or False     True                   enable compression of file contents
SNIPS_LIMITS_FILESIZE         Unsigned Integer  1048576                maximum file size in bytes
SNIPS_LIMITS_FILESPERUSER     Unsigned Integer  100                    maximum number of files per user
SNIPS_LIMITS_SESSIONDURATION  Duration          15m                    maximum ssh session duration
SNIPS_DB_FILEPATH             String            data/snips.db          path to database file
SNIPS_HTTP_INTERNAL           URL               http://localhost:8080  internal address to listen for http requests
SNIPS_HTTP_EXTERNAL           URL               http://localhost:8080  external http address displayed in commands
SNIPS_HTML_EXTENDHEADFILE     String                                   path to html file for extra content in <head>
SNIPS_SSH_INTERNAL            URL               ssh://localhost:2222   internal address to listen for ssh requests
SNIPS_SSH_EXTERNAL            URL               ssh://localhost:2222   external ssh address displayed in commands
SNIPS_SSH_HOSTKEYPATH         String            data/keys/snips        path to host keys (without extension)
SNIPS_SSH_AUTHORIZEDKEYSPATH  String                                   path to authorized keys, if specified will restrict SSH access
SNIPS_METRICS_STATSD          URL                                      statsd server address (e.g. udp://localhost:8125)
SNIPS_METRICS_USEDOGSTATSD    True or False     False                  use dogstatsd instead of statsd
```

### Addresses/Ports

For the HTTP and SSH services, the container image exposes ports `8080` and `2222` respectively, listening on all interfaces so they can be bound properly.

For convenience, the relevant environment variables are set in the Dockerfile already:

```dockerfile
ENV SNIPS_HTTP_INTERNAL=http://0.0.0.0:8080
ENV SNIPS_SSH_INTERNAL=ssh://0.0.0.0:2222

EXPOSE 8080 2222
```

However, for externally facing traffic (like the URLs that appear in commands), you must also specify the `SNIPS_*_EXTERNAL` environment variables (or else they point to localhost).

e.g. if I wanted to host snips on a subdomain on example.com, I would set the following variables like so:

```bash
SNIPS_HTTP_EXTERNAL=https://snips.example.com
SNIPS_SSH_EXTERNAL=ssh://snips.example.com:22
```

### Database

The file specified at `SNIPS_DB_FILEPATH` is the SQLite database that holds all user data. For more information managing the database, see [`database.md`](/docs/database.md).

Setting `SNIPS_FILECOMPRESSION` to `false` will disable compression when storing file content to disk. If this option was disabled at any point (or files were created before this option existed), it will not retroactively compress existing files.

### Host Keys

The directory holding the key files should be persistent and not change. If the host keys are not found, snips will automatically generate them when started.

For instance, if the `SNIPS_SSH_HOSTKEYPATH` is `/data/keys/snips`, the `keys` directory will look like so:

```
keys
├── snips_ed25519
└── snips_ed25519.pub
```

The generated keys will _always_ be [`ec25519`](https://en.wikipedia.org/wiki/EdDSA#Ed25519). It's recommended to let snips generate the keys instead of providing your own.

Also, it's very important that the host keys _don't_ change, or else any users are going to see a scary message like so:

```
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!
Someone could be eavesdropping on you right now (man-in-the-middle attack)!
It is also possible that a host key has just been changed.
```

Be sure to securely back up any host keys in the event they might be lost.

### Limiting SSH Access

By default, any user with a public key can connect to a snips.sh instance via SSH.

If you want to limit access to who can SSH (and upload) snippets, you can use the `SNIPS_SSH_AUTHORIZEDKEYSPATH` environment variable. If specified, this will limit the SSH server to the public keys defined there.

The format is exactly the same as `authorized_keys` for `sshd(8)`, e.g.

```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEnqsMuqOhEVw3HyWMp2fqqn6l1IZtJHD1UWkOXszUcl
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBBMu3TbOgxpvYrcQQG6VHSgrwMzAsFg2s+UX5JMNjNI
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKrOJrpYRgEiuGuoNhyPbeEldjIRkwRG/fjjySPUks/y
```

For an account on GitHub, you can use `https://github.com/<username>.keys` to generate it. For example, if I wanted to allow access for `robherley`:

```
curl https://github.com/robherley.keys > snips_authorized_keys
```

Alternatively, tools like [`ssh-import-id`](https://manpages.ubuntu.com/manpages/bionic/man1/ssh-import-id.1.html) can be used as well:

```
ssh-import-id gh:robherley -o snips_authorized_keys
```

### Statsd Metrics

At runtime, snips.sh will emit various metrics if the `SNIPS_METRICS_STATSD` is defined. This should be the full UDP address with the protocol, e.g. `udp://localhost:8125`.

### Build without File Type Detection

In order to "guess" what language a snippet is, snips.sh uses [magika-go](https://github.com/robherley/magika-go), a Go port of Google's Magika AI-powered file type detection system. The model and ONNX runtime are embedded at build time, so no external dependencies are required beyond CGO (for SQLite).

For local development, you can build as so:

```bash
script/build
```

If you do not want file type detection, you can build without the guesser (and avoid extra linking/env vars):

```bash
go build -tags noguesser .
```

## Examples

### Docker Compose

Create an environment file, e.g. `snips.env` with k/v pairs that'll look something like:

```bash
SNIPS_HTTP_EXTERNAL=https://snips.example.com
SNIPS_SSH_EXTERNAL=ssh://snips.example.com:22
SNIPS_HMACKEY=correct-horse-battery-staple
```

Then set up a `docker-compose.yml`:

```yaml
version: "3"
services:
  snips:
    image: 'ghcr.io/robherley/snips.sh:latest'
    restart: unless-stopped
    ports:
      - '80:8080'
      - '22:2222'
    volumes:
      - ./data:/data
    env_file:
      - snips.env
```
