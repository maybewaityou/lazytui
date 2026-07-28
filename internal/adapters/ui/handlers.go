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
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maybewaityou/lazytui/internal/core/domain"
)

// colorRed is the Tokyo Night red, used for error toasts in the status bar. It
// is defined here (not in const.go) to mirror lazytmux, which keeps the error
// color local to the handlers that surface errors.
const colorRed = "#f7768e"

// statusToastTimeout is how long a transient footer message (e.g. "Refreshed 3
// items") stays visible before reverting to the default keybinding hints.
const statusToastTimeout = 3 * time.Second

// handleGlobalKeys is the root key handler, mounted on t.root via
// SetInputCapture. It dispatches the CRUD actions (a/e/d/r/p/n/f) plus the
// navigation keys (/, q, ?, arrows, Enter). When the search bar is focused it
// gets out of the way so typing flows through unchanged.
//
// All tmux-specific actions from lazytmux (EnterSession / Kill / Detach /
// CurrentSession / suspend / clipboard copy) are gone. Enter here means "dive
// into the details pane" (template semantics), not "attach to a session".
func (t *TUI) handleGlobalKeys(e *tcell.EventKey) *tcell.EventKey {
	// When the search bar is focused, let it handle all keys (typing).
	if t.searchBarHasFocus() {
		return e
	}
	switch e.Rune() {
	case '/':
		t.focusSearch()
		return nil
	case 'q':
		t.app.Stop()
		return nil
	case 'r':
		count := t.refresh()
		t.setStatusTemporary(refreshStatusMessage(count))
		return nil
	case 'p':
		t.actOnSelected(func(it domain.Item) {
			if err := t.svc.TogglePin(it.Name); err == nil {
				t.refresh()
			} else {
				t.setStatusTemporary("[" + colorRed + "]Pin failed[-]")
			}
		})
		return nil
	case 'a':
		t.openNewItemForm()
		return nil
	case 'e':
		t.actOnSelected(t.openEditItemForm)
		return nil
	case 'd':
		t.actOnSelected(t.showDeleteConfirmModal)
		return nil
	case 'n':
		t.actOnSelected(t.editNote)
		return nil
	case 'f':
		t.openTagFilter()
		return nil
	case '?':
		t.openHelp()
		return nil
	}
	switch e.Key() {
	case tcell.KeyRight:
		// List → Details: arrow into the right-hand pane. The list's own
		// capture no longer swallows Right, so the key bubbles up here.
		if t.listHasFocus() {
			t.focusDetails()
			return nil
		}
	case tcell.KeyLeft:
		// Details → List: arrow back to the item list.
		if t.detailsHasFocus() {
			t.focusList()
			return nil
		}
	case tcell.KeyEnter:
		// Enter dives into the details pane. This is template semantics —
		// lazytmux's Enter attached to a tmux session; here there is nothing to
		// attach to, so Enter is a focus move.
		t.focusDetails()
		return nil
	}
	// ↑/↓ fall through to the list (its own InputCapture handles cursor move).
	return e
}

// handleSearchInput re-renders the list from the filtered pipeline and re-syncs
// the details pane to the filtered list's first item. The query is read fresh
// from the search bar inside visibleItems, so the tag filter and name search
// always compose through one pipeline. Routed through renderVisibleList so the
// details pane re-syncs — SetItems does not fire OnSelect on a cursor reset.
func (t *TUI) handleSearchInput(_ string) {
	t.renderVisibleList()
}

// --- Focus helpers -----------------------------------------------------------
//
// Centralizing SetFocus on these three helpers keeps every return-to-list path
// (arrow-back from details, Esc from the search bar, closing a form or modal) in
// one place. The *HasFocus queries delegate to each widget's HasFocus() rather
// than comparing app.GetFocus() to a widget pointer: pointer comparison is
// fragile across tview's focus layers (a keyboard SetFocus lands on the wrapper
// struct, a mouse click lands on the embedded tview widget, and InputField
// delegates mouse focus to an unexported inner TextArea), while HasFocus
// recurses through every layer uniformly and works headless (no live
// *tview.Application). This is the lazytmux convention verbatim.

