<div align="center">

# snips.sh ✂️

**SSH-powered pastebin with a human-friendly TUI and web UI**

<p align="center">
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#examples">Examples</a> •
  <a href="#credits">Credits</a>
</p>

<img alt="tui" width="75%" src="https://vhs.charm.sh/vhs-1MRS4DCN6XUpxzM2PrqCfL.gif" />

</div>

### Features

  - ⚡ **Zero-install**: use from any machine with SSH client installed
  - 🌐 **Web UI**: share your syntax-highlighted code with short links
  - 💻 **TUI**: never leave your terminal for management/viewing
  - 🔑 **No passwords**: all you need is an SSH key
  - 🕵️ **Anonymous**: no sign ups, no logins, no email required
  - ⏰ **URLs with TTL**: time-bombed access for sensitive sharing
  - 📦 **Self-hostable**: containerized and light on resources
  - 🧠 **ML language detection**: uses [guesslang model](https://github.com/yoeo/guesslang) to identify source code


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

## Policy

By using snips.sh, you accept the [Terms of Service](/docs/TOS.md) and [Acceptable Use Policy](/docs/AUP.md)

## Credits

The technology behind snips.sh is powered by these amazing projects:

- [`charmbracelet`](https://github.com/charmbracelet)
  - [`charmbracelet/wish`](https://github.com/charmbracelet/wish): SSH server
  - [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea): TUI framework
- [`alecthomas/chroma`](https://github.com/alecthomas/chroma): Syntax Highlighter
- [`yuin/goldmark`](github.com/yuin/goldmark): Markdown Parser
