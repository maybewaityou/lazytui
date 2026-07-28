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

	"github.com/gdamore/tcell/v2"
	"github.com/maybewaityou/lazytui/internal/core/domain"
	"github.com/rivo/tview"
)

// Indices of the form's items, in AddXxx order.
const (
	mfFieldName = iota
	mfFieldTags
	mfFieldNote
)

// noteVisibleRows is the Note text area's visible height. The modal is sized so
// Name + Tags + Note all fit at once (no scrolling of the modal itself); a note
// longer than this scrolls only within the Note field.
const noteVisibleRows = 5

// modalColumnHeight is the centered column's height: the form (border + Name +
// Tags + the noteVisibleRows Note area) plus one row for the key hint. With the
// form's vertical border padding cleared (see newItemForm), the form's inner
// area is tall enough that focusing the Note never scrolls the Name field out
// of view.
const modalColumnHeight = 13

// itemModalWidth is the centered column's width, shared with the standalone
// NoteForm so a note reads identically whether edited here or in its own modal.
const itemModalWidth = 62

// ItemForm is a Name / Tags / Note modal shared by New and Edit. Name and Tags
// are single-line inputs; Note is a multi-line text area. Enter submits from
// any field; Shift+Enter inserts a newline inside Note; Esc cancels. Fields
// are navigated with Tab/Shift+Tab everywhere, plus ↑/↓ on the single-line
// Name/Tags inputs (see moveField); the Note TextArea keeps ↑/↓ for cursor
// movement, so leave Note with Shift+Tab. (Shift+Enter requires a terminal
// that reports it distinctly from Enter — one speaking the CSI-u / kitty
// keyboard protocol; on a terminal that doesn't, Shift+Enter behaves like
// plain Enter and submits.)
//
// Only Name / Tags / Note are editable here. Pinned is toggled by the 'p' key
// in the list, and Created is set by the store on Create — neither belongs in
// this form. On submit the form merges the three edited fields with the
// initial Item's Pinned and Created so an Edit preserves them.
type ItemForm struct {
	form     *tview.Form
	hint     *tview.TextView
	initial  domain.Item
	onSubmit func(domain.Item)
	onCancel func()
}

// NewItemForm builds a complete centered modal for creating or editing an Item.
// initial prefills Name / Tags / Note; an empty initial.Name yields the "New
// item" title, otherwise "Edit item". onSubmit fires with a domain.Item whose
// Name/Tags/Note come from the form and whose Pinned/Created are carried over
// from initial (for a brand-new item, initial is the zero Item — Pinned is
// false and Created is the zero time, which the store overwrites on Create).
// The returned Primitive is ready to mount via app.SetRoot(..., true).
func NewItemForm(initial domain.Item, onSubmit func(domain.Item), onCancel func()) tview.Primitive {
	return newItemFormFor(initial, onSubmit, onCancel).Primitive()
}

// itemFormTitle picks the bordered form's title from the initial Item: a
// missing (or whitespace-only) Name means "New item", anything else means
// "Edit item". Extracted so NewItemForm and the test reach agree on the rule
// without the test duplicating the logic.
func itemFormTitle(initial domain.Item) string {
	if strings.TrimSpace(initial.Name) != "" {
		return "Edit item"
	}
	return "New item"
}

// newItemFormFor is the titled-form constructor used by NewItemForm. Split out
// so tests can reach the underlying *tview.Form via Form() (NewItemForm returns
// an opaque tview.Primitive wrapped by centerModal).
func newItemFormFor(initial domain.Item, onSubmit func(domain.Item), onCancel func()) *ItemForm {
	return newItemForm(itemFormTitle(initial), initial, onSubmit, onCancel)
}

