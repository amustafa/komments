# Komments

Non-inline code annotations. Leave comments on any line or range of code without modifying the source file — stored in a shared SQLite database and accessible from Neovim, the command line, and AI assistants via MCP.

---

## Why Komments?

Code review comments disappear after a PR merges. TODO comments clutter diffs and trigger linter warnings. Komments lives outside your source files — annotations are stored in a local database, tied to file paths and line numbers, and visible from multiple interfaces:

| Interface | Use case |
|-----------|----------|
| **CLI** (`komments`) | Scriptable interface — the single source of truth for all operations |
| **Neovim plugin** | Add/browse/edit comments while coding |
| **VS Code extension** | Sidebar tree view, gutter decorations, command palette |
| **MCP server** | AI assistants (Claude Code, Claude Desktop) read and triage comments |

All four share a single SQLite database. A comment added in Neovim is immediately visible in VS Code, the CLI, and MCP, and vice versa.

### Extending to other editors

The CLI with `--json` output is designed as a stable API. A JetBrains plugin, Emacs package, or any other integration follows the same pattern: a thin UI layer that shells out to `komments` and renders results in the editor's native UI.

---

## Architecture

The Go binary (`komments`) is the single implementation of all database logic. Everything else is a thin wrapper.

```
┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐
│ Neovim Plugin │ │VS Code Ext.   │ │  MCP Server   │ │ Your Tool     │
│    (Lua)      │ │ (TypeScript)  │ │ (TypeScript)  │ │ (any lang)    │
│               │ │               │ │               │ │               │
│vim.fn.system()│ │execFileSync() │ │execFileSync() │ │subprocess call│
│      │        │ │      │        │ │      │        │ │      │        │
└──────┼────────┘ └──────┼────────┘ └──────┼────────┘ └──────┼────────┘
       │                 │                 │                 │
       └─────────────────┼─────────────────┼─────────────────┘
                         │                 │
                                  │
                    ┌─────────────▼─────────────┐
                    │     komments CLI (Go)      │
                    │                            │
                    │  Single binary, zero deps  │
                    │  All DB logic lives here   │
                    └─────────────┬─────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │  ~/.local/komments/        │
                    │    comments.db  (SQLite)   │
                    │    WAL mode — concurrent   │
                    │    access safe             │
                    └───────────────────────────┘
```

### Database schema

```sql
CREATE TABLE comments (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  project_root  TEXT    NOT NULL,    -- absolute path, scopes comments per project
  timestamp     TEXT    NOT NULL,    -- ISO 8601 UTC
  file          TEXT    NOT NULL,    -- path relative to project_root
  position      TEXT    NOT NULL,    -- JSON: cursor or range (see below)
  text          TEXT    NOT NULL,    -- the annotation content
  archived      INTEGER NOT NULL DEFAULT 0
);
```

Each comment records the `project_root` it belongs to (detected as the git repo root, or cwd if not in a repo). Every interface auto-filters to the current project.

### Position types

```json
{ "type": "cursor", "line": 42, "col": 1 }

{ "type": "range", "start_line": 10, "start_col": 1,
  "end_line": 25, "end_col": 1 }
```

---

## Requirements

| Component | Requires |
|-----------|----------|
| CLI | Go >= 1.24 (build only — the resulting binary has no runtime deps) |
| Neovim plugin | Neovim >= 0.9, `komments` binary on PATH |
| VS Code extension | VS Code >= 1.85, `komments` binary on PATH |
| MCP server | Node.js >= 18, `komments` binary on PATH |

---

## Installation

### CLI (Go binary)

```bash
# Build
go build -o komments ./cmd/komments/

# Put it on your PATH
cp komments ~/.local/bin/    # or /usr/local/bin/, etc.
```

The binary is statically linked (pure-Go SQLite via `modernc.org/sqlite`) — no C compiler, no shared libraries, cross-compiles to any OS/arch.

### Neovim plugin

#### lazy.nvim (from GitHub)

```lua
{
  "your-user/komments.nvim",
  config = function()
    require("komments").setup()
  end,
}
```

#### lazy.nvim (local directory)

```lua
{
  dir = "/absolute/path/to/komments",
  config = function()
    require("komments").setup()
  end,
}
```

#### packer.nvim

