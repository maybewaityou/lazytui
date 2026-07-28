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

// TestSelectByName verifies the cursor lands on the matching item and that a
// missing name leaves the cursor untouched (so a vanished item degrades to
// wherever SetItems last placed it).
func TestSelectByName(t *testing.T) {
	il := NewItemList()
	il.SetItems([]domain.Item{
		{Name: "alpha"},
		{Name: "beta"},
		{Name: "gamma"},
	})

	if got := il.GetCurrentItem(); got != 0 {
		t.Fatalf("initial cursor = %d, want 0", got)
	}

	il.SelectByName("beta")
	if got := il.GetCurrentItem(); got != 1 {
		t.Errorf("after SelectByName(beta) cursor = %d, want 1", got)
	}

	// A missing name must not move the cursor.
	il.SelectByName("does-not-exist")
	if got := il.GetCurrentItem(); got != 1 {
		t.Errorf("after SelectByName(missing) cursor = %d, want 1 (unchanged)", got)
	}

	// CurrentName should reflect the restored cursor.
	if got := il.CurrentName(); got != "beta" {
		t.Errorf("CurrentName = %q, want %q", got, "beta")
	}
}

// TestSelectByNameRestoresAfterReload mirrors the refresh() flow: SetItems
// snaps the cursor back to the first item on every reload, and SelectByName is
// what brings the user's previous selection back.
func TestSelectByNameRestoresAfterReload(t *testing.T) {
	il := NewItemList()
	items := []domain.Item{{Name: "alpha"}, {Name: "beta"}, {Name: "gamma"}}
	il.SetItems(items)
	il.SelectByName("beta")
	if got := il.GetCurrentItem(); got != 1 {
		t.Fatalf("cursor before reload = %d, want 1", got)
	}

	// Reload (as refresh() does) — SetItems resets the cursor to 0.
	il.SetItems(items)
	if got := il.GetCurrentItem(); got != 0 {
		t.Fatalf("cursor after reload = %d, want 0", got)
	}

	// SelectByName restores the previously selected item.
	il.SelectByName("beta")
	if got := il.GetCurrentItem(); got != 1 {
		t.Errorf("cursor after restore = %d, want 1", got)
	}
}

// TestPinnedMarkerRendered verifies that Pinned items carry the 📌 marker and
// that unpinned items do not. Pinned is the only row-level marker this list
// renders.
func TestPinnedMarkerRendered(t *testing.T) {
	il := NewItemList()
	il.SetItems([]domain.Item{
		{Name: "alpha", Pinned: false},
		{Name: "beta", Pinned: true},
	})

	alpha, _ := il.GetItemText(0)
	beta, _ := il.GetItemText(1)

	if strings.Contains(alpha, "📌") {
		t.Errorf("unpinned row must not carry 📌, got: %q", alpha)
	}
	if !strings.Contains(beta, "📌") {
		t.Errorf("pinned row must carry 📌, got: %q", beta)
	}
}

// TestRenderTagChipsOnRow verifies that an item's tags surface on its list row
// via renderTagChips (the same helper the details pane uses), proving the list
// does not redefine chip styling.
func TestRenderTagChipsOnRow(t *testing.T) {
	il := NewItemList()
	il.SetItems([]domain.Item{
		{Name: "alpha", Tags: []string{"work", "urgent"}},
	})
	row, _ := il.GetItemText(0)
	// tagChip renders " <tag> " inside a black-on-accent pill; the bare tag
	// words must appear on the row.
	if !strings.Contains(row, "work") || !strings.Contains(row, "urgent") {
		t.Errorf("row missing tag chips: %q", row)
	}
}

// TestSetFilterAppendsToTitle verifies that an active filter is surfaced in the
// list border title alongside the sort mode, and that clearing it removes the
// Filter segment.
func TestSetFilterAppendsToTitle(t *testing.T) {
	il := NewItemList()
	il.SetSortTitle("Name ↑")
	title := il.GetTitle()
	if !strings.Contains(title, "Sort: Name ↑") {
		t.Fatalf("title missing sort: %q", title)
	}
	if strings.Contains(title, "Filter:") {
		t.Errorf("title should not show Filter when none set: %q", title)
	}

	il.SetFilter("work, personal")
	title = il.GetTitle()
	if !strings.Contains(title, "Filter: work, personal") {
		t.Errorf("title missing filter: %q", title)
	}
	if !strings.Contains(title, "Sort: Name ↑") {
		t.Errorf("title lost sort when filter set: %q", title)
	}

	il.SetFilter("")
	title = il.GetTitle()
	if strings.Contains(title, "Filter:") {
		t.Errorf("title should drop Filter after clear: %q", title)
	}
}

// TestTitleUsesItemsLabel verifies the border title names the list "Items".
func TestTitleUsesItemsLabel(t *testing.T) {
	il := NewItemList()
	il.SetSortTitle("Name ↑")
	if got := il.GetTitle(); !strings.Contains(got, "Items") {
		t.Errorf("title should label the list Items, got: %q", got)
	}
}

// TestClearDropsRows verifies Clear empties both the visible list and the
// backing slice, so CurrentName returns "" after a clear.
func TestClearDropsRows(t *testing.T) {
	il := NewItemList()
	il.SetItems([]domain.Item{{Name: "alpha"}, {Name: "beta"}})
	if il.GetItemCount() != 2 {
		t.Fatalf("item count before clear = %d, want 2", il.GetItemCount())
	}
	il.Clear()
	if il.GetItemCount() != 0 {
		t.Errorf("item count after clear = %d, want 0", il.GetItemCount())
	}
	if got := il.CurrentName(); got != "" {
		t.Errorf("CurrentName after clear = %q, want empty", got)
	}
}