// newItemForm builds the form with an explicit title. Tests use it directly to
// pin behavior that does not depend on the New/Edit title inference; the
// title-inferring entry point is newItemFormFor (wrapped by NewItemForm).
func newItemForm(title string, initial domain.Item, onSubmit func(domain.Item), onCancel func()) *ItemForm {
	f := &ItemForm{
		form:     tview.NewForm(),
		hint:     tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter),
		initial:  initial,
		onSubmit: onSubmit,
		onCancel: onCancel,
	}
	f.form.SetBorder(true).
		SetTitle(" "+title+" ").
		SetTitleColor(tcell.GetColor(colorTitle)).
		SetBorderColor(tcell.GetColor(colorBorder)).
		// Clear the vertical border padding. tview.NewForm defaults to a padding
		// of 1 on every side, which steals two rows from the inner area and makes
		// it one row shorter than the Note needs — so focusing the Note makes
		// tview.Form scroll the Name field up and out of view. Horizontal padding
		// is kept so the fields sit a column inside the border.
		SetBorderPadding(0, 0, 1, 1)
	f.form.SetLabelColor(tcell.GetColor(colorAccent)).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(tcell.GetColor(colorPrimary)).
		SetButtonTextColor(tcell.GetColor(colorPrimary)).
		SetButtonBackgroundColor(tcell.ColorDefault).
		SetBackgroundColor(tcell.ColorDefault)
	// fieldWidth 0 = expand to the form's width; the Note area is noteVisibleRows
	// rows tall and unbounded in length (maxLength 0).
	// Labels end with ":" — tview.Form pads one space after the label
	// (maxLabelWidth++ in Form.Draw), so "Name:" renders as "Name: [field]".
	f.form.AddInputField("Name:", "", 0, nil, nil).
		AddInputField("Tags:", "", 0, nil, nil).
		AddTextArea("Note:", "", 0, noteVisibleRows, 0, nil)
	// Placeholders mirror the single-field editors so a field reads identically
	// whether it is edited here or standalone: Tags = "comma-separated tags"
	// (the Tags-only flow), Note = "freeform note" (the Note-only flow).
	// AddInputField/AddTextArea take no placeholder, so set it on the underlying
	// widgets after adding them. The two widget types expose different
	// placeholder-color APIs: InputField takes a single text color, TextArea
	// takes a full style (background + foreground); both use the muted
	// colorSecondary so the hint recedes behind real input (colorPrimary).
	if field, ok := f.form.GetFormItem(mfFieldName).(*tview.InputField); ok {
		field.SetPlaceholder("item name").
			SetPlaceholderTextColor(tcell.GetColor(colorSecondary))
	}
	if field, ok := f.form.GetFormItem(mfFieldTags).(*tview.InputField); ok {
		field.SetPlaceholder("comma-separated tags").
			SetPlaceholderTextColor(tcell.GetColor(colorSecondary))
	}
	if area, ok := f.form.GetFormItem(mfFieldNote).(*tview.TextArea); ok {
		area.SetPlaceholder("freeform note").
			SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorDefault).Foreground(tcell.GetColor(colorSecondary)))
	}
	f.hint.SetText("[" + colorSecondary + "]↑↓(switch field) · Enter(save) · Shift+Enter(newline in Note) · Esc(cancel)[-]")

	// Prefill Name / Tags / Note from initial. Pinned and Created are not
	// surfaced as fields — they are merged back on submit. Tags is stored on
	// the Item as []string but edited here as a comma-joined string, so join
	// for display; an empty initial yields empty fields (New flow).
	if field, ok := f.form.GetFormItem(mfFieldName).(*tview.InputField); ok {
		field.SetText(initial.Name)
	}
	if field, ok := f.form.GetFormItem(mfFieldTags).(*tview.InputField); ok {
		field.SetText(strings.Join(initial.Tags, ", "))
	}
	if area, ok := f.form.GetFormItem(mfFieldNote).(*tview.TextArea); ok {
		area.SetText(initial.Note, true)
	}

	// Enter submits from any field — the Note text area never sees a plain
	// Enter (so no stray newline is inserted), making "type a name and press
	// Enter" create/save immediately. Shift+Enter is passed through to the
	// focused item so Note can insert a newline (tview's TextArea newlines on
	// KeyEnter regardless of modifiers, so only the shifted variant reaches
	// it). Esc cancels. ↑/↓ on the single-line Name/Tags inputs switch fields
	// (moveField); everything else falls through to tview.Form, which forwards
	// unhandled keys to the focused item (Tab/Shift+Tab navigate fields
	// there).
	f.form.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyESC:
			f.cancel()
			return nil
		case tcell.KeyEnter:
			if e.Modifiers()&tcell.ModShift != 0 {
				return e // let Note insert a newline
			}
			f.submit()
			return nil
		case tcell.KeyDown, tcell.KeyUp:
			// Only the single-line inputs (Name/Tags) react: the Note TextArea
			// keeps ↑/↓ for cursor movement, so moveField leaves it alone and
			// returns false, letting the event fall through to tview.Form.
			if f.moveField(e.Key() == tcell.KeyDown) {
				return nil
			}
		}
		return e
	})
	return f
}

