# memos-tui

A terminal UI client for [Memos](https://github.com/usememos/memos), the self-hosted note-taking app.

Browse, create, edit, and manage your memos without leaving the terminal. Uses your own `$EDITOR` for writing.

## Installation

Download a pre-built binary from the [releases page](https://github.com/yeniklas/memos-tui/releases), or install from source:

```sh
go install github.com/yeniklas/memos-tui@latest
```

## Configuration

Create `~/.config/memos-tui/config.toml`:

```toml
default_profile = "default"

[profiles.default]
url   = "https://your-memos-host"
token = "memos_pat_..."
```

Get your API token from **Memos → Settings → My account → Access tokens**.

### Multiple profiles

```toml
default_profile = "home"

[profiles.home]
url   = "https://notes.example.com"
token = "memos_pat_..."

[profiles.work]
url   = "https://work.example.com"
token = "memos_pat_..."
```

Switch profiles with `memos-tui --profile work`.

### Profile options

| Key | Default | Description |
|---|---|---|
| `url` | — | Base URL of your Memos instance |
| `token` | — | Personal access token |
| `markdown` | `true` | Render markdown in the preview panel |
| `journal_tags` | `["diary"]` | Tags applied to journal entries |

## Usage

```sh
memos-tui                   # launch with default profile
memos-tui --profile work    # launch with a specific profile
memos-tui --journal         # open or create today's journal entry
memos-tui --version         # print version and exit
```

## Keybindings

| Key | Action |
|---|---|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `PgDn` / `PgUp` | Page down / up |
| `g` / `G` | Jump to top / bottom |
| `Tab` | Toggle preview panel |
| `/` | Search |
| `f` | Filter by tag |
| `C` | Clear all filters |
| `c` | Calendar filter |
| `n` | New memo (opens `$EDITOR`) |
| `e` | Edit selected memo (opens `$EDITOR`) |
| `d` | Delete selected memo |
| `p` | Pin / unpin |
| `v` | Cycle visibility (private → protected → public) |
| `a` | Archive |
| `r` | Refresh |
| `q` / `Ctrl+C` | Quit |

## Theming

Colors can be customized in `~/.config/memos-tui/theme.toml`. Each field accepts an ANSI 256 color number (`0`–`255`) or a hex color (`#RGB` / `#RRGGBB`). Omit a field to keep the default.

```toml
primary  = "63"   # focused panel border
muted    = "241"  # unfocused borders and help bar
selected = "212"  # selected row highlight
date     = "246"  # date column
tag      = "79"   # tag chips
error    = "160"  # error messages
pinned   = "220"  # pinned memo indicator
```

A [Gruvbox dark theme](themes/gruvbox.toml) is included in the repository.

## Journal mode

`memos-tui --journal` opens today's journal entry in `$EDITOR`. If an entry for today already exists it is loaded for editing; otherwise a new one is created with today's date as the heading and your configured `journal_tags` applied.

```toml
journal_tags = ["diary", "daily"]
```