func (t *TUI) focusList()    { t.app.SetFocus(t.itemList) }
func (t *TUI) focusDetails() { t.app.SetFocus(t.details) }
func (t *TUI) focusSearch()  { t.app.SetFocus(t.search) }

// blurSearchBar returns focus to the list without clearing the query. Mirrors
// lazyssh: Esc only blurs the input, so the list (and its cursor) stays intact.
func (t *TUI) blurSearchBar() { t.focusList() }

func (t *TUI) listHasFocus() bool      { return t.itemList.HasFocus() }
func (t *TUI) detailsHasFocus() bool   { return t.details.HasFocus() }
func (t *TUI) searchBarHasFocus() bool { return t.search.HasFocus() }

// --- Refresh / render pipeline ----------------------------------------------

// refresh reloads items from the service, re-renders the list, and returns the
// count so the 'r' key can surface it in the footer toast.
//
// The current selection is preserved by name across the reload: SetItems always
// resets the cursor to the first item, so without SelectByName a refresh would
// snap the user back to the top. syncDetails is called explicitly afterwards —
// SetItems does not fire OnSelect on a cursor reset (the Task 10 workaround), so
// without the manual sync the details pane would keep showing stale data after a
// create/edit/delete/refresh.
func (t *TUI) refresh() int {
	items, err := t.svc.List()
	if err != nil {
		t.log.Warnw("list items failed", "error", err)
		t.status.SetStatus("[" + colorRed + "]error: " + err.Error() + "[-]")
		t.allCache = nil
		t.itemList.Clear()
		t.details.SetText("[" + colorSecondary + "]No items[-]")
		return 0
	}
	prevName := t.itemList.CurrentName()
	t.allCache = items
	t.applySortAndRender()
	if prevName != "" {
		t.itemList.SelectByName(prevName)
	}
	t.syncDetails()
	t.refreshStatusBarHints()
	return len(items)
}

// syncDetails mirrors the details pane to the list's current selection.
//
// tview's List.SetCurrentItem only fires SetChangedFunc when the index actually
// changes, and SetItems Clear()s the list first (which resets the cursor to 0)
// before SetCurrentItem(0) — so programmatic reloads never trigger OnSelect.
// After a create/edit/delete/refresh the right pane would otherwise keep showing
// stale data, so we re-sync explicitly here. (Task 10 workaround.)
func (t *TUI) syncDetails() {
	name := t.itemList.CurrentName()
	if name == "" {
		t.details.SetText("[" + colorSecondary + "]No items[-]")
		return
	}
	for _, it := range t.allCache {
		if it.Name == name {
			t.details.Render(it)
			return
		}
	}
}

// renderVisibleList re-renders the list from the filtered pipeline and re-syncs
// the details pane to the resulting selection. It is the single entry point that
// every "visible set changed" path — search input, sort cycle, tag filter — goes
// through. The sync cannot be left to tview for the same reason as refresh: see
// syncDetails.
func (t *TUI) renderVisibleList() {
	t.itemList.SetItems(t.visibleItems())
	t.syncDetails()
}

// applySortAndRender sorts allCache in place, updates the list title (sort mode
// + active filter), and re-renders the visible list. Sort is kept separate from
// visibleItems' filter so the sort runs once per state change rather than on
// every keystroke — visibleItems is called on every search keystroke and would
// otherwise re-sort an already-sorted cache each time.
func (t *TUI) applySortAndRender() {
	sortItemsForUI(t.allCache, t.sortMode)
	t.itemList.SetSortTitle(t.sortMode.String())
	t.itemList.SetFilter(filterDescription(t.tagFilter))
	t.renderVisibleList()
}

// actOnSelected runs fn against the item currently under the cursor, looking it
// up from allCache so fn receives the full Item (Tags/Note/Pinned/Created), not
// just the name. No-op when the list is empty — every CRUD handler that goes
// through actOnSelected is naturally a no-op on an empty list.
func (t *TUI) actOnSelected(fn func(domain.Item)) {
	name := t.itemList.CurrentName()
	if name == "" {
		return
	}
	for _, it := range t.allCache {
		if it.Name == name {
			fn(it)
			return
		}
	}
}

