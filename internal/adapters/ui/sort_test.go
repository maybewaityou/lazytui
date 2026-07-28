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
	"time"

	"github.com/maybewaityou/lazytui/internal/core/domain"
)

func TestSortItemsForUIByName(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	items := []domain.Item{
		{Name: "charlie", Created: t1},
		{Name: "alpha", Created: t2},
		{Name: "bravo", Created: t1},
	}
	sortItemsForUI(items, SortByNameAsc)
	got := []string{items[0].Name, items[1].Name, items[2].Name}
	want := []string{"alpha", "bravo", "charlie"}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("idx %d: got %v want %v", i, got, want)
		}
	}
}

func TestSortItemsForUIByCreated(t *testing.T) {
	oldest := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newest := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	items := []domain.Item{
		{Name: "old", Created: oldest},
		{Name: "new", Created: newest},
		{Name: "mid", Created: mid},
	}
	sortItemsForUI(items, SortByCreatedDesc)
	got := []string{items[0].Name, items[1].Name, items[2].Name}
	want := []string{"new", "mid", "old"}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("idx %d: got %v want %v", i, got, want)
		}
	}
}

// Pinned always wins regardless of sort mode.
func TestSortItemsForUIPinnedFirst(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, mode := range []SortMode{SortByNameAsc, SortByCreatedDesc} {
		items := []domain.Item{
			{Name: "zulu-pinned", Created: t1},
			{Name: "alpha-unpinned", Created: t2},
			{Name: "mid-unpinned", Created: t1},
		}
		// Pin the one that would sort last by either mode.
		items[0].Pinned = true
		sortItemsForUI(items, mode)
		if !items[0].Pinned {
			t.Fatalf("mode %v: expected pinned item first, got %q", mode, items[0].Name)
		}
	}
}

func TestSortModeNext(t *testing.T) {
	if got := SortByNameAsc.Next(); got != SortByCreatedDesc {
		t.Errorf("SortByNameAsc.Next()=%v want %v", got, SortByCreatedDesc)
	}
	if got := SortByCreatedDesc.Next(); got != SortByNameAsc {
		t.Errorf("SortByCreatedDesc.Next()=%v want %v", got, SortByNameAsc)
	}
}

func TestSortModeString(t *testing.T) {
	if got := SortByNameAsc.String(); got != "Name ↑" {
		t.Errorf("SortByNameAsc.String()=%q want %q", got, "Name ↑")
	}
	if got := SortByCreatedDesc.String(); got != "Created ↓" {
		t.Errorf("SortByCreatedDesc.String()=%q want %q", got, "Created ↓")
	}
}
