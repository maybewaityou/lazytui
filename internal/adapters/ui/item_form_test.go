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
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/maybewaityou/lazytui/internal/core/domain"
	"github.com/rivo/tview"
)

// TestItemFormStructure pins the shared form's shape: three items
// (Name / Tags / Note) plus no buttons, all starting empty when the initial
// Item is the zero value (New flow). This follows the lazytmux
// session_multifield_form_test.go style of asserting only constructable state
// (no event simulation); the create/edit orchestration is covered by the
// service layer tests.
func TestItemFormStructure(t *testing.T) {
	f := newItemForm("New item", domain.Item{}, nil, nil)
	if got := f.Form().GetFormItemCount(); got != 3 {
		t.Fatalf("form must have 3 items (name/tags/note), got %d", got)
	}
	if got := f.Form().GetButtonCount(); got != 0 {
		t.Fatalf("form must have no buttons (Enter saves, Esc cancels), got %d", got)
	}
	for _, i := range []int{mfFieldName, mfFieldTags, mfFieldNote} {
		if got := f.fieldText(i); got != "" {
			t.Errorf("item %d should start empty for a zero initial Item, got %q", i, got)
		}
	}
}

// TestItemFormInitialValues verifies the Edit flow prefills all three fields
// from the initial Item (Tags joined from []string, Note including a
// multi-line value), while empty fields stay empty.
func TestItemFormInitialValues(t *testing.T) {
	initial := domain.Item{
		Name:    "api",
		Tags:    []string{"work", "prod"},
		Note:    "line one\nline two",
		Pinned:  true,
		Created: time.Unix(1_700_000_000, 0),
	}
	f := newItemForm("Edit item", initial, nil, nil)
	if got := f.fieldText(mfFieldName); got != "api" {
		t.Errorf("name prefill: got %q, want api", got)
	}
	if got := f.fieldText(mfFieldTags); got != "work, prod" {
		t.Errorf("tags prefill: got %q, want %q", got, "work, prod")
	}
	if got := f.fieldText(mfFieldNote); got != "line one\nline two" {
		t.Errorf("note prefill (multi-line): got %q, want %q", got, "line one\nline two")
	}
}

// TestItemFormSubmitMergesPinnedAndCreated is the lazytui-specific behavioral
// test: submit must build a domain.Item that merges the three edited fields
// with the initial Item's Pinned and Created, so an Edit preserves them. A
// brand-new item (zero initial) submits Pinned=false / zero Created, which the
// store overwrites on Create. The Tags field is parsed from the
// comma-separated edit string back into []string.
func TestItemFormSubmitMergesPinnedAndCreated(t *testing.T) {
	created := time.Unix(1_700_000_000, 0)
	initial := domain.Item{
		Name:    "old",
		Tags:    []string{"x"},
		Note:    "old note",
		Pinned:  true,
		Created: created,
	}
	var got domain.Item
	f := newItemForm("Edit item", initial, func(it domain.Item) { got = it }, nil)
	// Simulate typing into the fields, including a fullwidth Chinese comma and
	// surrounding whitespace to exercise parseTags.
	if field, ok := f.Form().GetFormItem(mfFieldName).(*tview.InputField); ok {
		field.SetText("new name")
	}
	if field, ok := f.Form().GetFormItem(mfFieldTags).(*tview.InputField); ok {
		field.SetText("alpha, beta，gamma")
	}
	if area, ok := f.Form().GetFormItem(mfFieldNote).(*tview.TextArea); ok {
		area.SetText("fresh note", true)
	}
	f.submit()
	if got.Name != "new name" {
		t.Errorf("submit Name: got %q, want %q", got.Name, "new name")
	}
	wantTags := []string{"alpha", "beta", "gamma"}
	if strings.Join(got.Tags, ",") != strings.Join(wantTags, ",") {
		t.Errorf("submit Tags: got %v, want %v", got.Tags, wantTags)
	}
	if got.Note != "fresh note" {
		t.Errorf("submit Note: got %q, want %q", got.Note, "fresh note")
	}
	if !got.Pinned {
		t.Errorf("submit must preserve initial.Pinned = true, got false")
	}
	if !got.Created.Equal(created) {
		t.Errorf("submit must preserve initial.Created = %v, got %v", created, got.Created)
	}
}

