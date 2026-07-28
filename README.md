<div align="center">

# lazytui

A general-purpose, keyboard-driven **local CRUD TUI template** — the scaffold for the lazy-series tools (lazyssh / lazytmux / ...).

**[English](./README.md)** | [简体中文](./README.zh-CN.md)

</div>

---

`lazytui` is a template, not a product. It ships a complete, opinionated TUI for managing a list of local items — list, search, filter, create, edit, delete, pin, tag — backed by an atomic JSON store and wrapped in a hexagonal architecture. Run it as-is to manage generic `Item` records, or point the scaffold script at a new directory to generate a purpose-built tool (bookmarks, snippets, hosts, ...) in minutes.

Every key binding in the UI, the in-app help panel (`?`), the footer hint line, and the table below is derived from a single source of truth: the `keyBindings` slice in [`internal/adapters/ui/keybindings.go`](./internal/adapters/ui/keybindings.go). Add a key once, there, and every surface stays in sync.

---

## ✨ Features

### Item Management
- 📜 List items from the local JSON store with pinned favorites kept at the top.
- ➕ Create new items from the UI.
- ✏️ Edit items (name, tags, note) in place.
- 🗑️ Delete items safely (with confirmation).
- 📌 Pin / unpin favorites to keep them at the top.
- 🏷️ Tag items to group and find them later.

### Quick Navigation
- 🔍 Fuzzy search by item name.
- ↕️ Pinned items always rise to the top.
- 🏷️ Filter the list by tag (`f`, multi-select, OR‑matched).
- 🧩 Details pane with grouped, labeled sections per item.

### Workflow
- 💾 Atomic JSON store at `~/.lazytui/items.json` — a crashed run never leaves a half-written file.
- 🔄 Background refresh of list state.
- ❓ In-app help (`?`) — every key binding in a two-column, grouped panel, drawn from the same slice as this README.

---

## 🔒 How it works

`lazytui` is a pure Go binary with **no external runtime dependencies**. Unlike `lazytmux` (which shells out to the system `tmux`), it reads and writes only its own state:

- Item data lives in `~/.lazytui/items.json`.
- Logs go to `~/.lazytui/lazytui.log`.
- All writes are atomic (write-to-temp-then-rename), so a crash never corrupts the store.

Because there is no system-binary dependency, the Homebrew formula declares **no** `depends_on` — `brew install` is self-contained.

---

## 📷 Screenshots

> Screenshots live in [`./docs/resources/`](./docs/resources/) and are optional — the directory may be empty in a fresh template checkout. Drop `list.png`, `search.png`, `help.png`, etc. in there to populate this section.

<div align="center">

| Dashboard | Help |
| --- | --- |
| <i>./docs/resources/list.png</i> | <i>./docs/resources/help.png</i> |

</div>

---

## 📦 Installation

### Option 1: Homebrew (macOS)

```bash
brew install maybewaityou/tap/lazytui
```

`lazytui` is a self-contained Go binary, so Homebrew installs nothing else.

> **Newer Homebrew (5.1.15+/6.0):** third-party taps are untrusted by default. If install fails with `Refusing to load formula ... from untrusted tap`, trust the tap first (one-time):
>
> ```bash
> brew trust maybewaityou/tap
> ```

### Option 2: `go install`

```bash
go install github.com/maybewaityou/lazytui/cmd@latest
```

### Option 3: Download Binary from Releases

