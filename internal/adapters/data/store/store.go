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

package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/maybewaityou/lazytui/internal/core/domain"
	"github.com/maybewaityou/lazytui/internal/core/ports"
)

// ErrExists is returned by Create / Rename when the target name is already in use.
var ErrExists = errors.New("item already exists")

// fileModel is the on-disk JSON shape: items keyed by Name.
type fileModel struct {
	Items map[string]domain.Item `json:"items"`
}

// Store implements ports.Repository with thread-safe in-memory state backed by a
// JSON file written atomically.
type Store struct {
	mu   sync.RWMutex
	path string
	data fileModel
}

// NewStore loads (or creates) the item store at path.
func NewStore(path string) (*Store, error) {
	s := &Store{
		path: path,
		data: fileModel{
			Items: map[string]domain.Item{},
		},
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, &s.data)
}

// reloadLocked refreshes the in-memory snapshot from the latest on-disk state.
// It must be called with the write lock held. A missing or empty file leaves the
// snapshot untouched (the map is already initialized, and between mutations the
// snapshot already matches the last successful save).
//
// The JSON is decoded into a fresh fileModel rather than &s.data because
// json.Unmarshal *merges* into existing maps — it never drops keys absent from
// the payload — so reusing s.data would leave stale entries behind.
func (s *Store) reloadLocked() error {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) || len(b) == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	fresh := fileModel{
		Items: map[string]domain.Item{},
	}
	if err := json.Unmarshal(b, &fresh); err != nil {
		return err
	}
	s.data = fresh
	return nil
}

// mutate re-reads the freshest on-disk state under the write lock, applies fn to
// it, and persists atomically. Re-reading before every write stops one store
// instance from silently clobbering another's concurrent change: each mutation
// becomes a read-modify-write against the latest file instead of a blind
// overwrite of the snapshot captured at startup.
func (s *Store) mutate(fn func(d *fileModel)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.reloadLocked(); err != nil {
		return err
	}
	fn(&s.data)
	return s.save()
}

// save backs up the original file the first time, then writes atomically.
func (s *Store) save() error {
	if err := s.backupOriginalOnce(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) backupOriginalOnce() error {
	backup := s.path + ".original.backup"
	if _, err := os.Stat(backup); err == nil {
		return nil // already exists, never overwrite
	}
	src, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(backup, src, 0o644)
}

// relocate moves a single map entry with mv semantics: the target is cleared
// first (overwrite), then the source value — if present — is moved across,
// then the source key is removed. Generic over the value type so it serves any
// keyed map without reflection.
func relocate[V any](m map[string]V, oldName, newName string) {
	delete(m, newName)
	if v, ok := m[oldName]; ok {
		m[newName] = v
	}
	delete(m, oldName)
}

// --- ports.Repository ---

func (s *Store) List() ([]domain.Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Item, 0, len(s.data.Items))
	for _, it := range s.data.Items {
		out = append(out, it)
	}
	return out, nil
}

func (s *Store) Create(name string) (domain.Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.reloadLocked(); err != nil {
		return domain.Item{}, err
	}
	if _, ok := s.data.Items[name]; ok {
		return domain.Item{}, ErrExists
	}
	it := domain.Item{Name: name, Created: time.Now()}
	s.data.Items[name] = it
	if err := s.save(); err != nil {
		return domain.Item{}, err
	}
	return it, nil
}

func (s *Store) Update(name string, item domain.Item) error {
	return s.mutate(func(d *fileModel) {
		if _, ok := d.Items[name]; ok {
			d.Items[name] = item
		}
	})
}

func (s *Store) Delete(name string) error {
	return s.mutate(func(d *fileModel) { delete(d.Items, name) })
}

func (s *Store) Rename(oldName, newName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.reloadLocked(); err != nil {
		return err
	}
	if _, ok := s.data.Items[newName]; ok {
		return ErrExists
	}
	relocate(s.data.Items, oldName, newName)
	// relocate preserves the moved struct verbatim, but Item.Name is denormalized
	// (it is both the map key and a field), so the identity field must track the
	// new key. This is the only field touched — Tags/Note/Pinned/Created are
	// inherited unchanged from the source, matching mv semantics.
	if it, ok := s.data.Items[newName]; ok {
		it.Name = newName
		s.data.Items[newName] = it
	}
	return s.save()
}

// Compile-time check that Store satisfies the port.
var _ ports.Repository = (*Store)(nil)
