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
	"github.com/rivo/tview"
)

// StatusBar shows the keybinding hint line. SetStatus updates it (e.g. errors).
type StatusBar struct {
	*tview.TextView
}

func NewStatusBar() *StatusBar {
	sb := &StatusBar{TextView: tview.NewTextView()}
	sb.SetDynamicColors(true)
	sb.SetTextAlign(tview.AlignCenter)
	sb.SetBackgroundColor(tcell.ColorDefault)
	sb.SetText(defaultHints())
	return sb
}

// SetStatus replaces the hint line with a (possibly error) message.
func (s *StatusBar) SetStatus(msg string) { s.SetText(msg) }

// ResetHints restores the default keybinding hints.
func (s *StatusBar) ResetHints() { s.SetText(defaultHints()) }

// ShowEmpty swaps in the minimal empty-state hint. With no items loaded, most
// keys are no-ops, so we surface only the actions that still work plus a pointer
// to the full reference.
func (s *StatusBar) ShowEmpty() { s.SetText(emptyHints()) }

// defaultHints builds the footer hint line by walking the keyBindings single
// source and joining every entry flagged Footer==true. This is the fix for the
// lazytmux double-source drift: the footer is *derived* from the same table the
// help modal reads, so the two can never disagree. Add or change a footer key
// by editing keyBindings (and only keyBindings).
func defaultHints() string {
	var parts []string
	for _, kb := range keyBindings {
		if kb.Footer {
			parts = append(parts, "["+colorCyan+"]"+kb.Key+"[-] "+kb.Action)
		}
	}
	return strings.Join(parts, "  •  ")
}

// emptyHints is the footer for the no-items state — a lead-in label plus the few
// keys that remain meaningful when the list is empty. Hand-written because the
// empty state deliberately hides most keys (they would be no-ops), which the
// Footer flag on keyBindings cannot express.
func emptyHints() string {
	k := colorCyan
	return "[" + k + "]No items[-]  •  " +
		"[" + k + "]a[-] New  •  " +
		"[" + k + "]?[-] Help  •  " +
		"[" + k + "]q[-] Quit"
}