// TestItemFormSubmitEmptyNameIsIgnored verifies an accidental Save on an empty
// Name does not fire onSubmit (the form stays open for a retry).
func TestItemFormSubmitEmptyNameIsIgnored(t *testing.T) {
	called := false
	f := newItemForm("New item", domain.Item{}, func(domain.Item) { called = true }, nil)
	f.submit()
	if called {
		t.Fatal("submit with empty Name must not fire onSubmit")
	}
}

// TestItemFormNoteFocusKeepsNameVisible reproduces the bug where Tabbing into
// the Note text area scrolled the Name field up and out of view. Root cause:
// tview.NewForm applies a default border padding of 1 on every side, which
// made the form's inner area one row too short for the Note, so focusing it
// triggered tview.Form's scroll-to-focused-item. It draws the form at the size
// the modal allocates, focuses the Note, and asserts the Name label is still
// on screen.
func TestItemFormNoteFocusKeepsNameVisible(t *testing.T) {
	f := newItemForm("Edit item", domain.Item{}, nil, nil)
	form := f.Form()
	// Size the form as the modal does: full width, and the height the column
	// allocates after reserving the hint row.
	form.SetRect(0, 0, itemModalWidth, modalColumnHeight-1)
	// Focus the Note like Tabbing into it would. Focus flips the item's
	// hasFocus flag, which Form.Draw reads to decide whether to scroll the
	// focused item into view.
	form.GetFormItem(mfFieldNote).(*tview.TextArea).Focus(func(tview.Primitive) {})

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("screen init: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 40)
	form.Draw(screen)

	if !screenContains(screen, "Name") {
		t.Errorf("the Name field scrolled out of view when the Note was focused; " +
			"the form's inner area is too short for all three items")
	}
}

// screenContains reports whether needle appears anywhere on the simulation
// screen (one row's worth of cells is searched at a time).
func screenContains(screen tcell.SimulationScreen, needle string) bool {
	w, h := screen.Size()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, _, _, _ := screen.GetContent(x, y)
			b.WriteRune(r)
		}
		b.WriteByte('\n')
	}
	return strings.Contains(b.String(), needle)
}