```lua
use {
  "/absolute/path/to/komments",
  config = function()
    require("komments").setup()
  end,
}
```

#### Manual (no plugin manager)

```lua
vim.opt.rtp:prepend("/absolute/path/to/komments")
require("komments").setup()
```

#### Configuration

All options are optional — defaults shown:

```lua
require("komments").setup({
  bin = "komments",            -- path to komments binary
  keymap = "<leader>kc",       -- add comment (normal + visual)
  list_keymap = "<leader>kl",  -- open comment list
  ui = {
    input = {
      width = 60,              -- input window width (columns)
      height = 5,              -- input window height (lines)
      border = "rounded",
    },
    list = {
      width = 0.8,             -- fraction of editor width
      height = 0.6,            -- fraction of editor height
      border = "rounded",
    },
  },
})
```

If the `komments` binary isn't on your PATH, set `bin` to the absolute path.

### VS Code extension

```bash
cd vscode-komments
npm install
npm run build
```

To install locally for development:

```bash
code --extensionDevelopmentPath="$(pwd)/vscode-komments" .
```

Or symlink into your extensions directory:

```bash
ln -s "$(pwd)/vscode-komments" ~/.vscode/extensions/komments
```

#### Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `komments.bin` | `"komments"` | Path to the `komments` binary |

### MCP server

```bash
cd mcp-server
npm install
npm run build
```

The MCP server is a thin wrapper (~60 lines) that translates MCP tool calls into `komments` CLI invocations. It requires the `komments` binary to be on PATH (or set `KOMMENTS_BIN` env var).

#### Claude Code (automatic)

```bash
./scripts/install-claude.sh
```

This builds both the Go binary and MCP server, then registers the server in `~/.claude/settings.json` with the correct `KOMMENTS_BIN` path.

#### Claude Code (manual)

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "komments": {
      "command": "node",
      "args": ["/absolute/path/to/komments/mcp-server/dist/index.js"],
      "env": {
        "KOMMENTS_BIN": "/absolute/path/to/komments/komments"
      }
    }
  }
}
```

#### Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "komments": {
      "command": "node",
      "args": ["/absolute/path/to/komments/mcp-server/dist/index.js"],
      "env": {
        "KOMMENTS_BIN": "/absolute/path/to/komments/komments"
      }
    }
  }
}
```

---

## Usage

### CLI

```
komments list [--all] [--json]              List comments (active only by default)
komments add <file> <line> <text>           Add a comment at a specific line
komments add <file> <start>-<end> <text>    Add a comment on a line range
komments get <id> [--json]                  Show a single comment
komments edit <id> <text>                   Update a comment's text
komments archive <id>                       Archive a comment
komments unarchive <id>                     Unarchive a comment
komments delete <id>                        Permanently delete a comment
komments watch [--interval <seconds>]       Watch for changes and emit JSONL events
```

#### Examples

```bash
# Add a comment
komments add src/auth.ts 42 "TODO: add rate limiting here"

# Add on a range
komments add src/auth.ts 10-25 "This whole block needs refactoring"

# List active comments
komments list

# JSON output for scripting
komments list --json | jq '.[].text'

# Archive after addressing
komments archive 3
```

### Watch mode

`komments watch` polls the database and emits JSONL events to stdout whenever comments change:

```bash
komments watch                  # poll every 2 seconds (default)
komments watch --interval 5     # poll every 5 seconds
```

Each line is a JSON object with an `event` field:

```jsonl
{"event":"added","comment":{"id":7,"file":"src/auth.ts","position":{"type":"cursor","line":42,"col":1},"text":"Add rate limiting","archived":false,...}}
{"event":"edited","comment":{"id":7,...,"text":"Rate limiting added in PR #78",...}}
{"event":"archived","comment":{"id":7,...,"archived":true,...}}
{"event":"unarchived","comment":{"id":7,...,"archived":false,...}}
{"event":"deleted","comment":{"id":7,...}}
```

This is designed for integration with tools like Claude Code. For example, you could pipe watch output to a script that triggers AI review on each new comment, or feed it into a CI notification system.

### Neovim plugin

#### Adding comments

| Mode | Keymap | Action |
|------|--------|--------|
| Normal | `<leader>kc` | Add comment at cursor line |
| Visual | `<leader>kc` | Add comment on selected range |

