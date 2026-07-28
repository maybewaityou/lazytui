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
	"time"

	"github.com/maybewaityou/lazytui/internal/core/domain"
)

func TestItemDetailsRenderBasicGroup(t *testing.T) {
	d := NewItemDetails()
	created := time.Date(2026, 7, 27, 14, 30, 0, 0, time.UTC)
	d.Render(domain.Item{Name: "api", Pinned: true, Created: created})
	got := d.GetText(false) // keep color tags so the title tag is visible

	if !strings.Contains(got, "Basic") {
		t.Errorf("Basic group title missing: %q", got)
	}
	if !strings.Contains(got, "name:") {
		t.Errorf("name label missing: %q", got)
	}
	if !strings.Contains(got, "["+colorPrimary+"]api[-]") {
		t.Errorf("name value missing: %q", got)
	}
	if !strings.Contains(got, "pinned:") {
		t.Errorf("pinned label missing: %q", got)
	}
	if !strings.Contains(got, "["+colorPrimary+"]true[-]") {
		t.Errorf("pinned=true value missing: %q", got)
	}
	if !strings.Contains(got, "created:") {
		t.Errorf("created label missing: %q", got)
	}
	if !strings.Contains(got, "2026-07-27 14:30") {
		t.Errorf("created value missing: %q", got)
	}
}

func TestItemDetailsRenderPinnedFalse(t *testing.T) {
	d := NewItemDetails()
	d.Render(domain.Item{Name: "api", Created: time.Now()})
	got := d.GetText(false)
	if !strings.Contains(got, "["+colorPrimary+"]false[-]") {
		t.Errorf("pinned=false value missing: %q", got)
	}
}

func TestItemDetailsRenderTitleColored(t *testing.T) {
	d := NewItemDetails()
	d.Render(domain.Item{Name: "api", Created: time.Now()})
	got := d.GetText(false)
	// Title is the item name wrapped in the accent+bold tag, followed by a blank line.
	if !strings.Contains(got, "["+colorAccent+"::b]api[-]") {
		t.Errorf("accent/bold title missing: %q", got)
	}
}

func TestItemDetailsRenderMetadataWithTagsAndNote(t *testing.T) {
	d := NewItemDetails()
	d.Render(domain.Item{
		Name:    "api",
		Tags:    []string{"work", "urgent"},
		Note:    "primary box",
		Created: time.Now(),
	})
	got := d.GetText(false)

	if !strings.Contains(got, "Metadata") {
		t.Errorf("Metadata group title missing: %q", got)
	}
	if !strings.Contains(got, "tags:") {
		t.Errorf("tags label missing: %q", got)
	}
	if !strings.Contains(got, "[black:"+colorAccent+"] work [-:-:-]") {
		t.Errorf("first tag chip missing: %q", got)
	}
	if !strings.Contains(got, "[black:"+colorAccent+"] urgent [-:-:-]") {
		t.Errorf("second tag chip missing: %q", got)
	}
	if !strings.Contains(got, "note:") {
		t.Errorf("note label missing: %q", got)
	}
	if !strings.Contains(got, "["+colorPrimary+"]primary box[-]") {
		t.Errorf("note text value missing: %q", got)
	}
}

func TestItemDetailsRenderMetadataTagsOnly(t *testing.T) {
	d := NewItemDetails()
	d.Render(domain.Item{
		Name:    "api",
		Tags:    []string{"work"},
		Created: time.Now(),
	})
	got := d.GetText(false)
	if !strings.Contains(got, "Metadata") {
		t.Errorf("Metadata group should render when tags present: %q", got)
	}
	if !strings.Contains(got, "[black:"+colorAccent+"] work [-:-:-]") {
		t.Errorf("tag chip missing: %q", got)
	}
	if strings.Contains(got, "note:") {
		t.Errorf("empty note must not render a note line: %q", got)
	}
}

func TestItemDetailsRenderMetadataNoteOnly(t *testing.T) {
	d := NewItemDetails()
	d.Render(domain.Item{
		Name:    "api",
		Note:    "just a note",
		Created: time.Now(),
	})
	got := d.GetText(false)
	if !strings.Contains(got, "Metadata") {
		t.Errorf("Metadata group should render when note present: %q", got)
	}
	if !strings.Contains(got, "["+colorPrimary+"]just a note[-]") {
		t.Errorf("note value missing: %q", got)
	}
	if strings.Contains(got, "tags:") {
		t.Errorf("empty tags must not render a tags line: %q", got)
	}
}

// TestItemDetailsRenderNoMetadataWhenEmpty is the load-bearing assertion: with
// no tags AND no note the Metadata section is dropped entirely — only Basic renders.
func TestItemDetailsRenderNoMetadataWhenEmpty(t *testing.T) {
	d := NewItemDetails()
	d.Render(domain.Item{Name: "api", Created: time.Now()})
	if got := d.GetText(true); strings.Contains(got, "Metadata") {
		t.Errorf("Metadata group must not render without tags/note: %q", got)
	}
}