// --- Forms and modals -------------------------------------------------------

// openNewItemForm opens the Name/Tags/Note modal with empty initial values. On
// submit it creates the item, then best-effort saves tags and note — the
// service's Create only takes a name, so metadata is a separate write (mirroring
// lazytmux's CreateSession + SaveTags/SaveNote). A failed Create surfaces a
// toast and closes the form.
func (t *TUI) openNewItemForm() {
	form := newItemFormFor(domain.Item{}, func(it domain.Item) {
		if err := t.svc.Create(it.Name); err != nil {
			t.setStatusTemporary("[" + colorRed + "]Create failed[-]")
			t.closeForm()
			return
		}
		if len(it.Tags) > 0 {
			_ = t.svc.SaveTags(it.Name, it.Tags)
		}
		if note := strings.TrimSpace(it.Note); note != "" {
			_ = t.svc.SaveNote(it.Name, note)
		}
		t.refresh()
		t.closeForm()
	}, t.closeForm)
	t.app.SetRoot(form.Primitive(), true)
	t.app.SetFocus(form.Form())
}

// openEditItemForm opens the same form prefilled with the current item. Rename
// fires only when the name actually changed — a no-op rename (same name) can
// error, and the form guarantees a non-empty Name. Update + SaveTags/SaveNote
// then land at the (possibly new) key. A failed Rename surfaces a toast and
// closes the form without attempting the metadata writes.
func (t *TUI) openEditItemForm(current domain.Item) {
	form := newItemFormFor(current, func(it domain.Item) {
		if it.Name != current.Name {
			if err := t.svc.Rename(current.Name, it.Name); err != nil {
				t.setStatusTemporary("[" + colorRed + "]Rename failed[-]")
				t.closeForm()
				return
			}
		}
		_ = t.svc.Update(it.Name, it)
		_ = t.svc.SaveTags(it.Name, it.Tags)
		if note := strings.TrimSpace(it.Note); note != "" {
			_ = t.svc.SaveNote(it.Name, note)
		}
		t.refresh()
		t.closeForm()
	}, t.closeForm)
	t.app.SetRoot(form.Primitive(), true)
	t.app.SetFocus(form.Form())
}

// editNote opens the single-field note modal for the current item. An empty
// submit clears the note (SaveNote tolerates empty) — clearing freeform text is
// trivially reversible, so no confirm gate. The NoteForm is the same TextArea
// the New/Edit form's Note field uses, so editing feels identical standalone.
func (t *TUI) editNote(current domain.Item) {
	form := NewNoteForm("Note", "freeform note").
		InitialValue(current.Note).
		OnSubmit(func(input string) {
			note := strings.TrimSpace(input)
			if err := t.svc.SaveNote(current.Name, note); err == nil {
				t.refresh()
			} else {
				t.setStatusTemporary("[" + colorRed + "]Note failed[-]")
			}
			t.closeForm()
		}).
		OnCancel(t.closeForm)
	t.app.SetRoot(form.Primitive(), true)
	t.app.SetFocus(form.Area())
}

// showDeleteConfirmModal asks for confirmation, then deletes. ConfirmModal sets
// up the visuals and wires onYes to the "Confirm" button; we re-wrap
// SetDoneFunc afterwards so every dismissal path (Confirm, Cancel, Esc) also
// restores the main layout — ConfirmModal's own done func only runs onYes and
// would leave the modal mounted on Cancel/Esc.
func (t *TUI) showDeleteConfirmModal(current domain.Item) {
	msg := fmt.Sprintf("Delete item %s?\n\nThis action cannot be undone.", current.Name)
	onYes := func() {
		if err := t.svc.Delete(current.Name); err == nil {
			t.refresh()
		} else {
			t.setStatusTemporary("[" + colorRed + "]Delete failed[-]")
		}
	}
	modal := ConfirmModal("Delete", msg, onYes)
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonLabel == "Confirm" {
			onYes()
		}
		t.closeModal()
	})
	t.app.SetRoot(modal, true)
}

