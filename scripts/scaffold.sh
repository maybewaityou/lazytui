#!/usr/bin/env bash
# scaffold.sh — generate a new lazy-series TUI tool from the lazytui template.
#
# It copies the template tree, then rewrites the placeholder tokens:
#   github.com/maybewaityou/lazytui  ->  github.com/maybewaityou/<name>   (module path)
#   LAZYTUI                          ->  <NAME_UPPER>                    (logger prefix / app token)
#   Item                             ->  <Entity>                        (entity type: ItemList -> <Entity>List)
#   items                            ->  <entity>s                       (plural, e.g. demos)
#   item                             ->  <entity>                        (singular)
#   lazytui                          ->  <name>                          (binary / state dir, last)
#
# The catch the naive brief misses: tview's own API also carries "Item"
# (AddItem, GetCurrentItem, GetFormItem, GetFocusedItemIndex, GetItemCount,
# GetItemText, SetCurrentItem, SetItemText). A blind Item->Entity pass would
# rename those to Add<Entity>/GetCurrent<Entity>/... and break compilation.
# Word-boundary anchors (\b) would protect them but are not portable to BSD
# sed on macOS. Instead we shield each tview method behind a throwaway token
# before the entity rules run, then restore it — fully portable sed -i.bak.
set -euo pipefail

usage() {
  cat <<EOF
Usage: $0 --name <tool> --entity <Entity> --dir <target>
Example: $0 --name lazybookmark --entity Bookmark --dir ../lazybookmark

  --name     lower-case tool name, e.g. lazybookmark (module suffix + binary)
  --entity   PascalCase entity name, e.g. Bookmark (domain type; plural = +s)
  --dir      target directory to generate (created if missing)

Files are rewritten in place via sed -i.bak (portable across BSD/GNU sed);
*.bak backups are removed after each rewrite.
EOF
  exit "${1:-1}"
}

NAME="" ENTITY="" DIR=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --name) NAME="$2"; shift 2;;
    --entity) ENTITY="$2"; shift 2;;
    --dir) DIR="$2"; shift 2;;
    -h|--help) usage 0;;
    *) echo "unknown argument: $1" >&2; usage;;
  esac
done
[[ -n "$NAME" && -n "$ENTITY" && -n "$DIR" ]] || usage

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="$(cd "$SCRIPT_DIR/.." && pwd)"

ENTITY_LOWER="$(printf '%s' "$ENTITY" | tr '[:upper:]' '[:lower:]')"
NAME_UPPER="$(printf '%s' "$NAME" | tr '[:lower:]' '[:upper:]')"

# --- tview (and friends) method names that embed "Item" but must NOT be ---
# --- renamed. Longest first so prefix overlaps resolve cleanly.         ---
TVIEW_METHODS=(
  GetFocusedItemIndex
  GetFormItemCount
  GetFormItem
  GetCurrentItem
  SetCurrentItem
  GetItemText
  SetItemText
  GetItemCount
  AddItem
)

mkdir -p "$DIR"

# Copy the template, dropping local-only / build artefacts. scripts/ is kept
# (it carries this scaffold); the generated copy is rewritten like everything
# else, so it can scaffold the next tool in the series.
rsync -a \
  --exclude='.git' \
  --exclude='bin' \
  --exclude='.superpowers' \
  --exclude='docs/superpowers' \
  --exclude='.DS_Store' \
  "$SRC/" "$DIR/"

# Build the sed argument list once: protect -> rewrite -> restore.
protect_args=()
restore_args=()
for i in "${!TVIEW_METHODS[@]}"; do
  method="${TVIEW_METHODS[$i]}"
  tok="@TV${i}@"
  protect_args+=(-e "s|${method}|${tok}|g")
  restore_args+=(-e "s|${tok}|${method}|g")
done

# Rewrite every text source file. Whitelist extensions so we never touch
# binaries, vendor blobs, or the .bak files mid-flight. go.mod carries the
# module path and MUST be included or imports won't resolve after tidy.
find "$DIR" -type f \( \
  -name '*.go' -o \
  -name '*.mod' -o \
  -name '*.sum' -o \
  -name '*.md' -o \
  -name '*.yaml' -o \
  -name '*.yml' -o \
  -name '*.sh' -o \
  -name 'makefile' -o \
  -name 'Makefile' -o \
  -name '.gitignore' \
\) -not -path '*/.git/*' -print0 |
while IFS= read -r -d '' f; do
  # Single sed pass: shield tview APIs, run entity rewrite, restore APIs.
  # sed -i.bak works on both BSD (macOS) and GNU (Linux) sed; rm cleans up.
  sed -i.bak \
    "${protect_args[@]}" \
    -e "s|github.com/maybewaityou/lazytui|github.com/maybewaityou/${NAME}|g" \
    -e "s|LAZYTUI|${NAME_UPPER}|g" \
    -e "s|Item|${ENTITY}|g" \
    -e "s|items|${ENTITY_LOWER}s|g" \
    -e "s|item|${ENTITY_LOWER}|g" \
    -e "s|lazytui|${NAME}|g" \
    "${restore_args[@]}" \
    "$f"
  rm -f "${f}.bak"
done

echo "Generated $DIR"
echo "Next:"
echo "  cd $DIR && go mod tidy && go build ./..."
echo "Then customize (see docs/template-guide.md):"
echo "  - domain.<Entity> fields"
echo "  - <entity>_details.Render sections / fields"
echo "  - <entity>_form fields"
echo "  - keybindings"
echo "  - ports.Repository adapter (replace JSON store with your backend)"
