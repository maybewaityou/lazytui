#!/usr/bin/env bash
# scaffold_test.sh — correctness test for scaffold.sh.
#
# Generates a lazydemo project from the template into a temp dir, then asserts:
#   1. generation exits 0
#   2. `go mod tidy && go build ./...` succeeds (compiles cleanly)
#   3. no `lazytui` residue in any .go file (rewrite is total)
#   4. the entity type name is consistent (domain.Demo exists; domain.Item gone)
#
# Run: ./scripts/scaffold_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$SCRIPT_DIR/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

TARGET="$TMP/lazydemo"
echo "== [1/4] generate lazydemo into $TARGET =="
"$REPO/scripts/scaffold.sh" --name lazydemo --entity Demo --dir "$TARGET"

echo "== [2/4] go mod tidy + go build ./... =="
( cd "$TARGET" && go mod tidy && go build ./... )

echo "== [3/4] no 'lazytui' residue in *.go =="
residue="$(grep -rn "lazytui" --include='*.go' "$TARGET" || true)"
if [[ -n "$residue" ]]; then
  echo "FAIL: lazytui residue found in generated .go files:" >&2
  printf '%s\n' "$residue" >&2
  exit 1
fi
echo "ok: zero matches"

echo "== [4/4] entity type consistency (domain.Demo present, domain.Item absent) =="
if ! grep -rq "type Demo struct" "$TARGET/internal/core/domain/"; then
  echo "FAIL: expected 'type Demo struct' in internal/core/domain/" >&2
  exit 1
fi
if grep -rq "type Item struct" "$TARGET/internal/core/domain/"; then
  echo "FAIL: stale 'type Item struct' still present in internal/core/domain/" >&2
  exit 1
fi
if grep -rq "domain\.Item" "$TARGET" --include='*.go'; then
  echo "FAIL: stale 'domain.Item' reference still present" >&2
  exit 1
fi
echo "ok: domain.Demo is the sole entity type"

echo
echo "PASS: lazydemo generated, builds, and is residue-free."
