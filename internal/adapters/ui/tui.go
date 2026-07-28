// Copyright 2026.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"time"

	"github.com/rivo/tview"
	"go.uber.org/zap"

	"github.com/maybewaityou/lazytui/internal/core/domain"
	"github.com/maybewaityou/lazytui/internal/core/ports"
)

// TUI is the runnable lazytui application. It owns the tview Application, the UI
// components built by the earlier tasks, and the in-memory view state (allCache,
// tagFilter, sortMode) that drives the filter pipeline. There is no tmux here —
// every action is a CRUD call against ports.Service, which is what makes this
// tool a generic item manager rather than a session manager.
type TUI struct {
	log *zap.SugaredLogger
	svc ports.Service

	version string
	commit  string

	app *tview.Application

	itemList *ItemList
	details  *ItemDetails
	search   *SearchBar
	status   *StatusBar

	root    *tview.Flex
	content *tview.Flex

	sortMode SortMode
	allCache []domain.Item
	// tagFilter holds the active tag filter (OR semantics). Empty/nil means no
	// filter. Pure in-memory view state — never persisted — so a fresh launch
	// always starts unfiltered.
	tagFilter []string
	// statusTimer backs setStatusTemporary's revert-to-hints; kept on the struct
	// so a new toast cancels a pending one instead of stacking.
	statusTimer *time.Timer
}

// NewTUI constructs the TUI. Sub-components are built here (rather than in Run)
// so white-box tests can reach them via the struct fields without booting a
// tcell screen — the focus queries and the filter pipeline are testable
// headless.
func NewTUI(log *zap.SugaredLogger, svc ports.Service, version, commit string) *TUI {
	t := &TUI{log: log, svc: svc, version: version, commit: commit, sortMode: SortByNameAsc}
	t.buildComponents()
	return t
}

// Run boots the tview application: it builds the layout, wires the global key
// handler, runs the first refresh, and blocks in app.Run until the user quits.
// A panic anywhere in the app loop is logged and swallowed so a corrupt screen
// state never crashes the process without a trace.
func (t *TUI) Run() error {
	defer func() {
		if r := recover(); r != nil {
			t.log.Errorw("panic recovered", "error", r)
		}
	}()
	t.app = initializeTheme()
	t.app.EnableMouse(true)
	t.buildLayout()
	t.root.SetInputCapture(t.handleGlobalKeys)
	t.app.SetRoot(t.root, true)
	t.focusList()
	t.refresh()
	t.log.Infow("starting TUI", "version", t.version, "commit", t.commit)
	if err := t.app.Run(); err != nil {
		t.log.Errorw("application run error", "error", err)
		return err
	}
	return nil
}

// buildComponents constructs the widgets and wires their callbacks.
//
// OnSelect re-renders the details pane on every cursor move (arrow navigation).
// OnSearch re-runs the filter pipeline on every keystroke. OnEscape/OnNavigate
// return focus to the list — both the search bar's Esc and the list's
// Backspace/Left land back on the list, mirroring lazyssh/lazytmux.
func (t *TUI) buildComponents() {
	t.search = NewSearchBar().
		OnSearch(t.handleSearchInput).
		OnEscape(t.blurSearchBar).
		OnNavigate(func() { t.focusList() })
	t.itemList = NewItemList().
		OnSelect(func(it domain.Item) {
			t.details.Render(it)
		}).
		OnNavigate(func() { t.app.SetFocus(t.search) })
	t.details = NewItemDetails()
	t.status = NewStatusBar()
}

// buildLayout mounts the three-region layout the brief specifies: a one-row
// header TextView on top, the horizontal [search+list | details] split in the
// middle (3:2 ratio), and a one-row status bar at the bottom. The left column
// stacks the search bar (fixed 3 rows) over the list (flex); the right column
// is the details pane alone.
func (t *TUI) buildLayout() {
	header := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	header.SetText(t.headerText())

	left := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.search, 3, 0, true).
		AddItem(t.itemList, 0, 1, false)
	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.details, 0, 1, false)

	t.content = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(left, 0, 3, true).
		AddItem(right, 0, 2, false)

	t.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(t.content, 0, 1, true).
		AddItem(t.status, 1, 0, false)
}

// headerText composes the one-row header: the lazytui brand, plus version and
// short-commit chips when they are known. Empty/"unknown" values are omitted so
// a dev build does not flash a "vunknown" badge.
func (t *TUI) headerText() string {
	s := "[" + colorPrimary + "::b]lazy[-][" + colorAccent + "::b]tui[-]"
	if t.version != "" && t.version != "unknown" {
		s += "  [" + colorGreen + "]" + t.version + "[-]"
	}
	c := t.commit
	if len(c) > 7 {
		c = c[:7]
	}
	if c != "" && c != "unknown" {
		s += "  [" + colorPurple + "]" + c + "[-]"
	}
	return s
}

// currentSearchQuery returns the search bar's text, or "" when the search bar is
// not wired (e.g. in unit tests constructing a partial TUI). Reading the query
// fresh here — rather than passing it through handleSearchInput's argument —
// keeps the tag filter and name search composing through one pipeline in
// visibleItems.
func (t *TUI) currentSearchQuery() string {
	if t.search == nil {
		return ""
	}
	return t.search.GetText()
}

// visibleItems is the single filtering entry point: it narrows allCache through
// the active tag filter (OR semantics) and then the fuzzy name query. Sort is
// applied separately by applySortAndRender on every state change that touches
// allCache or sortMode, so this function only re-filters — cheap, and safe to
// call from the search path on every keystroke. The order (tag first, then
// name) means a tag narrows the category before the name search hunts within it.
func (t *TUI) visibleItems() []domain.Item {
	return applyFilters(t.allCache, t.tagFilter, t.currentSearchQuery())
}
