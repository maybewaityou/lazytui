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

package services

import (
	"github.com/maybewaityou/lazytui/internal/core/domain"
	"github.com/maybewaityou/lazytui/internal/core/ports"
)

// ItemService is the business-logic layer over the Item aggregate. It owns no
// state of its own: every mutating op is either a pass-through to the
// Repository or a read-modify-write against the latest Repository state. This
// mirrors lazytmux's SessionService shape but drops the tmux-only operations
// (window enumeration / attach / kill / suspend) and folds the per-field
// metadata edits (Pin / Tags / Note) into the Item itself rather than a side
// metadata store, since lazytui persists them as Item fields in the same JSON.
type ItemService struct {
	repo ports.Repository
}

// NewItemService wires the service to its Repository dependency.
func NewItemService(repo ports.Repository) *ItemService {
	return &ItemService{repo: repo}
}

func (s *ItemService) List() ([]domain.Item, error) { return s.repo.List() }

func (s *ItemService) Create(name string) error {
	_, err := s.repo.Create(name)
	return err
}

func (s *ItemService) Update(name string, item domain.Item) error {
	return s.repo.Update(name, item)
}

func (s *ItemService) Delete(name string) error { return s.repo.Delete(name) }

func (s *ItemService) Rename(oldName, newName string) error {
	return s.repo.Rename(oldName, newName)
}

func (s *ItemService) TogglePin(name string) error {
	return s.modify(name, func(it *domain.Item) { it.Pinned = !it.Pinned })
}

func (s *ItemService) SaveTags(name string, tags []string) error {
	return s.modify(name, func(it *domain.Item) { it.Tags = tags })
}

func (s *ItemService) SaveNote(name string, note string) error {
	return s.modify(name, func(it *domain.Item) { it.Note = note })
}

// modify is the shared read-modify-write helper for single-field metadata
// edits: it locates the named Item via the latest repo snapshot, applies fn to
// a local copy, and persists the result back through Update. An absent name
// short-circuits with ports.ErrNotFound before any write is attempted.
func (s *ItemService) modify(name string, fn func(*domain.Item)) error {
	it, err := s.find(name)
	if err != nil {
		return err
	}
	fn(&it)
	return s.repo.Update(name, it)
}

// find scans the current repo snapshot for an Item with the given Name. Used by
// modify instead of a hypothetical repo.Get to keep the Repository port narrow
// (the JSON store has no Get method either).
func (s *ItemService) find(name string) (domain.Item, error) {
	items, err := s.repo.List()
	if err != nil {
		return domain.Item{}, err
	}
	for _, it := range items {
		if it.Name == name {
			return it, nil
		}
	}
	return domain.Item{}, ports.ErrNotFound
}

// Compile-time check that ItemService satisfies the port.
var _ ports.Service = (*ItemService)(nil)
