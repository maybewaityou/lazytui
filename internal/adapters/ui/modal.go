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
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// centerModal wraps a primitive so it is rendered centered on screen.
//
// The layout is the lazytmux "Flex + spacer Items" idiom (lifted from
// lazytmux/internal/adapters/ui/handlers.go openHelp): an outer horizontal
// Flex [nil-spacer | column | nil-spacer] wraps an inner vertical Flex
// [nil-spacer | content | nil-spacer]. Proportional spacers on every side
// keep the content visually centered at any terminal size. width fixes the
// inner column; height fixes the content row. Callers usually mount the
// result via app.SetRoot(centerModal(...), true).
func centerModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false). // left spacer
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).  // top spacer
			AddItem(p, height, 0, true). // content
			AddItem(nil, 0, 1, false),   // bottom spacer
			width, 0, true).
		AddItem(nil, 0, 1, false) // right spacer
}

// ConfirmModal returns a centered yes/no confirmation modal.
//
// "Confirm" (or Enter on the focused button) runs onYes; "Cancel" or Esc
// dismisses without side effects. The caller is responsible for unmounting
// the modal after onYes (or wiring the cancel path) — this helper only
// builds the widget, mirroring lazytmux's show*ConfirmModal constructors.
// Mount it via app.SetRoot(ConfirmModal(...), true) directly, or wrap with
// centerModal when a fixed on-screen size is desired.
func ConfirmModal(title, msg string, onYes func()) *tview.Modal {
	m := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"Confirm", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Confirm" {
				onYes()
			}
		})
	m.SetBorder(true).
		SetTitle(title).
		SetBackgroundColor(tcell.ColorDefault)
	return m
}
