# Usage

snips.sh is an SSH-driven snippet manager. All interactions happen through your terminal — upload, download, edit, delete, and share files using standard SSH commands. No account creation needed; your SSH key is your identity.

- [Usage](#usage)
  - [Quick reference](#quick-reference)
  - [Authentication](#authentication)
  - [Uploading](#uploading)
    - [Private uploads](#private-uploads)
    - [Limits](#limits)
  - [Downloading](#downloading)
  - [Updating content](#updating-content)
  - [Deleting](#deleting)
  - [Signed URLs](#signed-urls)
    - [Duration format](#duration-format)
  - [Interactive TUI](#interactive-tui)
  - [Web access](#web-access)

## Quick reference

| Action | Command |
|--------|---------|
| Upload | `echo "content" \| ssh snips.sh` |
| Upload (private) | `echo "content" \| ssh snips.sh -private` |
| Upload (with type hint) | `echo "content" \| ssh snips.sh -ext py` |
| Upload (private + signed URL) | `echo "content" \| ssh snips.sh -private -ttl 24h` |
| Download | `ssh f:<id>@snips.sh` |
| Update | `echo "new" \| ssh f:<id>:content@snips.sh` |
| Delete | `ssh f:<id>@snips.sh rm` |
| Force delete | `ssh f:<id>@snips.sh rm -f` |
| Sign | `ssh f:<id>@snips.sh sign -ttl 1h` |
| Interactive TUI | `ssh snips.sh` |

## Authentication

snips.sh uses SSH public key authentication exclusively. The first time you connect with a key, a user account is automatically created and linked to your key fingerprint. All files you create are tied to that key.

If your server has an authorized keys file configured, only listed keys will be allowed to connect.

## Uploading

Pipe any content to the SSH server to create a new snippet:

```
echo "Hello, world!" | ssh snips.sh
cat main.go | ssh snips.sh
curl -s https://example.com | ssh snips.sh
```

The server auto-detects the file type from the content. To override detection, pass an extension hint:

```
cat config | ssh snips.sh -ext yaml
```

### Private uploads

By default, files are public. To upload a private file:

```
echo "secret" | ssh snips.sh -private
```

Private files are only accessible by the owner (via their SSH key) or through signed URLs.

You can combine `-private` with `-ttl` to get a signed URL back immediately:

```
echo "secret" | ssh snips.sh -private -ttl 24h
```

### Limits

- **Max file size:** 1 MB (default)
- **Max files per user:** 100 (default)
- Empty files are rejected.

## Downloading

Retrieve a file by connecting as `f:<id>`:

```
ssh f:abc123@snips.sh
```

Output goes to stdout, so you can pipe it:

```
ssh f:abc123@snips.sh > local_copy.txt
ssh f:abc123@snips.sh | less
```

Private files can only be downloaded by their owner.

## Updating content

Pipe new content to `f:<id>:content` to replace a file's contents:

```
echo "updated content" | ssh f:abc123:content@snips.sh
```

You can also change the file type during an update:

```
cat renamed.py | ssh f:abc123:content@snips.sh -ext py
```

Only the file owner can update content. Each update creates a revision with a unified diff of the changes (for text files). Old revisions are pruned once the limit (default 64, but configurable) is reached.

## Deleting

Delete a file with the `rm` command:

```bash
ssh f:abc123@snips.sh rm
```

This prompts for confirmation. To skip the prompt:

```bash
ssh f:abc123@snips.sh rm -f
```

Only the file owner can delete their files.

## Signed URLs

Private files can be shared via time-limited signed URLs. Use the `sign` command with a `-ttl` duration:

```bash
ssh f:abc123@snips.sh sign -ttl 1h
ssh f:abc123@snips.sh sign -ttl 7d
```

The returned URL can be opened by anyone until it expires. Signing only works on private files.

### Duration format

Durations support these units, and can be combined:

| Unit | Meaning |
|------|---------|
| `s`  | seconds |
| `m`  | minutes |
| `h`  | hours   |
| `d`  | days    |
| `w`  | weeks   |

Examples: `30s`, `2h30m`, `1w2d`, `7d`

## Interactive TUI

Connect without piping to open an interactive terminal UI:

```bash
ssh snips.sh
```

The TUI lets you browse your files, view contents, see revision history, delete files, and generate signed URLs. Sessions have a default timeout of 15 minutes.

## Web access

Public files are also available over HTTP:

```
https://snips.sh/f/<id>
```

The web view includes syntax highlighting, metadata, and revision history. Private files require a valid signed URL to access over HTTP.
