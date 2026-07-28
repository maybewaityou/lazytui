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

// TestEmptyHintsAnchorsUsefulKeys verifies the no-items footer surfaces the few
// keys that still work on an empty list, plus the help pointer.
func TestEmptyHintsAnchorsUsefulKeys(t *testing.T) {
	got := emptyHints()
	for _, want := range []string{"No items", "a", "?", "q"} {
		if !strings.Contains(got, "["+colorCyan+"]"+want+"[-]") {
			t.Errorf("emptyHints missing key %q in: %q", want, got)
		}
	}
}

// TestEmptyHintsOmitsNoOpKeys verifies the empty-state footer drops the keys
// that are meaningless with nothing selected.
func TestEmptyHintsOmitsNoOpKeys(t *testing.T) {
	got := emptyHints()
	for _, dead := range []string{"Edit", "Delete", "Refresh", "Pin", "Filter", "Enter", "Move", "Clear"} {
		if strings.Contains(got, dead) {
			t.Errorf("emptyHints should not advertise no-op %q: %q", dead, got)
		}
	}
}

// TestDefaultHintsDerivedFromKeyBindings complements
// TestKeyBindingsFooterDerivation at the status-bar layer: defaultHints() must
// carry every Footer==true key/action, and must NOT carry entries whose Footer
// flag is false (proving the derivation actually filters on the flag rather
// than dumping the whole table).
func TestDefaultHintsDerivedFromKeyBindings(t *testing.T) {
	got := defaultHints()
	if got == "" {
		t.Fatal("defaultHints empty")
	}
	for _, kb := range keyBindings {
		if kb.Footer {
			if !strings.Contains(got, kb.Key) {
				t.Errorf("defaultHints missing footer key %q: %q", kb.Key, got)
			}
			if !strings.Contains(got, kb.Action) {
				t.Errorf("defaultHints missing footer action %q: %q", kb.Action, got)
			}
		}
	}
	// Non-footer entries must be absent — otherwise the Footer flag is meaningless.
	for _, kb := range keyBindings {
		if kb.Footer {
			continue
		}
		if strings.Contains(got, kb.Action) {
			t.Errorf("defaultHints should not include non-footer action %q: %q", kb.Action, got)
		}
	}
}