// TestItemFormPlaceholders pins that the three empty fields each show a muted
// placeholder, mirroring the single-field editors so a field reads identically
// whether it is edited here or standalone (Tags = "comma-separated tags",
// Note = "freeform note"). Like TestItemFormNoteFocusKeepsNameVisible it
// renders the form to a simulation screen and checks each hint is on screen —
// tview paints a placeholder only while the field is empty, so this also
// guards against a regression that sets text instead of a placeholder.
func TestItemFormPlaceholders(t *testing.T) {
	f := newItemForm("New item", domain.Item{}, nil, nil)
	form := f.Form()
	form.SetRect(0, 0, itemModalWidth, modalColumnHeight-1)

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("screen init: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 40)
	form.Draw(screen)

	for _, want := range []string{"item name", "comma-separated tags", "freeform note"} {
		if !screenContains(screen, want) {
			t.Errorf("empty-field placeholder %q is not rendered", want)
		}
	}
}

// focusFormItem sets hasFocus on the i-th item so GetFocusedItemIndex reports
// it. This mirrors what tview.Form.Draw does to the focused item at render
// time; SetFocus alone does not focus the item when its index is 0 and nothing
// else is focused yet (a tview quirk that only surfaces without a Draw).
func focusFormItem(f *ItemForm, i int) {
	switch v := f.Form().GetFormItem(i).(type) {
	case *tview.InputField:
		v.Focus(func(tview.Primitive) {})
	case *tview.TextArea:
		v.Focus(func(tview.Primitive) {})
	}
}

// TestItemFormMoveField verifies ↑/↓ switch fields only on the single-line
// Name/Tags inputs: down moves forward (Name→Tags→Note), up moves backward,
// the Note TextArea is left alone (its ↑/↓ stay on the cursor), and ↑ on Name
// is a boundary no-op. Each case focuses the starting field, calls moveField,
// and checks both the "consumed" return and where focus landed.
func TestItemFormMoveField(t *testing.T) {
	for _, tc := range []struct {
		name   string
		from   int
		down   bool
		wantOK bool
		want   int // expected focused item afterwards
	}{
		{"Name down to Tags", mfFieldName, true, true, mfFieldTags},
		{"Tags down to Note", mfFieldTags, true, true, mfFieldNote},
		{"Tags up to Name", mfFieldTags, false, true, mfFieldName},
		{"Name up is a no-op (boundary)", mfFieldName, false, false, mfFieldName},
		{"Note down stays (cursor move)", mfFieldNote, true, false, mfFieldNote},
		{"Note up stays (cursor move)", mfFieldNote, false, false, mfFieldNote},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newItemForm("Edit item", domain.Item{}, nil, nil)
			focusFormItem(f, tc.from)
			got := f.moveField(tc.down)
			if got != tc.wantOK {
				t.Fatalf("moveField(down=%v) from item %d = %v, want %v", tc.down, tc.from, got, tc.wantOK)
			}
			item, _ := f.Form().GetFocusedItemIndex()
			if item != tc.want {
				t.Errorf("focused item after move = %d, want %d", item, tc.want)
			}
		})
	}
}

// TestNewItemFormTitleInfersNewVsEdit verifies the title inference exercised
// by the public NewItemForm path (via newItemFormFor / itemFormTitle): a
// missing or whitespace-only initial.Name yields "New item", anything else
// yields "Edit item". Reading the title off the bordered *tview.Form requires
// reaching the *ItemForm, so this goes through newItemFormFor (NewItemForm
// itself returns an opaque tview.Primitive wrapped by centerModal). The rule
// is also pinned directly against itemFormTitle to lock the inference without
// a rendered form.
func TestNewItemFormTitleInfersNewVsEdit(t *testing.T) {
	for _, tc := range []struct {
		name string
		init domain.Item
		want string // bordered form title, including the surrounding spaces
	}{
		{"zero initial is New", domain.Item{}, " New item "},
		{"whitespace-only name is New", domain.Item{Name: "   "}, " New item "},
		{"named initial is Edit", domain.Item{Name: "api"}, " Edit item "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := itemFormTitle(tc.init); !strings.Contains(tc.want, got) {
				t.Fatalf("itemFormTitle(%+v): got %q, want contained in %q", tc.init, got, tc.want)
			}
			f := newItemFormFor(tc.init, nil, nil)
			if got := f.form.GetTitle(); got != tc.want {
				t.Fatalf("title for initial=%+v: got %q, want %q", tc.init, got, tc.want)
			}
		})
	}
}

// TestNewItemFormReturnsCenteredPrimitive is a smoke test that the public
// NewItemForm entry point returns a non-nil, centered Primitive ready to
// mount. The full New/Edit orchestration is exercised in the handlers/tui
// layer (Task 14); here we only assert the constructor does not panic and
// yields a mountable widget for both the New and Edit flows.
func TestNewItemFormReturnsCenteredPrimitive(t *testing.T) {
	for _, init := range []domain.Item{
		{},                          // New
		{Name: "api", Pinned: true}, // Edit
	} {
		p := NewItemForm(init, func(domain.Item) {}, func() {})
		if p == nil {
			t.Fatalf("NewItemForm returned nil for initial=%+v", init)
		}
	}
}