// openTagFilter opens the multi-select tag filter modal. Candidates are the
// sorted union of every loaded item's tags; the active filter is pre-selected.
// Applying writes the selection to tagFilter (in-memory only — a fresh launch
// always starts unfiltered) and re-runs the render pipeline so the list and its
// title update together. When there are no tags at all, a transient hint is
// surfaced instead of an empty modal.
func (t *TUI) openTagFilter() {
	candidates := collectTags(t.allCache)
	if len(candidates) == 0 {
		t.setStatusTemporary("[" + colorYellow + "]No tags yet — add tags in the New/Edit form[-]")
		return
	}
	form := NewTagFilterForm(candidates, t.tagFilter).
		OnApply(func(tags []string) {
			t.tagFilter = tags
			t.closeModal()
			t.applySortAndRender()
		}).
		OnCancel(t.closeModal)
	t.app.SetRoot(form.Primitive(), true)
	t.app.SetFocus(form)
}

// openHelp shows the key-binding reference, derived from the keyBindings single
// source and topped with the current sort + filter status. ?, Esc, or q dismiss
// it. The body is wrapped in the same centered flex as lazytmux's openHelp so
// the two-column layout reads identically across the toolchain.
func (t *TUI) openHelp() {
	help := NewHelpModal(t.sortMode.String(), filterDescription(t.tagFilter))
	help.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch {
		case e.Key() == tcell.KeyESC, e.Rune() == '?', e.Rune() == 'q':
			t.closeModal()
			return nil
		}
		return e
	})
	flex := tview.NewFlex().AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 1, 0, false).
			AddItem(help, 0, 1, true).
			AddItem(nil, 1, 0, false), 64, 0, true).
		AddItem(nil, 0, 1, false)
	t.app.SetRoot(flex, true)
	t.app.SetFocus(help.focus)
}

// closeForm restores the main layout after a form (New/Edit/Note) and re-focuses
// the list. closeModal is the same operation with a name that reads better at
// modal dismissal sites (delete confirm, tag filter, help).
func (t *TUI) closeForm()  { t.app.SetRoot(t.root, true); t.focusList() }
func (t *TUI) closeModal() { t.app.SetRoot(t.root, true); t.focusList() }

// --- Status bar -------------------------------------------------------------

// refreshStatusMessage builds the transient footer toast shown after pressing
// 'r'. A non-empty list reports how many items were refreshed; an empty list
// gets a clearer line instead of the awkward "Refreshed 0 items". Pure function
// so it can be unit-tested directly.
func refreshStatusMessage(count int) string {
	if count == 0 {
		return "[" + colorCyan + "]No items to refresh[-]"
	}
	return fmt.Sprintf("["+colorGreen+"]Refreshed %d items[-]", count)
}

// setStatusTemporary shows a transient footer message, then reverts to the
// default keybinding hints after statusToastTimeout. Any pending toast is
// cancelled first so rapid presses reuse a single timer instead of stacking.
// The reset runs via QueueUpdateDraw because time.AfterFunc fires on its own
// goroutine and tview widgets are not concurrency-safe.
func (t *TUI) setStatusTemporary(msg string) {
	t.status.SetStatus(msg)
	if t.statusTimer != nil {
		t.statusTimer.Stop()
	}
	t.statusTimer = time.AfterFunc(statusToastTimeout, func() {
		t.app.QueueUpdateDraw(t.refreshStatusBarHints)
	})
}

// refreshStatusBarHints restores the footer line appropriate for the current
// list state. After a transient toast ("Refreshed", "Pin failed") the timer
// fires on its own goroutine, so we route through QueueUpdateDraw (in setStatusTemporary)
// and pick the empty-state or full hint set based on whether any item is loaded.
func (t *TUI) refreshStatusBarHints() {
	if len(t.allCache) == 0 {
		t.status.ShowEmpty()
		return
	}
	t.status.ResetHints()
}
