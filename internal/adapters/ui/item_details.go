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

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maybewaityou/lazytui/internal/core/domain"
)

// ItemDetails is the right-hand pane showing an item's metadata, grouped into
// titled sections via RenderSections (render.go). An Item has no window list
// and no notion of an attached/active entry, so this pane renders only Basic +
// optional Metadata groups — no window/attachment section.
type ItemDetails struct {
	*tview.TextView
}

// NewItemDetails builds the details pane with a centered placeholder; Render
// flips alignment back to left for multi-line content.
func NewItemDetails() *ItemDetails {
	d := &ItemDetails{TextView: tview.NewTextView().SetDynamicColors(true).SetWrap(true)}
	d.SetBorder(true).
		SetTitle(" Details ").
		SetTitleColor(tcell.GetColor(colorTitle)).
		SetBorderColor(tcell.GetColor(colorBorder))
	// Placeholder states (initial + empty) read better centered; Render flips
	// back to left-alignment for multi-line item content.
	d.SetTextAlign(tview.AlignCenter)
	d.SetText("[" + colorSecondary + "]select an item[-]")
	return d
}

// Render fills the pane. It always emits a Basic section (name/pinned/created)
// and additionally emits a Metadata section when the item carries tags or a
// note — so a bare item renders one group, not a hollow two-group layout. The
// title line is the item name in accent+bold, followed by a blank line.
func (d *ItemDetails) Render(it domain.Item) {
	d.SetTextAlign(tview.AlignLeft)
	sections := []Section{
		{Title: "Basic", Fields: []Field{
			{Label: "name", Value: it.Name, Kind: FieldText},
			{Label: "pinned", Value: boolStr(it.Pinned), Kind: FieldText},
			{Label: "created", Value: fmt.Sprintf("%s · %s", it.Created.Format("2006-01-02 15:04"), humanizeDuration(it.Created)), Kind: FieldText},
		}},
	}
	if len(it.Tags) > 0 || it.Note != "" {
		md := Section{Title: "Metadata"}
		if len(it.Tags) > 0 {
			md.Fields = append(md.Fields, Field{Label: "tags", Value: strings.Join(it.Tags, ","), Kind: FieldTag})
		}
		if it.Note != "" {
			md.Fields = append(md.Fields, Field{Label: "note", Value: it.Note, Kind: FieldNote})
		}
		sections = append(sections, md)
	}
	title := "[" + colorAccent + "::b]" + it.Name + "[-]\n\n"
	d.SetText(title + RenderSections(sections))
}

// boolStr renders a bool as the literal "true"/"false" shown in the details
// pane, mirroring the upstream pinned rendering so pinned values keep the
// same surface across the toolchain.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