Download a prebuilt binary from [GitHub Releases](https://github.com/maybewaityou/lazytui/releases). The snippet below detects the latest version and fetches the right tarball for your OS/arch (darwin/linux × amd64/arm64):

```bash
# Detect latest version
LATEST_TAG=$(curl -fsSL https://api.github.com/repos/maybewaityou/lazytui/releases/latest | jq -r .tag_name)

# Normalize OS/arch to the release asset name (darwin/linux × amd64/arm64)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
esac

# Download + extract + install
curl -LJO "https://github.com/maybewaityou/lazytui/releases/download/${LATEST_TAG}/lazytui_${OS}_${ARCH}.tar.gz"
tar -xzf lazytui_${OS}_${ARCH}.tar.gz
sudo mv lazytui /usr/local/bin/

# enjoy!
lazytui
```

### Option 4: Build from Source

```bash
# Clone the repository
git clone https://github.com/maybewaityou/lazytui.git
cd lazytui

# Build (runs fmt + go vet first)
make build
sudo mv bin/lazytui /usr/local/bin/

# Or run it directly without installing
make run
```

### Snapshot binaries (optional)

`make build-all` produces cross-compiled snapshots via [goreleaser](https://goreleaser.com) (linux/darwin × amd64/arm64):

```bash
make build-all
```

---

## ⌨️ Key Bindings

The table below is grouped exactly as the `keyBindings` slice in [`internal/adapters/ui/keybindings.go`](./internal/adapters/ui/keybindings.go) is grouped, and shows every entry in that slice. The in-app `?` help panel and the footer hint line are derived from the same slice — this README is the third consumer, so the three can never drift.

### Navigate

| Key     | Action              |
| ------- | ------------------- |
| `↑↓`    | Move                |
| `Enter` | Open details        |
| `←/→`   | Focus list/details  |
| `/`     | Search              |
| `q`     | Quit                |

### Item

| Key | Action   |
| --- | -------- |
| `a` | New      |
| `e` | Edit     |
| `d` | Delete   |
| `r` | Refresh  |

### Filter

| Key | Action          |
| --- | --------------- |
| `f` | Filter by tag   |

### Metadata

| Key | Action         |
| --- | -------------- |
| `p` | Pin / unpin    |
| `n` | Edit note      |
| `c` | Clear tags     |

### Other

| Key | Action                     |
| --- | -------------------------- |
| `?` | Help                       |

**In the item form:**

| Key             | Action              |
| --------------- | ------------------- |
| `↑↓`            | Switch field        |
| `Tab/Shift+Tab` | Move between fields |
| `Enter`         | Submit (save)       |
| `Shift+Enter`   | Newline (in Note)   |
| `Esc`           | Cancel              |

Tip: the status bar at the bottom shows the result of your last action.

---

## 🏗 Architecture

Hexagonal (ports & adapters):

```
cmd/main.go                       → cobra root, wires deps
internal/core/domain/             → Item model
internal/core/ports/              → Repository / Service ports
internal/core/services/           → business logic
internal/adapters/data/store/     → ~/.lazytui/items.json (atomic JSON CRUD)
internal/adapters/ui/             → tview TUI (Tokyo Night)
internal/logger/                  → zap → ~/.lazytui/lazytui.log
```

The store adapter is a generic JSON CRUD engine; to repurpose `lazytui` for a non-local backend, implement `ports.Repository` and swap the adapter in `cmd/main.go`.

---

## 🧰 Use as a template (scaffold a new tool)

`lazytui` ships a scaffold script that copies this tree and rewrites the placeholder tokens (`lazytui` → your tool, `Item` → your entity) to generate a new lazy-series tool.

```bash
./scripts/scaffold.sh --name <tool> --entity <Entity> --dir <target>
```

Example — generate a `lazybookmark` tool whose entity is `Bookmark`:

```bash
./scripts/scaffold.sh --name lazybookmark --entity Bookmark --dir ../lazybookmark
cd ../lazybookmark
go mod tidy && go build ./...
```

Arguments:

| Flag       | Meaning                                                          |
| ---------- | ---------------------------------------------------------------- |
| `--name`   | lower-case tool name, e.g. `lazybookmark` (module suffix + binary) |
| `--entity` | PascalCase entity name, e.g. `Bookmark` (domain type; plural = `+s`) |
| `--dir`    | target directory to generate (created if missing)                |

The rewrite order is significant (module path → `LAZYTUI` → `Item` → `items` → `item` → `lazytui`), and tview's own `*Item*` API methods are shielded behind throwaway tokens so they are never mis-rewritten. See [`docs/template-guide.md`](./docs/template-guide.md) for the full customization checklist (domain fields, details sections, form fields, keybindings, repository adapter).

---

## 🏗 Build

```bash
make build       # fmt + vet + build → bin/lazytui
make run         # go run with version ldflags
make test        # race + cover
make lint        # golangci-lint
make build-all   # goreleaser cross-compile snapshot
```

---

## 🤝 Contributing

Contributions are welcome!

- If you spot a bug or have a feature request, please [open an issue](https://github.com/maybewaityou/lazytui/issues).
- If you'd like to contribute, fork the repo and submit a pull request.

### Semantic commit messages

This project follows semantic commits. Please format your commit/PR title as:

- `type(scope): short descriptive subject`

Common types: `feat`, `fix`, `improve`, `refactor`, `docs`, `test`, `ci`, `chore`, `revert`.
Scope is optional (e.g. `ui`, `cli`, `core`).

---

## 📄 License

Licensed under the [Apache License 2.0](./LICENSE).

---

## 🙏 Acknowledgments

- Built with [tview](https://github.com/rivo/tview) + [tcell](https://github.com/gdamore/tcell), [cobra](https://github.com/spf13/cobra), and [zap](https://go.uber.org/zap).
- Architecture and UX language inherited from the lazy-series tools ([lazyssh](https://github.com/Adembc/lazyssh), lazytmux).
- Theme: Tokyo Night.
