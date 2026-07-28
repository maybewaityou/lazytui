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
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maybewaityou/lazytui/internal/core/domain"
)

// ItemList wraps tview.List and holds the current items for index lookup.
type ItemList struct {
	*tview.List
	items       []domain.Item
	sortLabel   string // current sort mode label, e.g. "Name ↑"
	filterLabel string // current filter label, "" when no filter
	onSelect    func(domain.Item)
	onNavigate  func()
}

func NewItemList() *ItemList {
	il := &ItemList{List: tview.NewList()}
	il.build()
	return il
}

func (il *ItemList) build() {
	il.List.ShowSecondaryText(false)
	il.List.SetBorder(true).
		SetTitle(" Items ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.GetColor(colorBorder)).
		SetTitleColor(tcell.GetColor(colorTitle))
	il.List.
		SetSelectedBackgroundColor(tcell.GetColor(colorSelected)).
		SetSelectedTextColor(tcell.GetColor(colorPrimary)).
		SetHighlightFullLine(true)

	il.List.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(il.items) && il.onSelect != nil {
			il.onSelect(il.items[index])
		}
	})

	il.List.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		// Right is reserved for List → Details focus and is handled by the
		// global key handler, so it is intentionally NOT swallowed here; the
		// remaining keys return focus to the search bar.
		case tcell.KeyLeft, tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyESC:
			if il.onNavigate != nil {
				il.onNavigate()
			}
			return nil
		}
		return e
	})
}

// SetItems rebuilds the list rows from items, resetting the cursor to the top.
// Callers that want to preserve the user's selection across a refresh should
// read CurrentName before SetItems and SelectByName afterwards.
func (il *ItemList) SetItems(items []domain.Item) {
	il.items = items
	il.List.Clear()
	for i := range items {
		il.List.AddItem(formatItemLine(items[i]), "", 0, nil)
	}
	if il.List.GetItemCount() > 0 {
		il.List.SetCurrentItem(0)
	}
}

// Clear drops every row and the backing item slice.
func (il *ItemList) Clear() {
	il.items = nil
	il.List.Clear()
}

// CurrentName returns the name of the item under the cursor, or "" when the
// list is empty. This is the read side of the save/restore-selection pair;
// SelectByName is the write side.
func (il *ItemList) CurrentName() string {
	idx := il.List.GetCurrentItem()
	if idx >= 0 && idx < len(il.items) {
		return il.items[idx].Name
	}
	return ""
}

// SelectByName moves the cursor to the first item with the given name, if
// any. SetItems always resets the cursor to the first item, so after a
// refresh we call this to keep the user's current selection rather than snapping
// back to the top of the list. If the name is gone — e.g. the item was deleted
// out-of-band — the cursor is left wherever SetItems last placed it.
func (il *ItemList) SelectByName(name string) {
	for i, it := range il.items {
		if it.Name == name {
			il.List.SetCurrentItem(i)
			return
		}
	}
}

func (il *ItemList) OnSelect(fn func(domain.Item)) *ItemList {
	il.onSelect = fn
	return il
}

func (il *ItemList) OnNavigate(fn func()) *ItemList {
	il.onNavigate = fn
	return il
}

// SetSortTitle records the sort label and refreshes the composed border title.
func (il *ItemList) SetSortTitle(mode string) {
	il.sortLabel = mode
	il.refreshTitle()
}

// SetFilter records the active filter label ("" = no filter) and refreshes the
// composed border title. The filter is shown alongside the sort mode so both
// pieces of view state share one surface — mirroring how sort is already shown.
func (il *ItemList) SetFilter(filter string) {
	il.filterLabel = filter
	il.refreshTitle()
}

// refreshTitle composes the list border title: always "Sort: <mode>", plus
// "— Filter: <tags>" when a filter is active.
func (il *ItemList) refreshTitle() {
	title := " Items — Sort: " + il.sortLabel + " "
	if il.filterLabel != "" {
		title = " Items — Sort: " + il.sortLabel + " — Filter: " + il.filterLabel + " "
	}
	il.List.SetTitle(title)
}

// formatItemLine renders one list row: the pinned marker (fixed 2 visible
// cells so pinned/unpinned rows stay aligned), the item name, and the tag
// chips. Tag chips reuse renderTagChips (render.go) so chip styling stays
// single-sourced; Pinned is the only per-row marker this list renders.
func formatItemLine(it domain.Item) string {
	// pin column: fixed 2 visible cells so pinned/unpinned rows stay aligned.
	pin := "  "
	if it.Pinned {
		pin = "[" + colorGreen + "]📌[-]"
	}
	name := "[" + colorPrimary + "::b]" + it.Name + "[-]"
	line := fmt.Sprintf("%s %s", pin, name)
	if chips := renderTagChips(it.Tags); chips != "" {
		line += "  " + chips
	}
	return line
}
