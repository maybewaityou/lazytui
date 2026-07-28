#!/bin/bash
#
# Release orchestrator for lazytui.
# Bumps the makefile VERSION, commits, tags, and pushes. Pushing the tag
# triggers .github/workflows/release.yml (goreleaser cross-compile +
# GitHub Release + Homebrew formula push).
#
# Usage:
#   ./scripts/deploy.sh patch|minor|major [--dry-run] [--yes] [--no-test]
#
# Part of the lazytui template; scaffolded tools inherit the same release flow.

set -euo pipefail

# --- config ---------------------------------------------------------------
MAKEFILE="makefile"
REMOTE="origin"
BRANCH="main"

# --- colors ---------------------------------------------------------------
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

# --- output helpers -------------------------------------------------------
info() { printf "\n${BLUE}▸ %s${NC}\n" "$*"; }
ok()   { printf "${GREEN}✓ %s${NC}\n" "$*"; }
warn() { printf "${YELLOW}⚠ %s${NC}\n" "$*"; }
die()  { printf "${RED}✗ %s${NC}\n" "$*" >&2; exit 1; }

# --- version helpers ------------------------------------------------------
# Echo the X.Y.Z numeric core, stripping any v prefix and trailing suffix.
#   v0.1.5       -> 0.1.5
#   0.1.5        -> 0.1.5
#   v0.1.5-beta  -> 0.1.5
# `|| true` so a no-match returns "" instead of aborting under pipefail -e.
normalize_version() {
  echo "$1" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1 || true
}

# Echo the most recent v* git tag (version numbers).
latest_tag() {
  git tag --sort=-v:refname 2>/dev/null | grep -E '^v[0-9]' | head -n1 || true
}

# Echo the next version (with v prefix) for the given core + bump level.
#   compute_next 0.1.5 patch -> v0.1.6
#   compute_next 0.1.5 minor -> v0.2.0
#   compute_next 0.1.5 major -> v1.0.0
compute_next() {
  local core="$1" level="$2"
  local major minor patch
  major=$(echo "$core" | cut -d. -f1)
  minor=$(echo "$core" | cut -d. -f2)
  patch=$(echo "$core" | cut -d. -f3)
  case "$level" in
    major) major=$((major + 1)); minor=0; patch=0 ;;
    minor) minor=$((minor + 1)); patch=0 ;;
    patch) patch=$((patch + 1)) ;;
    *) die "internal error: unknown bump level '$level'" ;;
  esac
  printf 'v%s.%s.%s\n' "$major" "$minor" "$patch"
}

# --- preflight ------------------------------------------------------------
check_clean_tree() {
  # -uno: ignore untracked files. Untracked files (e.g. tooling scratch like
  # .serena/, or macOS .DS_Store) never enter the release — `git commit -am`
  # stages only tracked modifications — so only uncommitted tracked changes
  # block a release.
  [ -z "$(git status --porcelain -uno)" ]
}

check_main_branch() {
  [ "$(git rev-parse --abbrev-ref HEAD)" = "$BRANCH" ]
}

# Local BRANCH must equal REMOTE/BRANCH after a fetch (no ahead/behind).
# `|| return 1` so a failed fetch (offline/flaky) fails the check instead of
# comparing against a stale origin ref (the `check_synced || die` caller
# disables set -e inside this body).
check_synced() {
  git fetch --quiet "$REMOTE" "$BRANCH" || return 1
  [ "$(git rev-parse HEAD)" = "$(git rev-parse "$REMOTE/$BRANCH")" ]
}

# Baseline invariant: makefile numeric core must equal the latest tag's
# numeric core (previous release was fully closed out).
check_version_invariant() {
  local mf_value mf_core tag_core
  mf_value=$(grep -E "^VERSION[[:space:]]*\?=" "$MAKEFILE" | head -n1 \
             | sed -E "s/^VERSION[[:space:]]*\?=//; s/[[:space:]]//g")
  mf_core=$(normalize_version "$mf_value")
  tag_core=$(normalize_version "$(latest_tag)")
  [ -n "$mf_core" ] && [ -n "$tag_core" ] && [ "$mf_core" = "$tag_core" ]
}

preflight() {
  info "Preflight checks"
  check_clean_tree        || die "working tree is dirty — commit or stash first"
  ok "working tree clean"
  check_main_branch       || die "not on '$BRANCH' branch"
  ok "on '$BRANCH' branch"
  local display_core; display_core=$(normalize_version "$(latest_tag)")
  check_version_invariant || die "makefile VERSION does not match latest git tag — close out the previous release first"
  ok "makefile version matches latest tag ($display_core)"
  check_synced           || die "local '$BRANCH' is out of sync with $REMOTE/$BRANCH"
  ok "in sync with $REMOTE/$BRANCH"
}

