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
	"testing"

	"github.com/gdamore/tcell/v2"
)

// TestNewSearchBarBuild confirms the builder wires label/title/border without
// panicking and that the public accessors round-trip the expected strings.
func TestNewSearchBarBuild(t *testing.T) {
	sb := NewSearchBar()
	if got := sb.GetTitle(); got != " Search " {
		t.Errorf("GetTitle = %q, want %q", got, " Search ")
	}
	if got := sb.GetLabel(); got != " 🔍 " {
		t.Errorf("GetLabel = %q, want %q", got, " 🔍 ")
	}
}

// TestSearchBarOnSearchInvoked drives SetChangedFunc through tview's own
// text setter, confirming the onSearch callback wired in build() fires with
// the latest text. The nil-callback path is also covered.
func TestSearchBarOnSearchInvoked(t *testing.T) {
	// nil callback must not panic when text changes.
	sb := NewSearchBar()
	sb.SetText("noop")

	var lastSearch string
	var calls int
	sb.OnSearch(func(s string) { lastSearch = s; calls++ })

	sb.SetText("api")
	if calls != 1 || lastSearch != "api" {
		t.Errorf("onSearch after SetText: calls=%d last=%q, want 1 / %q", calls, lastSearch, "api")
	}
}

// TestSearchBarChaining confirms the On* setters return the receiver so they
// can be chained at construction sites.
func TestSearchBarChaining(t *testing.T) {
	sb := NewSearchBar()
	if sb.OnSearch(nil) != sb || sb.OnEscape(nil) != sb || sb.OnNavigate(nil) != sb {
		t.Errorf("On* setters must return *SearchBar for chaining")
	}
}

// TestSearchBarInputCaptureRoutesToCallbacks injects the capture func through
// the public InputField API (GetInputCapture is provided by tview.Primitive)
// and checks that ESC/Enter/Down/Up are swallowed and routed, while an
// unrelated key passes through.
func TestSearchBarInputCaptureRoutesToCallbacks(t *testing.T) {
	sb := NewSearchBar()

	// Before callbacks are registered, the capture must still swallow the
	// routed keys (returning nil) without panicking on nil handlers.
	cap := sb.GetInputCapture()
	for _, k := range []tcell.Key{tcell.KeyESC, tcell.KeyEnter, tcell.KeyDown, tcell.KeyUp} {
		if hk := cap(tcell.NewEventKey(k, 0, tcell.ModNone)); hk != nil {
			t.Errorf("nil-callback key %v not swallowed: %v", k, hk)
		}
	}

	var escapeCalls, navigateCalls int
	sb.OnEscape(func() { escapeCalls++ })
	sb.OnNavigate(func() { navigateCalls++ })

	if hk := cap(tcell.NewEventKey(tcell.KeyESC, 0, tcell.ModNone)); hk != nil {
		t.Errorf("ESC not swallowed: %v", hk)
	}
	if escapeCalls != 1 {
		t.Errorf("onEscape want 1 call, got %d", escapeCalls)
	}

	for _, k := range []tcell.Key{tcell.KeyEnter, tcell.KeyDown, tcell.KeyUp} {
		if hk := cap(tcell.NewEventKey(k, 0, tcell.ModNone)); hk != nil {
			t.Errorf("key %v not swallowed: %v", k, hk)
		}
	}
	if navigateCalls != 3 {
		t.Errorf("onNavigate want 3 calls, got %d", navigateCalls)
	}

	// unrelated key passes through untouched.
	if hk := cap(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)); hk == nil {
		t.Errorf("unrelated key was swallowed unexpectedly")
	}
}
