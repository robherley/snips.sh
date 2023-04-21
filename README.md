<div align="center">

# snips.sh ✂️

**SSH-powered pastebin with a human-friendly TUI and web UI**

</div>

<img alt="tui" align="right" src="https://vhs.charm.sh/vhs-3jcrPS4PG7r0ELPcqFXDDw.gif" width="75%" />

### Features

- ⚡ Zero-install
- 🌐 Web UI
- 💻 TUI
- 🔑 No passwords
- 🕵️ Anonymous
- ⏰ URLs with TTL
- 🖨️ Self-hostable


<br>
<br>
<br>
<br>
<br>


## Getting Started 🎯

If you have an SSH key, you can upload:

```
echo '{ "hello" : "world" }' | ssh snips.sh
```

To access the TUI:

```
ssh snips.sh
```

## Examples 👀

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

## Acknowledgements 🏆

The technology behind snips.sh is powered by these amazing projects:

- [`charmbracelet`](https://github.com/charmbracelet)
  - [`charmbracelet/wish`](https://github.com/charmbracelet/wish): SSH server
  - [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea): TUI framework
- [`alecthomas/chroma`](https://github.com/alecthomas/chroma): Syntax Highlighter
- [`yuin/goldmark`](github.com/yuin/goldmark): Markdown Parser
