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
)

// TestKeyBindingsComplete locks the set of keys the TUI advertises, so adding a
// new binding without registering it here (and thus in the help modal / footer
// / README) is caught. Every Key and Action must be non-empty, and keys must be
// unique.
func TestKeyBindingsComplete(t *testing.T) {
	if len(keyBindings) == 0 {
		t.Fatal("keyBindings must not be empty")
	}
	seen := make(map[string]bool, len(keyBindings))
	for _, kb := range keyBindings {
		if kb.Group == "" || kb.Key == "" || kb.Action == "" {
			t.Errorf("incomplete binding: %+v", kb)
		}
		if seen[kb.Key] {
			t.Errorf("duplicate key %q", kb.Key)
		}
		seen[kb.Key] = true
	}
	// The keys these features add must be present.
	for _, want := range []string{"f", "?", "n", "a", "e", "d", "r"} {
		if !seen[want] {
			t.Errorf("keyBindings missing key %q", want)
		}
	}
}

// TestKeyBindingsGroupsContiguous verifies bindings are grouped contiguously:
// once a group's run ends and a new group begins, the old group must never
// reappear later. This keeps the help modal's section rendering sane.
func TestKeyBindingsGroupsContiguous(t *testing.T) {
	seen := make(map[string]bool)
	prev := ""
	for _, kb := range keyBindings {
		if kb.Group != prev {
			if seen[kb.Group] {
				t.Errorf("group %q is not contiguous: reappears after %q", kb.Group, prev)
			}
			seen[kb.Group] = true
			prev = kb.Group
		}
	}
}

// TestKeyBindingsFooterDerivation is the core of this task: the footer hint
// line is *derived* from keyBindings (entries flagged Footer==true), not
// hand-written. This locks the single-source invariant that lazytmux violated.
//
// Concretely: every Footer==true entry must have non-empty key/action, and the
// derived defaultHints() must contain every Footer entry's Key and Action. The
// test would fail if someone re-introduced a hand-written defaultHints() that
// forgot an entry, or flipped a Footer flag without checking the footer.
func TestKeyBindingsFooterDerivation(t *testing.T) {
	// Every Footer==true entry has non-empty key/action.
	for _, kb := range keyBindings {
		if kb.Footer && (kb.Key == "" || kb.Action == "") {
			t.Fatalf("empty footer binding: %#v", kb)
		}
	}

	// defaultHints is non-empty and carries every Footer entry's key and action.
	hints := defaultHints()
	if hints == "" {
		t.Fatal("defaultHints is empty")
	}
	for _, kb := range keyBindings {
		if !kb.Footer {
			continue
		}
		if !strings.Contains(hints, kb.Key) {
			t.Errorf("defaultHints missing footer key %q: %q", kb.Key, hints)
		}
		if !strings.Contains(hints, kb.Action) {
			t.Errorf("defaultHints missing footer action %q: %q", kb.Action, hints)
		}
	}
}

// TestKeyBindingsHaveHandlers is the reverse invariant of
// TestKeyBindingsComplete: it verifies that every rune advertised in
// keyBindings actually has a handler in handleGlobalKeys (via the
// globalRuneHandlers dispatch table). This is the test that would have caught
// the original dead-binding bug — keyBindings registered "c" → "Clear tags"
// but handleGlobalKeys had no case for it, so pressing 'c' was a silent no-op.
//
// Scope: only single-rune keys are cross-checked here. Multi-rune / named keys
// (↑↓, ←/→, Enter) are widget-delegated (↑↓ falls through to the list) or
// handled as tcell.Key events in the e.Key() switch — neither class is a rune,
// so they are out of scope for this dispatch table.
func TestKeyBindingsHaveHandlers(t *testing.T) {
	dispatched := dispatchedRunes()

	// Sanity: the dispatch table is non-empty and the helper is wired.
	if len(dispatched) == 0 {
		t.Fatal("dispatchedRunes() is empty — globalRuneHandlers not wired")
	}

	// Every single-rune keyBinding must have a handler.
	for _, kb := range keyBindings {
		rs := []rune(kb.Key)
		if len(rs) != 1 {
			// Widget-delegated (↑↓) or tcell.Key-handled (←/→, Enter) — not a
			// rune dispatched by globalRuneHandlers.
			continue
		}
		r := rs[0]
		if !dispatched[r] {
			t.Errorf("keyBinding %q (%s: %s) has no handler in globalRuneHandlers — registered but dead",
				kb.Key, kb.Group, kb.Action)
		}
	}

	// Regression guard for the original gap: 'c' (Clear tags) was registered
	// in keyBindings with no handler, so pressing it was a silent no-op. If
	// this fails, the clear-tags handler has been removed or renamed away.
	if !dispatched['c'] {
		t.Error("rune 'c' (Clear tags) must be dispatched — regression of the dead-binding bug")
	}
}