func (f *ItemForm) OnSubmit(fn func(domain.Item)) *ItemForm { f.onSubmit = fn; return f }

func (f *ItemForm) OnCancel(fn func()) *ItemForm { f.onCancel = fn; return f }

// submit is the Save handler. An empty Name is ignored (the form stays open)
// so an accidental Save can't act on a nameless item; otherwise it fires
// onSubmit with a domain.Item built from the three form fields plus the
// initial Item's Pinned and Created (preserved across an Edit, zero-valued
// for a brand-New item the store has not yet timestamped).
func (f *ItemForm) submit() {
	name := strings.TrimSpace(f.fieldText(mfFieldName))
	if name == "" {
		return
	}
	if f.onSubmit != nil {
		f.onSubmit(domain.Item{
			Name:    name,
			Tags:    parseTags(f.fieldText(mfFieldTags)),
			Note:    f.fieldText(mfFieldNote),
			Pinned:  f.initial.Pinned,
			Created: f.initial.Created,
		})
	}
}

func (f *ItemForm) cancel() {
	if f.onCancel != nil {
		f.onCancel()
	}
}

// moveField shifts focus to an adjacent field (down = next, up = previous) when
// the focus is on a single-line InputField (Name/Tags). It returns true when it
// moved focus, signalling the caller to consume the key. The Note TextArea is
// deliberately left alone so ↑/↓ keep moving its cursor for multi-line
// editing; from Note, return to Tags with Shift+Tab. Boundary moves (↑ on
// Name, ↓ on Tags into Note is allowed) that have no target also return false,
// so the key harmlessly falls through.
func (f *ItemForm) moveField(down bool) bool {
	item, _ := f.form.GetFocusedItemIndex()
	if item != mfFieldName && item != mfFieldTags {
		return false
	}
	target := item + 1
	if !down {
		target = item - 1
	}
	if target < 0 || target > mfFieldNote {
		return false
	}
	f.form.SetFocus(target)
	return true
}

// fieldText reads the current value of the i-th item, whether it is an
// InputField (Name/Tags) or a TextArea (Note).
func (f *ItemForm) fieldText(i int) string {
	switch v := f.form.GetFormItem(i).(type) {
	case *tview.InputField:
		return v.GetText()
	case *tview.TextArea:
		return v.GetText()
	}
	return ""
}

// Primitive returns a centered modal tall enough to show Name + Tags + Note
// (noteVisibleRows) plus the key hint, so focusing the Note never scrolls the
// Name field out of view. The hint sits below the bordered form and never
// takes focus (Tab stays within the form's items). Centering is delegated to
// centerModal (modal.go) so the layout idiom lives in one place.
func (f *ItemForm) Primitive() tview.Primitive {
	column := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(f.form, 0, 1, true).
		AddItem(f.hint, 1, 0, false)
	return centerModal(column, itemModalWidth, modalColumnHeight)
}

// Form returns the underlying form for SetFocus (focuses the first field) and
// for tests that need to inspect items.
func (f *ItemForm) Form() *tview.Form { return f.form }

// parseTags splits a tag input string on commas — accepting both the ASCII ","
// and the fullwidth Chinese "，" — trims surrounding whitespace from each token,
// and drops empty tokens. Unlike strings.Fields, spaces are NOT separators: a
// tag may itself contain spaces (e.g. "release v2"). (Ported from lazytmux
// handlers.go; the form takes a domain.Item, so the comma-joined edit string
// is parsed back into []string on submit.)
func parseTags(input string) []string {
	raw := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == '，'
	})
	tags := make([]string, 0, len(raw))
	for _, tag := range raw {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
