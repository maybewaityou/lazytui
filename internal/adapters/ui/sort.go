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
	"sort"

	"github.com/maybewaityou/lazytui/internal/core/domain"
)

// SortMode selects the field used to order the item list.
type SortMode int

const (
	SortByNameAsc SortMode = iota
	SortByCreatedDesc
)

func (s SortMode) String() string {
	switch s {
	case SortByCreatedDesc:
		return "Created ↓"
	default:
		return "Name ↑"
	}
}

// Next cycles to the following sort mode.
func (s SortMode) Next() SortMode { return (s + 1) % 2 }

// sortItemsForUI sorts in place: pinned items always first, then by mode
// (Name ascending, or Created descending).
func sortItemsForUI(items []domain.Item, mode SortMode) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		switch mode {
		case SortByCreatedDesc:
			return items[i].Created.After(items[j].Created)
		default:
			return items[i].Name < items[j].Name
		}
	})
}
