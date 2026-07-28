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
	"strings"
	"testing"

	"github.com/maybewaityou/lazytui/internal/core/domain"
)

// TestRefreshStatusMessage verifies the post-refresh footer toast: a non-empty
// list reports the count, while the empty state gets a dedicated message instead
// of the awkward "Refreshed 0 items".
func TestRefreshStatusMessage(t *testing.T) {
	// Empty state: dedicated wording, never "Refreshed 0 items".
	empty := refreshStatusMessage(0)
	if !strings.Contains(empty, "No items to refresh") {
		t.Errorf("refreshStatusMessage(0) = %q, want it to mention 'No items to refresh'", empty)
	}
	if strings.Contains(empty, "Refreshed 0") {
		t.Errorf("refreshStatusMessage(0) = %q, must not say 'Refreshed 0 items'", empty)
	}

	// Non-empty: the count is interpolated into the toast.
	for _, tc := range []struct {
		count int
		want  string
	}{
		{1, "Refreshed 1 items"},
		{3, "Refreshed 3 items"},
		{12, "Refreshed 12 items"},
	} {
		got := refreshStatusMessage(tc.count)
		if !strings.Contains(got, tc.want) {
			t.Errorf("refreshStatusMessage(%d) = %q, want it to contain %q", tc.count, got, tc.want)
		}
	}
}

// TestVisibleItemsAppliesSearchQuery pins the unified filter pipeline: the tag
// filter and the name search must compose (tag narrows first, then name). A name
// query alone narrows allCache to the fuzzy matches; a tag + name query narrows
// by tag first, then by name within that set — order preserved by both stages.
func TestVisibleItemsAppliesSearchQuery(t *testing.T) {
	// Query alone: allCache holds 3 items, only "api" fuzzy-matches "api".
	search := NewSearchBar()
	search.SetText("api")
	tt := &TUI{
		allCache: []domain.Item{
			{Name: "api"},
			{Name: "notes"},
			{Name: "legacy"},
		},
		search: search,
	}

	got := tt.visibleItems()
	if len(got) != 1 || got[0].Name != "api" {
		t.Fatalf("visibleItems(query=\"api\") = %v, want [api]", itemNames(got))
	}

	// Tag + name composition: the tag filter narrows to "work" items (dropping
	// "legacy"), then the name query narrows within that set (dropping "notes")
	// — order is preserved by both stages, so the result is deterministic.
	tagged := &TUI{
		allCache: []domain.Item{
			{Name: "api", Tags: []string{"work"}},
			{Name: "api-server", Tags: []string{"work"}},
			{Name: "notes", Tags: []string{"work"}},
			{Name: "legacy", Tags: []string{"personal"}},
		},
		tagFilter: []string{"work"},
		search:    search, // still holds query "api"
	}

	got = tagged.visibleItems()
	if len(got) != 2 || got[0].Name != "api" || got[1].Name != "api-server" {
		t.Fatalf("visibleItems(tag=\"work\", query=\"api\") = %v, want [api api-server]", itemNames(got))
	}
}

// itemNames returns the Name field of each item, in order, for terse test
// assertions and failure messages.
func itemNames(items []domain.Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Name
	}
	return out
}

// TestFocusQueriesUseWidgetHasFocus locks the contract that the pane-focus
// queries (listHasFocus / detailsHasFocus / searchBarHasFocus) delegate to the
// widget's own HasFocus() rather than comparing the *tview.Application's
// focused-primitive pointer.
//
// Pointer comparison is fragile across tview's focus layers: a keyboard
// SetFocus lands on the wrapper struct, a mouse click lands on the embedded
// tview widget, and InputField further delegates mouse focus to an internal,
// unexported *TextArea. HasFocus recurses through every layer uniformly and
// needs no live *tview.Application, so the query works headless; a pointer-based
// implementation would dereference t.app (nil here) and panic — exactly the
// coupling this test guards against.
func TestFocusQueriesUseWidgetHasFocus(t *testing.T) {
	tt := &TUI{
		itemList: NewItemList(),
		details:  NewItemDetails(),
		search:   NewSearchBar(),
		// app intentionally nil: a focus query must not depend on it.
	}

	if tt.listHasFocus() || tt.detailsHasFocus() || tt.searchBarHasFocus() {
		t.Fatal("no pane should report focus before any Focus() call")
	}

	cases := []struct {
		name  string
		focus func()
		blur  func()
		check func() bool
	}{
		{"list", func() { tt.itemList.Focus(nil) }, func() { tt.itemList.Blur() }, tt.listHasFocus},
		{"details", func() { tt.details.Focus(nil) }, func() { tt.details.Blur() }, tt.detailsHasFocus},
		{"search", func() { tt.search.Focus(nil) }, func() { tt.search.Blur() }, tt.searchBarHasFocus},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.focus()
			if !c.check() {
				t.Errorf("want %s focused after Focus()", c.name)
			}
			c.blur()
			if c.check() {
				t.Errorf("want %s unfocused after Blur()", c.name)
			}
		})
	}
}

