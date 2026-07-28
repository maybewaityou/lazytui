# Template guide — deriving a new lazy-series tool from lazytui

`lazytui` is the reference template for the `lazy*` TUI series (lazyssh / lazytmux /
fleetboard family). It ships a `scripts/scaffold.sh` that rewrites the four
placeholder tokens into a new tool's identifiers, so you start from a project
that already compiles and renders. This guide is the hand-written part the
script deliberately does **not** automate: the business-specific touches that
make the new tool actually do something.

---

## 0. Scaffold a compiling baseline

```bash
./scripts/scaffold.sh --name lazybookmark --entity Bookmark --dir ../lazybookmark
cd ../lazybookmark
go mod tidy && go build ./...      # must exit 0 before you touch anything
make run                            # smoke-test the empty UI
```

What the script rewrote (and what it left alone):

| token             | became (example)        | where                                  |
|-------------------|-------------------------|----------------------------------------|
| `github.com/maybewaityou/lazytui` | `.../lazybookmark` | `go.mod`, every `import`               |
| `LAZYTUI`         | `LAZYBOOKMARK`          | logger prefix (`cmd/main.go`)          |
| `Item`            | `Bookmark`              | `domain.Bookmark`, `BookmarkList`, ... |
| `items` / `item`  | `bookmarks` / `bookmark`| field names, file-local helpers        |
| `lazytui`         | `lazybookmark`          | binary name (`makefile`), `~/.lazybookmark` state dir, `AppName` |

The script **preserves tview's own API** (`AddItem`, `GetCurrentItem`,
`GetFormItem`, `GetFocusedItemIndex`, `GetItemCount`, `GetItemText`,
`SetCurrentItem`, `SetItemText`) by shielding them behind throwaway tokens for
the rewrite pass — so the generated project compiles without manual API fixes.

What the script does **not** do, and you must by hand:

- Rename the `item_*.go` source files to `bookmark_*.go` (optional; Go ignores
  file names at build time, but matching names keep the tree readable).
- Add domain fields, detail sections, form fields, keybindings — see steps 1-4.
- Swap the JSON `store` for your real backend — see step 5.

The five steps below are ordered: each leaves the project compiling.

---

## 1. Shape the entity — `internal/core/domain/<entity>.go`

The scaffolded entity is a 1:1 copy of `lazytui`'s generic `Item`:

```go
type Bookmark struct {
    Name    string
    Tags    []string
    Note    string
    Pinned  bool
    Created time.Time
}
```

Add the fields your tool actually manages (a URL for bookmarks, a host:port for
SSH targets, a session id for tmux, ...). Keep `Name` — it is the unique key the
whole stack keys on (`Repository.Rename` uses mv semantics on it). Remove fields
you don't need; the form/details steps below will stop referencing them.

After editing, `go build ./...` will fail in the places that still reference the
old field set — that's the work list for steps 2 and 3.

---

## 2. Render the details pane — `internal/adapters/ui/<entity>_details.go`

`<Entity>Details.Render` emits a slice of `Section{Title, Fields}` (the
`Section`/`Field` helpers live in `render.go`). The template ships two groups:
`Basic` (name / pinned / created) and an optional `Metadata` group (tags + note).

To model your entity:

- Add a `Field` per new domain field under the right `Section`. `Field.Kind`
  selects rendering (`FieldText`, `FieldURL`, `FieldCode`, ... — see `render.go`).
- Add a whole new `Section` when a group of fields belongs together (e.g. a
  `Connection` section for SSH: host, port, user). Sections render in slice
  order with a titled header, so order here == order on screen.
- Keep the "Metadata section only when non-empty" pattern so a bare entry shows
  one group, not a hollow two-group layout.

`Render` stays a pure function of the entity — no I/O, no service calls. That is
what makes it unit-testable without a running backend.

---

## 3. Wire the create/edit form — `internal/adapters/ui/<entity>_form.go`

The form is a `Name / Tags / Note` modal. Field order is fixed by the `iota`
block at the top of the file:

```go
const (
    mfFieldName = iota
    mfFieldTags
    mfFieldNote
)
```

To add a field (say `URL`):

1. Insert `mfFieldURL` into the `iota` block in the position that matches the
   visual order.
2. Add the widget in `new<Entity>Form` (an `InputField` for short text, a
   `TextArea` for long) at the same position via `form.AddFormItem`.
3. Update `submit` to read the value off the form (by index, using the `iota`
   constant) and write it onto the entity before `Update`.
4. Update `setValues` (edit path) to pre-fill the field from the existing entity.
5. Re-check `modalColumnHeight` / `itemModalWidth` constants — they size the
   centered column and assume a fixed field count; adding a row means bumping the
   height.

`noteVisibleRows`, `modalColumnHeight`, and `itemModalWidth` are the sizing
knobs; the comment block above them explains the constraint (Name + Tags + Note
must all fit without scrolling the modal itself).

---

## 4. Adjust keybindings — `internal/adapters/ui/keybindings.go`

`keyBindings` is the **single source of truth**: the help modal, the one-line
footer hint, and the docs are all derived from it (see `status_bar.go`), so you
add a key exactly once here and it propagates everywhere.

```go
var keyBindings = []KeyBinding{
    {"Navigate", "j/k", "Move", true},
    ...
    {"Item", "a", "New", true},     // <- the "Item" group label is cosmetic;
    {"Item", "e", "Edit", true},    //    rename it to "Bookmark" etc. if you like
    ...
}
```

Notes:

- `Group` doubles as the help modal's section header. The
  `TestKeyBindingsGroupsContiguous` test forbids non-adjacent groups, so keep
  entries for one `Group` contiguous in the slice.
- `Footer: true` also surfaces the binding in the footer single-line hint; keep
  that line to the 5-6 most important keys or it overflows.
- The action handlers themselves live in `handlers.go` (the `switch` on the key);
  adding a key here without a handler case is a no-op, not a crash.

---

## 5. Plug in a real backend — implement `ports.Repository`

Everything above is UI; the data plane is one interface:

```go
// internal/core/ports/repository.go
type Repository interface {
    List() ([]domain.Bookmark, error)
    Create(name string) (domain.Bookmark, error)
    Update(name string, item domain.Bookmark) error
    Delete(name string) error
    Rename(oldName, newName string) error
}
```

The template ships a JSON-file implementation in
`internal/adapters/data/store/` (`store.go`) — fine for a local CRUD tool, and
the default `cmd/main.go` wires it in at the composition root:

```go
repo := store.New(dataPath)
svc := services.NewBookmarkService(repo)
```

To talk to a real system (SSH, tmux, an HTTP API, ...), write a new adapter that
satisfies `ports.Repository` and swap the one line in `cmd/main.go`. `core`
(domain / ports / services) never imports the adapter, so the business logic is
unaffected by where the data lives. This is the whole reason the adapter boundary
exists — keep `store` around as a reference and as a fallback for tests.

---

## TL;DR checklist

- [ ] `scaffold.sh` ran, `go build ./...` is green
- [ ] (optional) rename `item_*.go` files to match the new entity
- [ ] `domain.<Entity>` has the fields the tool actually manages
- [ ] `<entity>_details.Render` shows every field, grouped into sections
- [ ] `<entity>_form` can create/edit every field
- [ ] `keybindings.go` advertises every key you added a handler for
- [ ] `ports.Repository` is implemented against the real backend (or the JSON
      store is acceptable for v1), and `cmd/main.go` wires it in
- [ ] `make test` passes, `make run` exercises the real flow
