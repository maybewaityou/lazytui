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
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// FieldKind selects how a Field's value is rendered inside a Section.
type FieldKind int

const (
	// FieldText renders the value as a single detailField line (label:value).
	FieldText FieldKind = iota
	// FieldTag renders the CSV value as black-on-accent tag chips.
	FieldTag
	// FieldNote behaves like FieldText; reserved for callers that want to
	// distinguish note values semantically when building Sections.
	FieldNote
)

// Field is one label/value pair in a Section.
type Field struct {
	Label string
	Value string
	Kind  FieldKind
}

// Section groups titled fields for RenderSections.
type Section struct {
	Title  string
	Fields []Field
}

// detailLabelWidth is the visible width every detail label is padded to (the
// 2-space indent plus label and colon), so all values start at the same column.
// Sized comfortably above the longest lazytui label (e.g. "  created:") — the
// value is locked by render_test.go, so changing it here will fail the width
// assertion there. tview color tags occupy bytes but no visible width, so
// padding must use tview.TaggedStringWidth, not fmt's %-N. (The original
// lazytmux sized this to its "  last attached:" field; that tmux concept does
// not exist in lazytui, but the wider column is retained for visual
// consistency across the toolchain and to keep the snapshot test stable.)
const detailLabelWidth = 16

// RenderSections renders grouped fields with a colored title per section and
// aligned label/value pairs. Tags render as chips (bypassing detailField to
// avoid nested color tags); notes render as a wrapped label-value.
func RenderSections(sections []Section) string {
	var b strings.Builder
	for i, sc := range sections {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[" + colorTitle + "::b]" + sc.Title + "[-]\n")
		for _, f := range sc.Fields {
			switch f.Kind {
			case FieldTag:
				b.WriteString(padTagged("  ["+colorSecondary+"]"+f.Label+":[-]", detailLabelWidth) + renderTagChips(splitCSV(f.Value)) + "\n")
			default:
				b.WriteString(detailField(f.Label, f.Value))
			}
		}
	}
	return b.String()
}

// splitCSV splits a comma-separated value into tags. Empty input yields nil so
// FieldTag lines with no value render no chips.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// detailField renders one "  label: value" line with the label in
// colorSecondary and the value in colorPrimary, label padded to
// detailLabelWidth. Callers rendering values that carry their own color tags
// (e.g. tag chips) must NOT use this helper — see RenderSections's FieldTag
// case. (Verbatim port from lazytmux session_details.go.)
func detailField(label, value string) string {
	return padTagged("  ["+colorSecondary+"]"+label+":[-]", detailLabelWidth) + "[" + colorPrimary + "]" + value + "[-]\n"
}

// padTagged right-pads s with spaces to screen width w, preserving tview style
// tags (they don't count toward width, so colors survive the padding).
// (Verbatim port from lazytmux help.go.)
func padTagged(s string, w int) string {
	if pad := w - tview.TaggedStringWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// tagChip renders one tag as a black-on-accent pill. The trailing [-:-:-]
// resets foreground/background/style, so callers must not wrap the result in
// another color tag — that inner reset would clash with an outer wrap.
// (Verbatim port from lazytmux utils.go.)
func tagChip(t string) string {
	return fmt.Sprintf("[black:%s] %s [-:-:-]", colorAccent, t)
}

// renderTagChips renders every tag as a pill for the details pane (no
// truncation). (Verbatim port from lazytmux utils.go.)
func renderTagChips(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	chips := make([]string, 0, len(tags))
	for _, t := range tags {
		chips = append(chips, tagChip(t))
	}
	return strings.Join(chips, " ")
}

// renderTagBadgesForList renders at most two tag chips for a list row (space is
// tight) and collapses any overflow into a dim "+N" marker, matching the
// lazyssh/lazytmux list style. Use this for list rows; reach for renderTagChips
// in the details pane where there is room to show every tag. Returns "" when
// there are no tags so the row tail stays clean. (Ported from lazytmux utils.go.)
func renderTagBadgesForList(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	const maxTags = 2
	shown := tags
	if len(tags) > maxTags {
		shown = tags[:maxTags]
	}
	parts := make([]string, 0, len(shown)+1)
	for _, t := range shown {
		parts = append(parts, tagChip(t))
	}
	if extra := len(tags) - len(shown); extra > 0 {
		parts = append(parts, fmt.Sprintf("[%s]+%d[-]", colorDim, extra))
	}
	return strings.Join(parts, " ")
}