A floating input window opens. Type your annotation, then:

- **`<C-CR>`** (insert mode) or **`<CR>`** (normal mode) — save
- **`q`** or **`<Esc>`** (normal mode) — cancel

#### Browsing comments

Open the list with **`<leader>kl`** or the `:Komments` command.

| Key | Action |
|-----|--------|
| `a` / `dd` | Archive comment under cursor |
| `u` | Unarchive comment |
| `<CR>` / `e` | Edit comment text |
| `gd` | Jump to comment location in source |
| `q` / `<Esc>` | Close list |

### VS Code extension

#### Sidebar tree view

The extension adds a **Komments** panel to the activity bar. It lists all comments for the current workspace with file path, line number, and a text preview. Click any comment to jump to its location.

Toolbar buttons:
- **Refresh** — reload comments from the database
- **Toggle Archived** — show/hide archived comments

Right-click context menu on each comment:
- Archive / Unarchive
- Edit
- Delete
- Go to Comment Location

#### Gutter decorations

When you open a file that has comments, a small indicator appears on the commented lines. Hover to see the comment text.

#### Commands (command palette)

| Command | Description |
|---------|-------------|
| `Komments: Add Comment at Cursor` | Add a comment at the current line |
| `Komments: Add Comment at Selection` | Add a comment on the selected range |
| `Komments: Refresh Comments` | Refresh the tree view and decorations |
| `Komments: Toggle Archived Comments` | Show/hide archived in the tree view |

Right-click in the editor also shows **Add Comment at Cursor** / **Add Comment at Selection**.

### MCP tools

The MCP server exposes three tools for AI assistants:

| Tool | Parameters | Description |
|------|------------|-------------|
| `list_comments` | `include_archived?: boolean` | List comments (active by default) |
| `get_comment` | `id: number` | Get a single comment by ID |
| `archive_comment` | `id: number` | Archive a comment after addressing it |

#### AI workflow

1. Add comments in Neovim or via CLI to flag code for review
2. Start a conversation with Claude Code / Claude Desktop
3. The assistant calls `list_comments` to see active annotations
4. It reads the referenced source code, addresses each comment
5. It calls `archive_comment` to mark each one as resolved

A Claude Code slash command is included at `commands/user:komments.md` that automates this workflow.

---

## Storage

All comments live in a single database:

```
~/.local/komments/comments.db
```

Created automatically on first use. SQLite WAL mode enables safe concurrent access from multiple processes (Neovim, CLI, MCP server running simultaneously).

### Backup

```bash
cp ~/.local/komments/comments.db ~/.local/komments/comments.db.bak

# Or export as JSON
komments list --all --json > komments-backup.json
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `KOMMENTS_PROJECT_ROOT` | Override automatic project root detection |
| `KOMMENTS_BIN` | Path to komments binary (used by MCP server) |

---

## Project structure

```
komments/
├── cmd/komments/
│   └── main.go              # CLI entry point — all commands including watch
├── internal/store/
│   └── store.go             # SQLite operations — the single implementation
├── go.mod
├── lua/komments/            # Neovim plugin (thin wrapper)
│   ├── init.lua             #   Plugin entry, keymaps, public API
│   ├── config.lua           #   User configuration
│   ├── store.lua            #   Calls komments CLI via vim.fn.system()
│   └── ui.lua               #   Floating windows (input, list, edit)
├── plugin/
│   └── komments.lua         #   Vim plugin loader, :Komments command
├── vscode-komments/         # VS Code extension (thin wrapper)
│   ├── src/
│   │   ├── extension.ts     #   Activate, register commands
│   │   ├── cli.ts           #   Calls komments CLI via execFileSync()
│   │   ├── treeView.ts      #   Sidebar tree view provider
│   │   ├── decorations.ts   #   Gutter decorations with hover tooltips
│   │   └── types.ts         #   Comment/Position interfaces
│   └── package.json
├── mcp-server/              # MCP server (thin wrapper)
│   ├── src/
│   │   └── index.ts         #   Calls komments CLI via execFileSync()
│   └── package.json
├── commands/
│   └── user:komments.md     #   Claude Code slash command
├── scripts/
│   └── install-claude.sh    #   Builds + registers for Claude Code
└── README.md
```

---

## License

MIT