# --- confirm --------------------------------------------------------------
# confirm <assume_yes> : 0 = proceed, 1 = abort.
confirm() {
  if [ "${1:-0}" = "1" ]; then return 0; fi
  printf "Continue? [y/N] "
  local reply
  read -r reply
  case "$reply" in
    y|Y|yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}

# --- mutation -------------------------------------------------------------
# Rewrite the makefile VERSION line. BSD sed (macOS) needs a backup suffix
# with -i; remove it immediately.
bump_makefile() {
  local newver="$1"
  sed -i.bak -E "s/^(VERSION[[:space:]]*\?=).*/\1 ${newver}/" "$MAKEFILE"
  rm -f "${MAKEFILE}.bak"
}

# owner/repo parsed from the origin remote URL.
repo_slug() {
  git remote get-url "$REMOTE" \
    | sed -E "s#.*github.com[:/]##; s#\.git\$##"
}

# The full mutation sequence. newver already validated + confirmed upstream.
release() {
  local newver="$1"
  info "Applying release $newver"
  bump_makefile "$newver"
  git commit -am "chore: bump version to ${newver}" \
    || die "commit failed — restore with: git checkout -- $MAKEFILE"
  git tag -a "$newver" -m "Release ${newver}" \
    || die "tag failed (exists?) — commit landed locally; finish: git tag -a '$newver' -m 'Release $newver' && git push origin '$newver'  OR undo: git reset --hard HEAD~1"
  git push "$REMOTE" "$BRANCH" \
    || die "push of $BRANCH failed — commit+tag local only; retry: git push origin $BRANCH && git push origin $newver  OR undo: git reset --hard HEAD~1 && git tag -d $newver"
  git push "$REMOTE" "$newver" \
    || die "tag push failed — commit is on remote $BRANCH but release NOT triggered; finish manually: git push origin $newver"

  local slug
  slug=$(repo_slug)
  printf "\n${GREEN}✨ Released %s${NC}\n" "$newver"
  printf "Actions:  https://github.com/%s/actions\n" "$slug"
  printf "Release:  https://github.com/%s/releases/tag/%s\n" "$slug" "$newver"
}

# --- usage ----------------------------------------------------------------
usage() {
  cat <<EOF
Usage: $(basename "$0") <patch|minor|major> [options]

Bump the makefile version, commit, tag, and push. Pushing the tag
triggers .github/workflows/release.yml (goreleaser cross-compile +
GitHub Release + Homebrew formula).

Options:
  --dry-run   Print the plan and exit; no mutation, no push.
  --yes       Skip the confirmation prompt.
  --no-test   Skip 'make test'.
  -h, --help  Show this help.
EOF
}

# --- main -----------------------------------------------------------------
main() {
  local level="" dry_run=0 assume_yes=0 skip_test=0
  while [ $# -gt 0 ]; do
    case "$1" in
      patch|minor|major) level="$1" ;;
      --dry-run) dry_run=1 ;;
      --yes)     assume_yes=1 ;;
      --no-test) skip_test=1 ;;
      -h|--help) usage; exit 0 ;;
      *) die "unknown argument: $1 (see --help)" ;;
    esac
    shift
  done

  [ -n "$level" ] || { usage; exit 1; }

  preflight
  local cur_tag cur_core newver
  cur_tag=$(latest_tag)
  cur_core=$(normalize_version "$cur_tag")
  newver=$(compute_next "$cur_core" "$level")

  if [ "$skip_test" = "0" ]; then
    info "Running tests"
    make test
    ok "tests passed"
  else
    warn "skipping tests (--no-test)"
  fi

  info "Release plan"
  printf "  current:  %s\n" "$cur_tag"
  printf "  next:     %s  (%s bump)\n" "$newver" "$level"
  printf "  → commit + tag + push (triggers release workflow)\n"

  if [ "$dry_run" = "1" ]; then
    info "Dry run — no changes made"
    exit 0
  fi

  confirm "$assume_yes" || die "aborted — no changes made"
  release "$newver"
}

# Run only when executed directly, not when sourced (so functions are
# unit-testable). The BASH_VERSION guard short-circuits before BASH_SOURCE
# is touched, so sourcing from a non-bash shell (e.g. zsh) does not trip
# `set -u` with "BASH_SOURCE: parameter unbound".
if [ -n "${BASH_VERSION:-}" ] && [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  main "$@"
fi
