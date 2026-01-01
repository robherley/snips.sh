<div align="center">

# snips.sh ✂️

**SSH-powered pastebin with a human-friendly TUI and web UI**

<p align="center">
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#examples">Examples</a> •
  <a href="#docs">Docs</a> •
  <a href="#credits">Credits</a>
</p>

<img alt="tui" width="85%" src="https://vhs.charm.sh/vhs-1MRS4DCN6XUpxzM2PrqCfL.gif" />

</div>

### Features

  - ⚡ **Zero-install**: use from any machine with SSH client installed
  - 🌐 **Web UI**: syntax-highlighted code with short links and Markdown rendering
  - 💻 **TUI**: never leave your terminal for snippet management/viewing
  - 🔑 **No passwords**: all you need is an SSH key
  - 🕵️ **Anonymous**: no sign ups, no logins, no email required
  - ⏰ **URLs with TTL**: time-limited access for sensitive sharing
  - 📦 **Self-hostable**: containerized and light on resources
  - 🧠 **ML language detection**: intelligently identify source code


## Quick Start

If you have an SSH key, you can upload:

```
echo '{ "hello" : "world" }' | ssh snips.sh
```

To access the TUI:

```
ssh snips.sh
```

## Examples

<div align="center">

<table>
  <tr align="center">
    <td>Upload from any machine, no install necessary.</td>
  </tr>
  <tr align="center">
    <td>
      <img alt="upload" width="600" src="https://vhs.charm.sh/vhs-2GYlJ8Fh4RYnYpN141jgtT.gif" />
    </td>
  </tr>
  <tr align="center">
    <td>Download files and pipe into your favorite <code>$PAGER</code>. </td>
  </tr>
  <tr align="center">
    <td>
      <img alt="download" width="600" src="https://vhs.charm.sh/vhs-7j0LzNCGaBjF6v91QkXJgr.gif" />
    </td>
  </tr>
  <tr align="center">
    <td>Something secret to share? Create a temporary, time-bound URL for restricted access.</td>
  </tr>
  <tr align="center">
    <td>
      <img alt="private" width="600" src="https://vhs.charm.sh/vhs-52eZOU1lp0y0ZwUFN6lkUm.gif" />
    </td>
  </tr>
</table>

</div>

## Docs

- [Contributing](/docs/contributing.md): How you can contribute to snips.sh
- [Database](/docs/database.md): How snips.sh stores it's data
- [Self Hosting](/docs/self-hosting.md): How to host your own instance of snips.sh
- [Terms of Service](/docs/terms-of-service.md): What we (snips.sh provider) and you can/can't do
- [Acceptable Use Policy](/docs/acceptable-use-policy.md): What you can/can't upload

## Contributors

<a href="https://github.com/robherley/snips.sh/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=robherley/snips.sh" />
</a>

## Credits

The technology behind snips.sh is powered by these amazing projects:

- [`charmbracelet`](https://github.com/charmbracelet)
  - [`charmbracelet/wish`](https://github.com/charmbracelet/wish): SSH server
  - [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea): TUI framework
- [`google/magika`](https://github.com/google/magika): AI-powered file type detection
- [`alecthomas/chroma`](https://github.com/alecthomas/chroma): Syntax Highlighter
- [`yuin/goldmark`](https://github.com/yuin/goldmark): Markdown Parser
- [`microcosm-cc/bluemonday`](https://github.com/microcosm-cc/bluemonday): HTML Sanitizer
- [`tdewolff/minify`](https://github.com/tdewolff/minify): Minifier
