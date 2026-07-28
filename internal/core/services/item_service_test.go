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
	"errors"
	"sort"
	"testing"

	"github.com/maybewaityou/lazytui/internal/core/domain"
	"github.com/maybewaityou/lazytui/internal/core/ports"
)

// errExists is the fakeRepo's stand-in for store.ErrExists: Create / Rename
// return it on a name collision so tests can assert the service passes backend
// errors through verbatim without binding the test to the concrete store.
var errExists = errors.New("item already exists")

// fakeRepo is an in-memory ports.Repository backed by a map. Create rejects
// duplicates (mirroring the real store), letting the tests exercise the
// service's delegation and read-modify-write behavior without touching disk.
type fakeRepo struct {
	items map[string]domain.Item
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[string]domain.Item{}}
}

func (f *fakeRepo) List() ([]domain.Item, error) {
	out := make([]domain.Item, 0, len(f.items))
	for _, it := range f.items {
		out = append(out, it)
	}
	return out, nil
}

func (f *fakeRepo) Create(name string) (domain.Item, error) {
	if _, ok := f.items[name]; ok {
		return domain.Item{}, errExists
	}
	it := domain.Item{Name: name}
	f.items[name] = it
	return it, nil
}

// Update mimics the store's absent-name-is-noop contract so the service's
// read-modify-write path does not sneak a stale entry into the map.
func (f *fakeRepo) Update(name string, item domain.Item) error {
	if _, ok := f.items[name]; !ok {
		return nil
	}
	f.items[name] = item
	return nil
}

func (f *fakeRepo) Delete(name string) error {
	delete(f.items, name)
	return nil
}

func (f *fakeRepo) Rename(oldName, newName string) error {
	if _, ok := f.items[newName]; ok {
		return errExists
	}
	if it, ok := f.items[oldName]; ok {
		it.Name = newName
		f.items[newName] = it
		delete(f.items, oldName)
	}
	return nil
}

// Compile-time check that fakeRepo satisfies the port.
var _ ports.Repository = (*fakeRepo)(nil)

func TestCreateDelegates(t *testing.T) {
	repo := newFakeRepo()
	svc := NewItemService(repo)

	if err := svc.Create("alpha"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, _ := svc.List()
	if len(got) != 1 || got[0].Name != "alpha" {
		t.Fatalf("Create did not delegate to repo: %#v", got)
	}

	// Duplicate surfaces the repo's error verbatim.
	if err := svc.Create("alpha"); !errors.Is(err, errExists) {
		t.Fatalf("duplicate Create want errExists, got %v", err)
	}
}

func TestUpdateDelegates(t *testing.T) {
	repo := newFakeRepo()
	svc := NewItemService(repo)
	_ = svc.Create("alpha")

	updated := domain.Item{Name: "alpha", Note: "changed", Pinned: true}
	if err := svc.Update("alpha", updated); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := svc.List()
	if len(got) != 1 || got[0].Note != "changed" || !got[0].Pinned {
		t.Fatalf("Update did not delegate: %#v", got)
	}
}

func TestTogglePinPersists(t *testing.T) {
	repo := newFakeRepo()
	svc := NewItemService(repo)
	_ = svc.Create("alpha")

	if err := svc.TogglePin("alpha"); err != nil {
		t.Fatalf("TogglePin: %v", err)
	}
	// Re-List to prove the flip was persisted through the repo, not just
	// in-memory on a transient copy.
	if !findItem(t, svc, "alpha").Pinned {
		t.Fatal("Pinned not flipped to true after first TogglePin")
	}

	if err := svc.TogglePin("alpha"); err != nil {
		t.Fatalf("TogglePin second: %v", err)
	}
	if findItem(t, svc, "alpha").Pinned {
		t.Fatal("Pinned not flipped back to false after second TogglePin")
	}
}

func TestSaveTagsPersists(t *testing.T) {
	repo := newFakeRepo()
	svc := NewItemService(repo)
	_ = svc.Create("alpha")

	want := []string{"prod", "web"}
	if err := svc.SaveTags("alpha", want); err != nil {
		t.Fatalf("SaveTags: %v", err)
	}
	it := findItem(t, svc, "alpha")
	gotTags := append([]string(nil), it.Tags...)
	sort.Strings(gotTags)
	if len(gotTags) != 2 || gotTags[0] != "prod" || gotTags[1] != "web" {
		t.Fatalf("Tags not persisted: %#v", it.Tags)
	}
}

func TestSaveNotePersists(t *testing.T) {
	repo := newFakeRepo()
	svc := NewItemService(repo)
	_ = svc.Create("alpha")

	if err := svc.SaveNote("alpha", "primary box"); err != nil {
		t.Fatalf("SaveNote: %v", err)
	}
	if got := findItem(t, svc, "alpha").Note; got != "primary box" {
		t.Fatalf("Note not persisted: got %q want %q", got, "primary box")
	}
}

func TestDeleteDelegates(t *testing.T) {
	repo := newFakeRepo()
	svc := NewItemService(repo)
	_ = svc.Create("alpha")
	_ = svc.Create("beta")

	if err := svc.Delete("alpha"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, _ := svc.List()
	if len(got) != 1 || got[0].Name != "beta" {
		t.Fatalf("Delete did not delegate: %#v", got)
	}
}

func TestRenameDelegates(t *testing.T) {
	repo := newFakeRepo()
	svc := NewItemService(repo)
	_ = svc.Create("foo")

	if err := svc.Rename("foo", "bar"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	got, _ := svc.List()
	if len(got) != 1 || got[0].Name != "bar" {
		t.Fatalf("Rename did not delegate: %#v", got)
	}

	// Rename onto an existing name surfaces the repo's error verbatim.
	_ = svc.Create("baz")
	if err := svc.Rename("bar", "baz"); !errors.Is(err, errExists) {
		t.Fatalf("rename onto existing want errExists, got %v", err)
	}
}

// TestModifyAbsentReturnsErrNotFound exercises the read-modify-write helpers
// (TogglePin / SaveTags / SaveNote) against a name the repo does not have: the
// service must surface ports.ErrNotFound rather than silently no-oping.
func TestModifyAbsentReturnsErrNotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewItemService(repo)

	cases := []struct {
		op  string
		err error
	}{
		{"TogglePin", svc.TogglePin("ghost")},
		{"SaveTags", svc.SaveTags("ghost", nil)},
		{"SaveNote", svc.SaveNote("ghost", "x")},
	}
	for _, c := range cases {
		if !errors.Is(c.err, ports.ErrNotFound) {
			t.Errorf("%s on absent name: want ErrNotFound, got %v", c.op, c.err)
		}
	}
}

// findItem returns the named item from the service's current view, failing the
// test if it is absent — a helper so each test can assert post-mutation state
// without re-reading the repo directly.
func findItem(t *testing.T, svc *ItemService, name string) domain.Item {
	t.Helper()
	got, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, it := range got {
		if it.Name == name {
			return it
		}
	}
	t.Fatalf("item %q not found in %#v", name, got)
	return domain.Item{}
}
