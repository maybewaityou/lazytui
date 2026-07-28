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

	"github.com/rivo/tview"
)

func TestRenderSectionsGroups(t *testing.T) {
	out := RenderSections([]Section{
		{Title: "Basic", Fields: []Field{{Label: "name", Value: "alpha", Kind: FieldText}}},
		{Title: "Metadata", Fields: []Field{{Label: "tags", Value: "x,y", Kind: FieldTag}}},
	})
	if !strings.Contains(out, "Basic") || !strings.Contains(out, "Metadata") {
		t.Fatal("missing groups")
	}
	if !strings.Contains(out, "alpha") {
		t.Fatal("missing value")
	}
}

func TestRenderSectionsColoredTitle(t *testing.T) {
	// Each section title is wrapped in the colorTitle tag so it renders bold/blue.
	out := RenderSections([]Section{{Title: "Header", Fields: nil}})
	if !strings.Contains(out, "["+colorTitle+"::b]Header[-]") {
		t.Fatalf("title not color-tagged, got %q", out)
	}
}

func TestRenderSectionsTagChips(t *testing.T) {
	out := RenderSections([]Section{
		{Title: "Meta", Fields: []Field{{Label: "tags", Value: "dev,urgent", Kind: FieldTag}}},
	})
	// Tag chips render as black-on-accent pills; both tags must appear, and
	// the value must NOT be wrapped in colorPrimary (chips carry their own tags).
	if !strings.Contains(out, "[black:"+colorAccent+"] dev ") {
		t.Fatalf("missing first chip, got %q", out)
	}
	if !strings.Contains(out, "[black:"+colorAccent+"] urgent ") {
		t.Fatalf("missing second chip, got %q", out)
	}
	if !strings.Contains(out, "tags:") {
		t.Fatalf("missing tag label, got %q", out)
	}
}

func TestRenderSectionsMultipleSectionsSeparated(t *testing.T) {
	out := RenderSections([]Section{
		{Title: "One", Fields: []Field{{Label: "a", Value: "1", Kind: FieldText}}},
		{Title: "Two", Fields: []Field{{Label: "b", Value: "2", Kind: FieldText}}},
	})
	// Two section titles plus two values, and a blank line between sections.
	if strings.Count(out, "\n") < 4 {
		t.Fatalf("expected multi-line output with section break, got %q", out)
	}
	if !strings.Contains(out, "One") || !strings.Contains(out, "Two") || !strings.Contains(out, "1") || !strings.Contains(out, "2") {
		t.Fatalf("missing section content, got %q", out)
	}
}

func TestDetailFieldAlignsAndColors(t *testing.T) {
	got := detailField("name", "alpha")
	// Label is indented, in colorSecondary, ending with colon; value in colorPrimary.
	// The colon is part of the colored label text, then [-] closes the tag
	// (verbatim detailField format: "  [color]label:[-]").
	if !strings.HasPrefix(got, "  ["+colorSecondary+"]name:[-]") {
		t.Fatalf("label not formatted, got %q", got)
	}
	if !strings.Contains(got, "["+colorPrimary+"]alpha[-]\n") {
		t.Fatalf("value not colored, got %q", got)
	}
	// The visible width (tview color tags stripped) of the label half must be
	// exactly detailLabelWidth, so every value starts at the same column.
	labelPart := got[:strings.Index(got, "["+colorPrimary+"]")]
	if w := tview.TaggedStringWidth(labelPart); w != detailLabelWidth {
		t.Fatalf("label width = %d, want %d (got %q)", w, detailLabelWidth, labelPart)
	}
}

func TestPadTagged(t *testing.T) {
	// Visible width 5 padded to detailLabelWidth; color tags survive untouched.
	tagged := "[#fff]hi[-]"
	got := padTagged(tagged, 10)
	if w := tview.TaggedStringWidth(got); w != 10 {
		t.Fatalf("padded width = %d, want 10", w)
	}
	if !strings.HasPrefix(got, tagged) {
		t.Fatalf("original tagged string not preserved, got %q", got)
	}
	// No padding when already at/over width.
	if g := padTagged("already-long", 5); g != "already-long" {
		t.Fatalf("expected no padding, got %q", g)
	}
}

func TestRenderTagChips(t *testing.T) {
	// Empty input returns "".
	if got := renderTagChips(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	got := renderTagChips([]string{"a", "b"})
	if !strings.Contains(got, "[black:"+colorAccent+"] a [-:-:-]") {
		t.Fatalf("missing chip a, got %q", got)
	}
	if !strings.Contains(got, "[black:"+colorAccent+"] b [-:-:-]") {
		t.Fatalf("missing chip b, got %q", got)
	}
	if !strings.Contains(got, " ") {
		t.Fatalf("chips should be joined with space, got %q", got)
	}
}

func TestRenderTagBadgesForList(t *testing.T) {
	// Empty input returns "" so an untagged list row keeps a clean tail.
	if got := renderTagBadgesForList(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	// Up to two tags render in full, with no overflow marker.
	two := renderTagBadgesForList([]string{"a", "b"})
	if !strings.Contains(two, "[black:"+colorAccent+"] a [-:-:-]") {
		t.Fatalf("missing first badge, got %q", two)
	}
	if !strings.Contains(two, "[black:"+colorAccent+"] b [-:-:-]") {
		t.Fatalf("missing second badge, got %q", two)
	}
	if strings.Contains(two, "+") {
		t.Fatalf("no overflow marker expected for 2 tags, got %q", two)
	}
	// Three tags truncate to two plus a dim "+1"; the third chip must be dropped.
	three := renderTagBadgesForList([]string{"a", "b", "c"})
	if !strings.Contains(three, "["+colorDim+"]+1[-]") {
		t.Fatalf("missing +1 overflow marker, got %q", three)
	}
	if strings.Contains(three, "[black:"+colorAccent+"] c ") {
		t.Fatalf("third tag must be truncated, got %q", three)
	}
	// Four tags truncate to two plus a dim "+2".
	four := renderTagBadgesForList([]string{"a", "b", "c", "d"})
	if !strings.Contains(four, "["+colorDim+"]+2[-]") {
		t.Fatalf("missing +2 overflow marker, got %q", four)
	}
}

func TestSplitCSV(t *testing.T) {
	// Empty string returns nil (no tags), not a one-element slice.
	if got := splitCSV(""); got != nil {
		t.Fatalf("expected nil for empty input, got %v", got)
	}
	got := splitCSV("x,y,z")
	want := []string{"x", "y", "z"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("idx %d = %q, want %q", i, got[i], want[i])
		}
	}
}