// TestSearchInputClearsDetailsWhenNoMatch reproduces the bug where typing a search
// query that matches nothing left the details pane pinned to the previously
// shown item instead of clearing to the empty state.
//
// Root cause: handleSearchInput re-rendered the list via SetItems but never
// re-synced the details pane, and tview's List.Clear()+SetCurrentItem(0) never
// fires SetChangedFunc — so OnSelect was never invoked and the details pane kept
// showing whatever was selected before the filter changed. With no matches the
// list empties and the pane would keep showing the last match indefinitely. This
// is the Task 10 workaround: syncDetails is called explicitly after every
// SetItems.
func TestSearchInputClearsDetailsWhenNoMatch(t *testing.T) {
	details := NewItemDetails()
	list := NewItemList()
	search := NewSearchBar()
	tt := &TUI{
		details:  details,
		itemList: list,
		search:   search,
		allCache: []domain.Item{
			{Name: "alpha"},
			{Name: "beta"},
		},
	}

	// Prime: the list holds both items and the details pane shows the first
	// one, exactly as it would after startup with the cursor resting on item 0.
	list.SetItems(tt.visibleItems())
	tt.details.Render(domain.Item{Name: "alpha"})
	if got := details.GetText(true); !strings.Contains(got, "alpha") {
		t.Fatalf("precondition: details should show alpha, got %q", got)
	}

	// Type a query that matches nothing. The list empties; the details pane
	// must clear to the "No items" placeholder instead of leaving alpha on
	// screen.
	search.SetText("zzz-no-match")
	tt.handleSearchInput("zzz-no-match")

	if got := details.GetText(true); strings.Contains(got, "alpha") {
		t.Errorf("details should be cleared when the search matches nothing, still shows alpha: %q", got)
	}
	if !strings.Contains(details.GetText(true), "No items") {
		t.Errorf("details should show the empty placeholder when the search matches nothing, got %q", details.GetText(true))
	}
}

// TestSearchInputUpdatesDetailsToFirstMatch locks the other half of the Task 10
// workaround: when the search still matches something, the details pane must
// follow the list onto the (new) first match rather than staying on the
// pre-search selection.
//
// Unlike lazytmux, there is no async window load here — OnSelect renders
// synchronously, and syncDetails renders synchronously — so the assertion needs
// no goroutine synchronization.
func TestSearchInputUpdatesDetailsToFirstMatch(t *testing.T) {
	details := NewItemDetails()
	list := NewItemList()
	search := NewSearchBar()
	tt := &TUI{
		details:  details,
		itemList: list,
		search:   search,
		allCache: []domain.Item{
			{Name: "alpha"},
			{Name: "beta"},
		},
	}

	// Prime: details shows alpha (the item at index 0 before the search).
	list.SetItems(tt.visibleItems())
	tt.details.Render(domain.Item{Name: "alpha"})

	// Narrow the query so "beta" becomes the only — and therefore first — match.
	search.SetText("beta")
	tt.handleSearchInput("beta")

	got := details.GetText(true)
	if !strings.Contains(got, "beta") {
		t.Errorf("details should follow the list onto the first match 'beta', got %q", got)
	}
	if strings.Contains(got, "alpha") {
		t.Errorf("details should no longer show the pre-search selection 'alpha', got %q", got)
	}
}

// TestSyncDetailsLooksUpFromAllCache pins that syncDetails finds the current
// item by name from allCache (the full cache, not just the filtered list) and
// renders its full state — Tags/Note/Pinned — not just the name. This is the
// load-bearing assumption behind actOnSelected too.
func TestSyncDetailsLooksUpFromAllCache(t *testing.T) {
	details := NewItemDetails()
	list := NewItemList()
	tt := &TUI{
		details:  details,
		itemList: list,
		allCache: []domain.Item{
			{Name: "alpha", Tags: []string{"work"}, Note: "a note", Pinned: true},
			{Name: "beta"},
		},
	}
	list.SetItems([]domain.Item{{Name: "alpha"}, {Name: "beta"}})
	list.SelectByName("alpha")

	tt.syncDetails()

	got := details.GetText(true)
	if !strings.Contains(got, "alpha") {
		t.Errorf("syncDetails should render the current item's name, got %q", got)
	}
	if !strings.Contains(got, "work") {
		t.Errorf("syncDetails should render the current item's tags, got %q", got)
	}
	if !strings.Contains(got, "a note") {
		t.Errorf("syncDetails should render the current item's note, got %q", got)
	}
}

// TestActOnSelectedNoopOnEmpty verifies actOnSelected is a no-op when the list is
// empty (CurrentName returns ""), so none of the CRUD handlers that route
// through it crash or fire on an empty list.
func TestActOnSelectedNoopOnEmpty(t *testing.T) {
	list := NewItemList()
	tt := &TUI{
		itemList: list,
		allCache: nil,
	}
	called := false
	tt.actOnSelected(func(domain.Item) { called = true })
	if called {
		t.Error("actOnSelected must not fire on an empty list")
	}
}

// TestHeaderTextOmitsUnknownVersion verifies the header degrades gracefully when
// version/commit are empty or "unknown": the brand always shows, the chips do
// not. This keeps a dev build from flashing a "vunknown" badge.
func TestHeaderTextOmitsUnknownVersion(t *testing.T) {
	cases := []struct {
		name    string
		version string
		commit  string
	}{
		{"empty", "", ""},
		{"unknown", "unknown", "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tt := &TUI{version: c.version, commit: c.commit}
			got := tt.headerText()
			if !strings.Contains(got, "lazy") || !strings.Contains(got, "tui") {
				t.Errorf("header must always show the brand, got %q", got)
			}
			if strings.Contains(got, "unknown") {
				t.Errorf("header must not show 'unknown' values, got %q", got)
			}
		})
	}

	// Known values do surface.
	tt := &TUI{version: "v1.2.3", commit: "abcdef1234"}
	got := tt.headerText()
	if !strings.Contains(got, "v1.2.3") {
		t.Errorf("header should show version, got %q", got)
	}
	if !strings.Contains(got, "abcdef1") {
		t.Errorf("header should show short commit (7 chars), got %q", got)
	}
}
