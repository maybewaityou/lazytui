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
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/maybewaityou/lazytui/internal/core/domain"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "items.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func TestCreateListRoundTrip(t *testing.T) {
	s, err := NewStore(filepath.Join(t.TempDir(), "items.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("alpha"); err != nil {
		t.Fatal(err)
	}
	got, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "alpha" {
		t.Fatalf("got %#v", got)
	}
}

func TestCreateDuplicate(t *testing.T) {
	s, _ := NewStore(filepath.Join(t.TempDir(), "items.json"))
	if _, err := s.Create("alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("alpha"); err == nil {
		t.Fatal("want duplicate error")
	}
}

func TestCreateDuplicateReturnsErrExists(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create("alpha"); err != nil {
		t.Fatal(err)
	}
	_, err := s.Create("alpha")
	if err == nil {
		t.Fatal("want duplicate error")
	}
	if !errors.Is(err, ErrExists) {
		t.Fatalf("want ErrExists, got %v", err)
	}
}

func TestCreateSetsCreatedTimestamp(t *testing.T) {
	s := newTestStore(t)
	before := time.Now()
	it, err := s.Create("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if it.Created.Before(before) {
		t.Fatalf("Created before Create() call: %v < %v", it.Created, before)
	}
	if it.Name != "alpha" {
		t.Fatalf("returned item name: got %q want alpha", it.Name)
	}
}

func TestUpdateChangesFields(t *testing.T) {
	s := newTestStore(t)
	it, _ := s.Create("alpha")
	it.Tags = []string{"prod", "web"}
	it.Note = "primary box"
	it.Pinned = true

	if err := s.Update("alpha", it); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// New instance reads from disk to prove persistence, not just in-memory.
	s2, err := NewStore(s.path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, _ := s2.List()
	if len(got) != 1 {
		t.Fatalf("want 1 item, got %d", len(got))
	}
	gotTags := append([]string(nil), got[0].Tags...)
	sort.Strings(gotTags)
	if got[0].Note != "primary box" || !got[0].Pinned || len(gotTags) != 2 || gotTags[0] != "prod" || gotTags[1] != "web" {
		t.Fatalf("fields not persisted: %#v", got[0])
	}
}

func TestUpdateAbsentNameIsNoop(t *testing.T) {
	s := newTestStore(t)
	// Updating a name that does not exist must not create a stale entry.
	if err := s.Update("ghost", domain.Item{Note: "nope"}); err != nil {
		t.Fatalf("Update absent: %v", err)
	}
	got, _ := s.List()
	if len(got) != 0 {
		t.Fatalf("absent Update should be noop, got %#v", got)
	}
}

func TestDeleteRemovesFromList(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.Create("alpha")
	_, _ = s.Create("beta")

	if err := s.Delete("alpha"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, _ := s.List()
	if len(got) != 1 || got[0].Name != "beta" {
		t.Fatalf("after delete got %#v", got)
	}
	// Deleting a missing name is not an error.
	if err := s.Delete("ghost"); err != nil {
		t.Fatalf("Delete ghost: %v", err)
	}
}

func TestRenameMvSemantics(t *testing.T) {
	s, _ := NewStore(filepath.Join(t.TempDir(), "items.json"))
	it, _ := s.Create("old")
	it.Pinned = true
	_ = s.Update("old", it)
	if err := s.Rename("old", "new"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.List()
	if len(got) != 1 || got[0].Name != "new" || !got[0].Pinned {
		t.Fatalf("got %#v", got)
	}
	if err := s.Rename("x", "new"); err == nil {
		t.Fatal("want exists error")
	}
}

func TestRenamePreservesAllFields(t *testing.T) {
	s := newTestStore(t)
	it, _ := s.Create("foo")
	it.Tags = []string{"work", "prod"}
	it.Note = "primary"
	it.Pinned = true
	created := it.Created
	_ = s.Update("foo", it)

	if err := s.Rename("foo", "bar"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	// old name fully cleared
	got, _ := s.List()
	for _, it := range got {
		if it.Name == "foo" {
			t.Fatalf("old name should be cleared: %#v", got)
		}
	}
	// find bar
	var bar *domain.Item
	for i := range got {
		if got[i].Name == "bar" {
			bar = &got[i]
		}
	}
	if bar == nil {
		t.Fatalf("bar not found: %#v", got)
	}
	gotTags := append([]string(nil), bar.Tags...)
	sort.Strings(gotTags)
	if !bar.Pinned || bar.Note != "primary" || len(gotTags) != 2 || gotTags[0] != "prod" || gotTags[1] != "work" {
		t.Fatalf("fields did not migrate: %#v", bar)
	}
	if !bar.Created.Equal(created) {
		t.Fatalf("Created timestamp not preserved: got %v want %v", bar.Created, created)
	}
}

func TestRenameTargetExistsReturnsErrExists(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.Create("foo")
	_, _ = s.Create("bar")

	if err := s.Rename("foo", "bar"); !errors.Is(err, ErrExists) {
		t.Fatalf("Rename onto existing name: want ErrExists, got %v", err)
	}
	// original entries untouched after a rejected rename
	got, _ := s.List()
	names := map[string]bool{}
	for _, it := range got {
		names[it.Name] = true
	}
	if !names["foo"] || !names["bar"] {
		t.Fatalf("rejected rename mutated state: %#v", got)
	}
}

func TestPersistenceAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "items.json")
	s1, _ := NewStore(path)
	_, _ = s1.Create("dev")

	s2, err := NewStore(path) // reload from disk
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, _ := s2.List()
	if len(got) != 1 || got[0].Name != "dev" {
		t.Fatalf("Create did not persist to disk: %#v", got)
	}
}

func TestAtomicWriteNoTmpLeftBehind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "items.json")
	s, _ := NewStore(path)
	_, _ = s.Create("alpha")
	_, _ = s.Create("beta")

	// After successful writes the .tmp staging file must be gone (renamed away).
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatal(".tmp file left behind after successful save")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat .tmp: %v", err)
	}

	// Main file must exist and be valid JSON readable by a fresh instance.
	s2, err := NewStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, _ := s2.List()
	if len(got) != 2 {
		t.Fatalf("want 2 items after reopen, got %d", len(got))
	}
}

func TestBackupOriginalOnceCreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "items.json")
	// Seed a pre-existing items.json so the backup has something to capture.
	seed := []byte(`{"items":{"seed":{"Name":"seed","Tags":null,"Note":"","Pinned":false,"Created":"0001-01-01T00:00:00Z"}}}`)
	if err := os.WriteFile(path, seed, 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	// First mutation triggers the backup of the original file.
	if _, err := s.Create("alpha"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	backup := path + ".original.backup"
	bb, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("backup not created: %v", err)
	}
	if string(bb) != string(seed) {
		t.Fatalf("backup content = original seed mismatch:\n got %q\nwant %q", bb, seed)
	}

	// Second mutation must NOT overwrite the backup.
	if _, err := s.Create("beta"); err != nil {
		t.Fatalf("Create beta: %v", err)
	}
	bb2, _ := os.ReadFile(backup)
	if string(bb2) != string(seed) {
		t.Fatal("backup was overwritten by a later mutation")
	}
}

// TestConcurrentCreateDoesNotLoseUpdates mirrors lazytmux's
// TestConcurrentInstancesDontClobber: two Store instances pointing at the same
// file must not silently clobber each other's writes, because every mutation
// re-reads the freshest on-disk state before saving (reloadLocked inside
// mutate / hand-locked Create).
func TestConcurrentCreateDoesNotLoseUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "items.json")
	s1, _ := NewStore(path) // loads empty
	s2, _ := NewStore(path) // loads empty — both startup snapshots are empty

	const n = 50
	var wg sync.WaitGroup
	wg.Add(2 * n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = s1.Create("a")
		}()
		go func() {
			defer wg.Done()
			_, _ = s2.Create("b")
		}()
	}
	wg.Wait()

	s3, err := NewStore(path) // re-read from disk
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, _ := s3.List()
	names := map[string]bool{}
	for _, it := range got {
		names[it.Name] = true
	}
	// Both distinct names must survive despite concurrent stale snapshots.
	if !names["a"] || !names["b"] {
		t.Fatalf("concurrent create lost an update: %#v", names)
	}
	if len(got) != 2 {
		t.Fatalf("want exactly 2 items, got %d", len(got))
	}
}

// TestConcurrentDistinctCreatesWithinOneInstance exercises the -race detector
// against a single Store: many goroutines each creating a unique name must end
// up with all of them persisted, none lost to a data race on the in-memory map.
func TestConcurrentDistinctCreatesWithinOneInstance(t *testing.T) {
	s := newTestStore(t)
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_, _ = s.Create(uniqueName(i))
		}(i)
	}
	wg.Wait()

	got, _ := s.List()
	if len(got) != n {
		t.Fatalf("want %d items, got %d (updates lost)", n, len(got))
	}
}

func uniqueName(i int) string {
	// 0..99 -> "i00".."i99" so they sort and stay distinct.
	return "i" + string(rune('0'+i/10)) + string(rune('0'+i%10))
}
