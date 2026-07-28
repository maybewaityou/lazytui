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

// KeyBinding is one row of the help panel and the single source the footer hint
// line is derived from. Keeping every binding here means the help modal, the
// footer, and the docs can never drift apart — add a key once, in this slice.
//
// The Footer flag marks bindings that should also be surfaced in the one-line
// footer hint (defaultHints). This fixes the lazytmux design, where defaultHints
// was hand-written and drifted from this table: here it is derived (see
// status_bar.go), so the slice is the only source.
type KeyBinding struct {
	Group  string
	Key    string
	Action string
	Footer bool // true → also show in the footer single-line hint
}

// keyBindings is the single source of truth for advertised keys, grouped for
// display. Group strings double as the help modal's section headers. Order is
// significant: the help modal walks this slice in order and the contiguity test
// (TestKeyBindingsGroupsContiguous) forbids non-adjacent groups.
var keyBindings = []KeyBinding{
	{"Navigate", "↑↓", "Move", true},
	{"Navigate", "Enter", "Open details", true},
	{"Navigate", "←/→", "Focus list/details", false},
	{"Navigate", "/", "Search", true},
	{"Navigate", "q", "Quit", true},
	{"Item", "a", "New", true},
	{"Item", "e", "Edit", true},
	{"Item", "d", "Delete", true},
	{"Item", "r", "Refresh", true},
	{"Filter", "f", "Filter by tag", false},
	{"Metadata", "p", "Pin / unpin", false},
	{"Metadata", "n", "Edit note", false},
	{"Metadata", "c", "Clear tags", false},
	{"Other", "?", "Help", true},
}
